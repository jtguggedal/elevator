package order_handler

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"./../driver"
	"./../backup"
	"./../fsm"
	"./../network"
	"./../network/peers"
)

const (
	directionUp   = driver.ButtonExternalUp
	directionDown = driver.ButtonExternalDown
	directionNone = 3
)

const (
	orderInternal = 1
	orderExternal = 2
)

const (
	costIdle             = 2
	costMovingSameDir    = 1
	costMovingOpositeDir = 10
	costPerFloor         = 3
	costPerOrderDepth    = 10
	costStuck			 = 10000
)

const orderResendInterval = 1000 * time.Millisecond

type orderDirection int
type orderType int

type Order struct {
	Id         int
	Type       orderType
	Origin     network.Ip
	Floor      int
	Direction  orderDirection
	AssignedTo network.Ip
	Done       bool
}

type orderList []Order

var localId network.Ip
var localElevatorData fsm.ElevatorData_t

func Init(orderRx <-chan network.UDPmessage,
	orderTx chan<- network.UDPmessage,
	orderDoneRxChannel <-chan network.UDPmessage,
	orderDoneTxChannel chan<- network.UDPmessage,
	buttonEventChannel <-chan driver.ButtonEvent,
	targetFloorChannel chan<- int,
	floorCompletedChannel <-chan int,
	stateRxChannel <-chan network.UDPmessage,
	peerUpdateChannel <-chan peers.PeerUpdate,
	resendStateChannel chan<- bool) {

	var externalOrders orderList
	var internalOrders orderList
	completedOrders := make(map[int]Order)
	var allElevatorStates []fsm.ElevatorData_t
	var singleElevator bool = true
	handlingOrder := false
	var activeOrder Order
	activeOrder.Floor = -1
	heading := directionNone


	distributeOrderChannel := make(chan Order)

	localId = network.GetLocalId()
	localElevatorData = fsm.GetElevatorData()

	go buttonEventListener(buttonEventChannel,
		distributeOrderChannel)


	// Import remaining queue from backup file
	backup.ReadFromFile(&internalOrders)
	if len(internalOrders) > 0 {
		fmt.Println("Imported orders from backup:", internalOrders)
		for _, order := range internalOrders {
			if order.Floor >= 0 {
				driver.SetButtonLamp(driver.ButtonInternalOrder, order.Floor, 1)
			}
		}
		activeOrder = getNextOrder(internalOrders, heading)
		heading = getHeading(activeOrder)
		targetFloorChannel <- activeOrder.Floor
	}

	// Goroutine that continuosly resends external orders until they are done
	/*go func() {
		for {
			if len(externalOrders) > 0 {
				for _, order := range externalOrders {
					if order.Origin == localId && order.Done != true {
						orderJson, _ := json.Marshal(order)
						orderTx <- network.UDPmessage{Type: network.MsgNewOrder, Data: orderJson}
					}
				}
			}
			time.Sleep(orderResendInterval)
		}
	}()*/

	/*go func() {
		for {
			select {
			case <- elevatorStateTimer:
				e := fms.GetElevatorData()
				if e.State == fsm.Stuck {

				}

			}
		}
	}()*/

	for {
		select {
		case msg := <-orderRx:

			// Order received via UDP
			var receivedOrder Order
			err := json.Unmarshal(msg.Data, &receivedOrder)
			if err == nil {
				if !orderExists(externalOrders, receivedOrder) {
					driver.SetButtonLamp(driver.ButtonType(receivedOrder.Direction), receivedOrder.Floor, 1)
					receivedOrder.AssignedTo = assignOrder(allElevatorStates, receivedOrder)
					externalOrders = addOrder(externalOrders, receivedOrder)
					fmt.Println("External orders updated:", externalOrders)
					fmt.Printf("Order assigned to %s:\tFloor: %d\n", receivedOrder.AssignedTo, receivedOrder.Floor)
					if receivedOrder.AssignedTo == localId {
						if !handlingOrder {
							activeOrder = getNextOrder(externalOrders, heading)
							handlingOrder = true
							heading = getHeading(activeOrder)
							targetFloorChannel <- activeOrder.Floor
						}
					}
				} else {
					//fmt.Println("External order already exists", receivedOrder)
				}
			} else {
				fmt.Println("External order message wrongly formatted:", err)
			}

		case order := <-distributeOrderChannel:
			switch order.Type {

			case orderInternal:
				if !orderExists(internalOrders, order) {
					if int(activeOrder.Direction) == getHeading(order) || activeOrder.Floor == -1 {
						internalOrders = addOrder(internalOrders, order)
						backup.SaveToFile(internalOrders)
						fmt.Println(isOrderOnTheWay(order, activeOrder))
						if activeOrder.Floor == -1 || isOrderOnTheWay(order, activeOrder) {
							activeOrder = order
							heading = getHeading(activeOrder)
							targetFloorChannel <- activeOrder.Floor
						}
					} else {
						driver.SetButtonLamp(driver.ButtonInternalOrder, order.Floor, 0)
					}
				} else {
					fmt.Println("Internal order already exists", order)
				}

			case orderExternal:
				order.Origin = localId
				if singleElevator {
					order.AssignedTo = localId
					externalOrders = addOrder(externalOrders, order)
					if !handlingOrder || isOrderOnTheWay(order, activeOrder) {
						handlingOrder = true
						activeOrder = order
						heading = getHeading(activeOrder)
						targetFloorChannel <- activeOrder.Floor
					} else {
						// No action, order is queued
					}
				} else {
					orderJson, _ := json.Marshal(order)
					go func() {
						for i := 0; i < 20; i++{
							orderTx <- network.UDPmessage{Type: network.MsgNewOrder, Data: orderJson}
							time.Sleep(50 * time.Millisecond)
						}
					}()
				}
			}

		case floor := <-floorCompletedChannel:
			if floor == activeOrder.Floor {
				activeOrder.Done = true
				if activeOrder.Type == orderInternal {
					internalOrders = orderCompleted(internalOrders, activeOrder)
					backup.SaveToFile(internalOrders)
				} else if activeOrder.Type == orderExternal {
					externalOrders = orderCompleted(externalOrders, activeOrder)
					if !singleElevator {
						order := activeOrder
						// Goroutine that sends "order completed" message repeatedly for redundancy
						go func() {
							orderJson, _ := json.Marshal(order)
							for i := 0; i < 30; i++ {
								orderDoneTxChannel <- network.UDPmessage{Type: network.MsgFinishedOrder, Data: orderJson}
								time.Sleep(200 * time.Millisecond)
							}
						}()
					}
				}
				if len(internalOrders) > 0 {
					fmt.Println("Remaining internal orders", internalOrders)
					activeOrder = getNextOrder(internalOrders, heading)
				} else {
					fmt.Println("Remaining external orders", externalOrders)
					activeOrder = getNextOrder(externalOrders, heading)
					if activeOrder.Floor == -1 {
						handlingOrder = false
					}
				}
			}
			if activeOrder.Floor != -1 {
				fmt.Println("Next order", activeOrder)
				heading = getHeading(activeOrder)
				handlingOrder = true
				targetFloorChannel <- activeOrder.Floor
			} else {
				handlingOrder = false
			}

		case stateJson := <-stateRxChannel:
			var state fsm.ElevatorData_t
			jsonToStruct(stateJson.Data, &state)
			allElevatorStates = updatePeerState(allElevatorStates, state)

		case peers := <-peerUpdateChannel:
			var newPeerFlag, lostPeerFlag bool
			allElevatorStates, singleElevator, newPeerFlag, lostPeerFlag = updateLivePeers(peers, allElevatorStates)
			if newPeerFlag {
				resendStateChannel <- true
			}
			if newPeerFlag || lostPeerFlag {
				externalOrders = reassignOrders(externalOrders, allElevatorStates)
				candidateOrder := getNextOrder(externalOrders, heading)
				if (candidateOrder.Floor != -1) && (candidateOrder.AssignedTo == localId) && !handlingOrder {
					activeOrder = candidateOrder
					heading = getHeading(activeOrder)
					handlingOrder = true
					targetFloorChannel <- activeOrder.Floor
				} else {
					//
				}
			}

		case orderMsg := <-orderDoneRxChannel:
			var order Order
			jsonToStruct(orderMsg.Data, &order)
			if _, ok := completedOrders[order.Id]; !ok {
				completedOrders[order.Id] = order
				externalOrders = orderCompleted(externalOrders, order)
				fmt.Println("DONE", order)
			}
		}
	}
}

