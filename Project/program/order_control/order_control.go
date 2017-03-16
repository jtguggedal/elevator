package order_control

import (
	"encoding/json"
	"fmt"
	"time"
	"math"

	"./../driver"
	"./../backup"
	"./../fsm"
	"./../network"
	"./../network/peers"
)

const (
	DirectionUp   = driver.ButtonExternalUp
	DirectionDown = driver.ButtonExternalDown
	DirectionNone = 3
)

const (
	OrderInternal = 1
	OrderExternal = 2
)

const (
	Idle             = 2
	MovingSameDir    = 1
	MovingOpositeDir = 10
	PerFloor         = 3
	PerOrderDepth    = 10
	Stuck			 = 10000
)

const orderTimeout = 24 * time.Second

type OrderDirection int
type OrderType int

type Order struct {
	Id         int
	Type       OrderType
	Origin     string
	Floor      int
	Direction  OrderDirection
	AssignedTo string
	Done       bool
}

type OrderList []Order

var localId string
var localElevatorData fsm.ElevatorData_t

// The main function in the order handler module and the only one available externally
func Init(	orderChannels network.ChannelPair,
			orderDoneChannels network.ChannelPair,
			buttonEventChan <-chan driver.ButtonEvent,
			targetFloorChan chan<- int,
			floorCompletedChan <-chan int,
			stateRxChan <-chan network.UDPmessage,
			peerUpdateChan <-chan peers.PeerUpdate) {

	var externalOrders OrderList
	var internalOrders OrderList
	completedOrders := make(map[int]Order)
	var allElevatorStates []fsm.ElevatorData_t
	var singleElevator bool = true
	handlingOrder := false
	var activeOrder Order
	heading := DirectionNone

	distributeOrderChan := make(chan Order)

	localId = network.GetLocalId()
	localElevatorData = fsm.GetElevatorData()
	localElevatorData.Id = localId
	activeOrder.Floor = -1

	// Event listener to register new button events and translate to order
	go buttonEventListener(buttonEventChan,	distributeOrderChan)

	// Import possibly remaining queue from backup file
	backup.ReadFromFile(&internalOrders)
	if len(internalOrders) > 0 {
		fmt.Println("Imported orders from backup:", internalOrders)
		for _, order := range internalOrders {
			if order.Floor >= 0 {
				driver.SetButtonLamp(driver.ButtonInternalOrder, order.Floor, 1)
			}
		}
		activeOrder, _ = getNextOrder(internalOrders, localId)
		heading = getElevatorHeading(activeOrder)
		targetFloorChan <- activeOrder.Floor
	}

	go func() {
		for{
			select{
			case <-time.After(orderTimeout):
				for _, order := range externalOrders {
					if order.Id + 10000 < getOrderId() {
						externalOrders = reassignOrders(externalOrders, allElevatorStates)
						candidateOrder, _ := getNextOrder(externalOrders, localId)
						if (candidateOrder.Floor != -1) && (candidateOrder.AssignedTo == localId) && !handlingOrder {
							activeOrder = candidateOrder
							heading = getElevatorHeading(activeOrder)
							handlingOrder = true
							targetFloorChan <- activeOrder.Floor
						}
					}
				}
			}
		}
		}()

	// Main loop for order handling
	go func() {
		for {
			select {
			case msg := <-orderChannels.Rx:

				// Order received as JSON via UDP and stored to local queue
				var receivedOrder Order
				err := json.Unmarshal(msg.Data, &receivedOrder)
				if err == nil {
					if !orderExists(externalOrders, receivedOrder) && receivedOrder.Done != true {
						driver.SetButtonLamp(driver.ButtonType(receivedOrder.Direction), receivedOrder.Floor, 1)

						// Calculating the received order cost for all connected elevators and assigning accordingly
						receivedOrder.AssignedTo = assignOrder(allElevatorStates, receivedOrder)
						externalOrders = addOrder(externalOrders, receivedOrder)

						// Execute order immediately if it is assigned to this elevator
						if receivedOrder.AssignedTo == localId && !handlingOrder {
							activeOrder, handlingOrder = getNextOrder(externalOrders, localId)
							heading = getElevatorHeading(activeOrder)
							targetFloorChan <- activeOrder.Floor
						}
					}
				}

			// Newly created orders received from button event listener. Distributed according to type and elevator status.
			case order := <-distributeOrderChan:
				switch order.Type {
				case OrderInternal:
					if !orderExists(internalOrders, order) {

						// 	Only internal orders for the same direction as the hall call indicated is taken
						if int(activeOrder.Direction) == getElevatorHeading(order) || activeOrder.Floor == -1 {
							internalOrders = addOrder(internalOrders, order)

							// Sets received order floor as target only if it is on the way to current target or if idle
							if localElevatorData.State == fsm.Idle || takeOrderOnTheWay(order, activeOrder) {
								activeOrder = order
								heading = getElevatorHeading(activeOrder)
								targetFloorChan <- activeOrder.Floor
							}
						} else {
							driver.SetButtonLamp(driver.ButtonInternalOrder, order.Floor, 0)
						}
					}

				case OrderExternal:
					order.Origin = localId

					// If elevator i in single mode, order is only added locally
					if singleElevator {
						order.AssignedTo = localId
						externalOrders = addOrder(externalOrders, order)
						if !handlingOrder || takeOrderOnTheWay(order, activeOrder) {
							handlingOrder = true
							activeOrder = order
							heading = getElevatorHeading(activeOrder)
							targetFloorChan <- activeOrder.Floor
						}
					} else {
						// If multiple elevators, the order is broadcasted to all
					}

					broadcastOrder(order, orderChannels.Tx)
				}

			case floor := <-floorCompletedChan:

				// If the floor was the target floor in the active order, the order is considered completed
				if floor == activeOrder.Floor {
					activeOrder.Done = true
					if activeOrder.Type == OrderInternal {
						internalOrders = orderCompleted(internalOrders, activeOrder, orderDoneChannels.Tx)
					} else if activeOrder.Type == OrderExternal {
						externalOrders = orderCompleted(externalOrders, activeOrder, orderDoneChannels.Tx)
					}

					// Internal orders originating from an external order are handled before next external order is considered
					if len(internalOrders) > 0 {
						fmt.Println("Remaining internal orders", internalOrders)
						activeOrder, _ = getNextOrder(internalOrders, localId)
					} else {
						fmt.Println("Remaining external orders", externalOrders)
						activeOrder, _ = getNextOrder(externalOrders, localId)
						if activeOrder.Floor == -1 {
							handlingOrder = false
						}
					}
				}

				// If a new order is scheduled, it is executed. If not, elevator waits for next incoming order.
				if activeOrder.Floor != -1 {
					fmt.Println("Active order:", activeOrder)
					heading = getElevatorHeading(activeOrder)
					handlingOrder = true
					targetFloorChan <- activeOrder.Floor
				} else {
					handlingOrder = false
				}

			// Receiving message about completed orders from other elevators
			case orderMsg := <-orderDoneChannels.Rx:
				var order Order
				jsonToOrder(orderMsg.Data, &order)
				if _, ok := completedOrders[order.Id]; !ok {
					completedOrders[order.Id] = order
					externalOrders = orderCompleted(externalOrders, order, orderDoneChannels.Tx)
				}
			}
		}
	}()

	// Goroutine to keep track of all elevator states for use in cost calculation
	go func() {
		for {
			select {
			case stateJson := <-stateRxChan:
				var state fsm.ElevatorData_t
				jsonToOrder(stateJson.Data, &state)
				allElevatorStates = fsm.UpdatePeerState(allElevatorStates, state)
				//fmt.Println("UPDATED, all states:", allElevatorStates)

			case peers := <-peerUpdateChan:
				var newPeerFlag, lostPeerFlag bool
				allElevatorStates, singleElevator, newPeerFlag, lostPeerFlag = updateLivePeerList(peers, allElevatorStates)
				if newPeerFlag || lostPeerFlag {
					externalOrders = reassignOrders(externalOrders, allElevatorStates)
					candidateOrder, _ := getNextOrder(externalOrders, localId)
					if (candidateOrder.Floor != -1) && (candidateOrder.AssignedTo == localId) && !handlingOrder {
						activeOrder = candidateOrder
						heading = getElevatorHeading(activeOrder)
						handlingOrder = true
						targetFloorChan <- activeOrder.Floor
					}
				}
			}
		}
	}()
}