func buttonEventListener(buttonEventChannel <-chan driver.ButtonEvent,
	distributeOrderChannel chan<- Order) {
	for {
		select {
		case buttonEvent := <-buttonEventChannel:
			driver.SetButtonLamp(buttonEvent.Type, buttonEvent.Floor, 1)

			switch buttonEvent.Type {
			case driver.ButtonExternalUp:
				fmt.Println("External button event: UP from floor", buttonEvent.Floor)
				distributeOrderChannel <- Order{Id: getOrderId(),
					Type:      orderExternal,
					Floor:     buttonEvent.Floor,
					Direction: directionUp}

			case driver.ButtonExternalDown:
				fmt.Println("External button event: DOWN from floor", buttonEvent.Floor)
				distributeOrderChannel <- Order{Id: getOrderId(),
					Type:      orderExternal,
					Floor:     buttonEvent.Floor,
					Direction: directionDown}

			case driver.ButtonInternalOrder:
				fmt.Println("Internal button event: go to floor", buttonEvent.Floor)
				distributeOrderChannel <- Order{Id: getOrderId(),
					Type:       orderInternal,
					Floor:      buttonEvent.Floor,
					AssignedTo: localId}
			}
		}
	}
}

func orderCost(orders orderList, o Order, e fsm.ElevatorData_t) int {
	var cost int
	distance := o.Floor - e.Floor

	idle := e.State == fsm.Idle
	movingSameDir := (int(o.Direction) == int(e.State)) && !idle
	movingOpositeDir := !movingSameDir && !idle
	stuck := e.State == fsm.Stuck

	targetAbove := distance > 0
	targetBelow := distance < 0
	distance = int(math.Abs(float64(distance)))

	if targetAbove || targetBelow {
		cost += distance * costPerFloor
	}
	if stuck {
		cost += costStuck
	}
	if movingSameDir {
		cost += costMovingSameDir
	} else if movingOpositeDir {
		cost += costMovingOpositeDir
	} else if idle {
		cost += costIdle
	}

	if o.Type == orderExternal {
		orderDepth := 0
		for _, ord := range orders {
			if ord.AssignedTo == e.Id {
				orderDepth += 1
			}
		}
		cost += orderDepth * costPerOrderDepth
	}
	fmt.Println("Order cost:", cost, o)
	return cost
}

func assignOrder(allElevatorStates []fsm.ElevatorData_t, order Order) network.Ip {
	cost := 0
	lowestCost := 100000
	var assignTo network.Ip
	order.AssignedTo = ""
	for _, elevator := range allElevatorStates {
		cost = orderCost(orderList{}, order, elevator)
		fmt.Printf("Cost for %s: %d\n", elevator.Id, cost)
		if cost < lowestCost {
			lowestCost = cost
			assignTo = elevator.Id
		} else if cost == lowestCost && elevator.Id > assignTo {
			assignTo = elevator.Id
		}
	}
	return assignTo
}

func reassignOrders(orders orderList, allElevatorStates []fsm.ElevatorData_t) orderList {
	fmt.Println("Incoming for reassignmnet", orders, allElevatorStates)
	for i, order := range orders {
		orders[i].AssignedTo = assignOrder(allElevatorStates, order)
		fmt.Println("Reassigned to elevator", orders[i].AssignedTo)
	}
	fmt.Println("Orders reassigned", orders)
	return orders
}

func updateLivePeers(	peers peers.PeerUpdate,
						allElevatorStates []fsm.ElevatorData_t) ([]fsm.ElevatorData_t, bool, bool, bool) {
	newPeer := len(peers.New) > 0
	lostPeer := len(peers.Lost) > 0
	var singleElevator bool
	if network.IsConnected() {
		for i, storedPeer := range allElevatorStates {
			for _, lostPeer := range peers.Lost {
				if network.Ip(lostPeer) == storedPeer.Id && len(allElevatorStates) > i && network.Ip(lostPeer) != localId {
					fmt.Println("Lost peers", peers.Lost)
					allElevatorStates = append(allElevatorStates[:i], allElevatorStates[i+1:]...)
				}
			}
		}
		fmt.Println("Connected peers:", peers.Peers, allElevatorStates)
		singleElevator = len(peers.Peers) == 1
	} else {
		fmt.Println("Disconnected from network. Finishing internal queue.")
		allElevatorStates = allElevatorStates[:0]
		allElevatorStates = append(allElevatorStates, localElevatorData)
		singleElevator = true
	}
	return allElevatorStates, singleElevator, newPeer, lostPeer
}