// Function to be run as goroutine to listen to button events and translate them to elevator orders
func buttonEventListener(buttonEventChan <-chan driver.ButtonEvent,
	distributeOrderChan chan<- Order) {
	for {
		select {
		case buttonEvent := <-buttonEventChan:
			driver.SetButtonLamp(buttonEvent.Type, buttonEvent.Floor, 1)

			switch buttonEvent.Type {
			case driver.ButtonExternalUp:
				fmt.Println("External button event: UP from floor", buttonEvent.Floor)
				distributeOrderChan <- Order{Id: getOrderId(),
					Type:      OrderExternal,
					Floor:     buttonEvent.Floor,
					Direction: DirectionUp}

			case driver.ButtonExternalDown:
				fmt.Println("External button event: DOWN from floor", buttonEvent.Floor)
				distributeOrderChan <- Order{Id: getOrderId(),
					Type:      OrderExternal,
					Floor:     buttonEvent.Floor,
					Direction: DirectionDown}

			case driver.ButtonInternalOrder:
				fmt.Println("Internal button event: go to floor", buttonEvent.Floor)
				distributeOrderChan <- Order{Id: getOrderId(),
					Type:       OrderInternal,
					Floor:      buttonEvent.Floor,
					AssignedTo: localId}
			}
		}
	}
}

// Cost function for elevator orders
func getOrderCost(orders OrderList, order Order, elevatorData fsm.ElevatorData_t) int {
	var orderCost int
	distance := order.Floor - elevatorData.Floor

	idle := elevatorData.State == fsm.Idle
	movingSameDir := (int(order.Direction) == int(elevatorData.State)) && !idle
	movingOpositeDir := !movingSameDir && !idle
	stuck := elevatorData.State == fsm.Stuck

	targetAbove := distance > 0
	targetBelow := distance < 0
	distance = int(math.Abs(float64(distance)))

	if targetAbove || targetBelow {
		orderCost += distance * PerFloor
	}
	if stuck {
		orderCost += Stuck
	}
	if movingSameDir {
		orderCost += MovingSameDir
	} else if movingOpositeDir {
		orderCost += MovingOpositeDir
	} else if idle {
		orderCost += Idle
	}

	if order.Type == OrderExternal {
		orderDepth := 0
		for _, ord := range orders {
			if ord.AssignedTo == elevatorData.Id {
				orderDepth += 1
			}
		}
		orderCost += orderDepth * PerOrderDepth
	}
	//fmt.Println("Order cost:", orderCost, o)
	return orderCost
}

/**  Cost calculation functions  **/

// Function that, based on cost function, returns the next order to take fram an order list
func getNextOrder(orders OrderList, localId string) (Order, bool) {
	if len(orders) == 0 {
		return Order{Floor: -1}, false
	}
	oldestEntryTime := int(10e15)
	var nextOrder Order
	nextOrder.Floor = -1
	lowestCost := 1000000
	elevatorData := fsm.GetElevatorData()

	// Serving internal orders first
	if orders[0].Type == OrderInternal {
		for _, order := range orders {
			cost := getOrderCost(orders, order, elevatorData)
			if cost < lowestCost {
				lowestCost = cost
				nextOrder = order
			}
		}
		return nextOrder, nextOrder.Floor != -1
	}
	for _, order := range orders {
		if order.Id < oldestEntryTime && string(order.AssignedTo) == localId {
			oldestEntryTime = order.Id
			nextOrder = order
		}
	}
	for _, order := range orders {
		if takeOrderOnTheWay(order, nextOrder) && string(order.AssignedTo) == localId {
			nextOrder = order
		}
	}
	return nextOrder, nextOrder.Floor != -1
}

// Function that returns true if a given order is on the way from current elevator location to target floor
func takeOrderOnTheWay(order, activeOrder Order) bool {
	elevatorState := fsm.GetElevatorData()
	if activeOrder.Direction == order.Direction {
		if (elevatorState.Floor > order.Floor) && (order.Floor > activeOrder.Floor) {
			return true
		} else if elevatorState.Floor < order.Floor && order.Floor < activeOrder.Floor {
			return true
		}
	}
	return false
}


// ** Helper functions	**//

func broadcastOrder(order Order, orderTxChan chan<- network.UDPmessage) {
	go func() {
		orderJson, _ := json.Marshal(order)
		// Repeatedly sends new order to ensure it is not lost
		for i := 0; i < 20; i++ {
			orderTxChan <- network.UDPmessage{Type: network.MsgNewOrder, Data: orderJson}
			time.Sleep(100 * time.Millisecond)
		}
	}()
}


// Function to calculate which elevator should be assigned to an order and returns ID of that elevator
func assignOrder(allElevatorStates []fsm.ElevatorData_t, order Order) string {
	orderCost := 0
	lowestCost := 100000
	var assignTo string
	order.AssignedTo = ""
	fmt.Println("RECEIVED for cost calculation:", allElevatorStates)
	for _, elevator := range allElevatorStates {
		orderCost = getOrderCost(OrderList{}, order, elevator)
		fmt.Printf("Cost for %s: %d\n", elevator.Id, orderCost)
		if orderCost < lowestCost {
			lowestCost = orderCost
			assignTo = elevator.Id
		} else if orderCost == lowestCost && elevator.Id > assignTo {
			assignTo = elevator.Id
		}
	}
	return assignTo
}