func updatePeerState(allElevatorStates []fsm.ElevatorData_t,
	state fsm.ElevatorData_t) []fsm.ElevatorData_t {
	var stateExists bool
	for key, data := range allElevatorStates {
		if state.Id == data.Id {
			stateExists = true
			allElevatorStates[key] = state
		}
	}
	if !stateExists {
		allElevatorStates = append(allElevatorStates, state)
	}
	return allElevatorStates
}

func getHeading(order Order) int {
	currentFloor := fsm.GetElevatorData().Floor
	diff := currentFloor - order.Floor
	if diff == 0 {
		return directionNone
	} else if diff < 0 {
		return directionUp
	}
	return directionDown
}

func getNextOrder(orders orderList, heading int) Order {
	if len(orders) == 0 {
		return Order{Floor: -1}
	}
	oldestEntryTime := int(10e15)
	var nextOrder Order
	var internal bool
	nextOrder.Floor = -1
	lowestCost := 1000000
	elevatorData := fsm.GetElevatorData()

	// Serving internal orders first
	for _, order := range orders {
		if order.Type == orderInternal {
			cost := orderCost(orders, order, elevatorData)
			if cost < lowestCost {
				lowestCost = cost
				nextOrder = order
				internal = true
			}
		}
	}
	if internal {
		return nextOrder
	}

	for _, order := range orders {
		if order.Id < oldestEntryTime && order.AssignedTo == localId {
			oldestEntryTime = order.Id
			nextOrder = order
		}
	}
	for _, order := range orders {
		if isOrderOnTheWay(order, nextOrder) && order.AssignedTo == localId {
			nextOrder = order
		}
	}
	return nextOrder
}

func isOrderOnTheWay(order, activeOrder Order) bool {
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

func addOrder(orders []Order, newOrder Order) []Order {
	if !orderExists(orders, newOrder) {
		orders = append(orders, newOrder)
	}
	return orders
}

func orderCompleted(orders orderList, order Order) orderList {
	if order.Type == orderInternal {
		driver.SetButtonLamp(driver.ButtonInternalOrder, order.Floor, 0)
	} else {
		driver.SetButtonLamp(driver.ButtonType(order.Direction), order.Floor, 0)
	}
	return removeDoneOrders(orders, order)
}

func removeDoneOrders(orders orderList, doneOrder Order) orderList {
	for key, order := range orders {
		if ordersEqual(order, doneOrder) && doneOrder.Done == true {
			orders = append(orders[:key], orders[key+1:]...)
		}
	}
	return orders
}

func getOrderId() int {
	return int(time.Now().UnixNano()/1e8 - 1488*1e7)
}

func jsonToStruct(input []byte, output interface{}) {
	temp := output
	err := json.Unmarshal(input, temp)
	if err == nil {
		output = temp
	} else {
		fmt.Println("Error decoding JSON:", err)
	}
}

func orderToJson(id int, orderType orderType, floor int, direction orderDirection) []byte {
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