// Function that recalculates all order assignments and returns order list
func reassignOrders(orders OrderList, allElevatorStates []fsm.ElevatorData_t) OrderList {
	for i, order := range orders {
		orders[i].AssignedTo = assignOrder(allElevatorStates, order)
	}
	fmt.Println("Orders have been reassigned")
	return orders
}

// Function that updates which elevators are currently connected to the network. Returns array of elevator states.
func updateLivePeerList (peers peers.PeerUpdate,
						allElevatorStates []fsm.ElevatorData_t) ([]fsm.ElevatorData_t, bool, bool, bool) {
	newPeerFlag := len(peers.New) > 0
	lostPeerFlag := len(peers.Lost) > 0
	var singleElevator bool
	fmt.Println("Connected peers:", peers.Peers)

	// Check if elevator is still on the network, and if so, parse received peer updates
	if network.IsConnected() {
		//if lostPeerFlag {
			for i, storedPeer := range allElevatorStates {
				found := false
				for _, peer := range peers.Peers {
					if 	peer == storedPeer.Id {
						found = true
					}
				}
				if !found {
					fmt.Println("Lost peer", peers.Lost)
					allElevatorStates = append(allElevatorStates[:i], allElevatorStates[i+1:]...)
					//fmt.Println("LOST, all states:", allElevatorStates)
				}
			}
		//}
		singleElevator = len(peers.Peers) == 1 && peers.Peers[0] == localId
	} else {
		fmt.Println("Disconneted from network")
		allElevatorStates = nil
		allElevatorStates = append(allElevatorStates, fsm.GetElevatorData())
		fmt.Println("SINGLE, all states:", allElevatorStates)
		singleElevator = true
	}
	return allElevatorStates, singleElevator, newPeerFlag, lostPeerFlag
}


/** Helper functions for order control **/

func getElevatorHeading(order Order) int {
	currentFloor := fsm.GetElevatorData().Floor
	diff := currentFloor - order.Floor
	if diff == 0 {
		return DirectionNone
	} else if diff < 0 {
		return DirectionUp
	}
	return DirectionDown
}

// Function that asserts that order is completed and removes it from queue
func orderCompleted(orders OrderList, order Order, orderDoneTxChan chan<- network.UDPmessage) OrderList {
	orders = removeDoneOrders(orders, order)
	if order.Type == OrderInternal {
		backup.SaveToFile(orders)
		driver.SetButtonLamp(driver.ButtonInternalOrder, order.Floor, 0)
	} else if order.Type == OrderExternal {
		driver.SetButtonLamp(driver.ButtonType(order.Direction), order.Floor, 0)
		if order.AssignedTo == localId {

			// Send "order completed" message repeatedly for redundancy
			go func() {
				orderJson, _ := json.Marshal(order)
				for i := 0; i < 15; i++ {
					orderDoneTxChan <- network.UDPmessage{Type: network.MsgFinishedOrder, Data: orderJson}
					time.Sleep(200 * time.Millisecond)
				}
			}()
		}
	}
	return orders
}

func addOrder(orders []Order, newOrder Order) []Order {
	if !orderExists(orders, newOrder) {
		orders = append(orders, newOrder)
	}
	if newOrder.Type == OrderInternal {
		backup.SaveToFile(orders)
	}
	return orders
}

func removeDoneOrders(orders OrderList, doneOrder Order) OrderList {
	for key, order := range orders {
		if ordersEqual(order, doneOrder) && doneOrder.Done == true {
			orders = append(orders[:key], orders[key+1:]...)
		}
	}
	return orders
}

func getOrderId() int {
	return int(time.Now().UnixNano()/1e6)
}

func jsonToOrder(input []byte, output interface{}) {
	temp := output
	err := json.Unmarshal(input, temp)
	if err == nil {
		output = temp
	} else {
		fmt.Println("Error decoding JSON:", err)
	}
}

func orderToJson(id int, orderType OrderType, floor int, direction OrderDirection) []byte {
	ret, _ := json.Marshal(Order{
		Id:        id,
		Type:      orderType,
		Floor:     floor,
		Direction: direction})
	return ret
}

func ordersEqual(o1, o2 Order) bool {
	if o1.Type == o2.Type &&
		o1.Floor == o2.Floor &&
		o1.Direction == o2.Direction {
		return true
	}
	return false
}

func orderExists(orders []Order, candidateOrder Order) bool {
	for _, order := range orders {
		if (order.Type == candidateOrder.Type) &&
			(order.Floor == candidateOrder.Floor) &&
			(order.Direction == candidateOrder.Direction) {
			return true
		}
	}
	return false
}
