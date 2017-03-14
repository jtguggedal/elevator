package order_handler

import (
	"encoding/json"
	"fmt"
	"time"

	"./../driver"
	"./../backup"
	"./../fsm"
	"./../network"
	"./../network/peers"
	cost "./../order_cost"
	. "./../order_common"
)

var localId string
var localElevatorData fsm.ElevatorData_t

// The main function in the order handler module and the only one available externally
func Init(	orderRxChan <-chan network.UDPmessage,
			orderTxChan chan<- network.UDPmessage,
			orderDoneRxChan <-chan network.UDPmessage,
			orderDoneTxChan chan<- network.UDPmessage,
			buttonEventChan <-chan driver.ButtonEvent,
			targetFloorChan chan<- int,
			floorCompletedChan <-chan int,
			stateRxChan <-chan network.UDPmessage,
			peerUpdateChan <-chan peers.PeerUpdate,
			resendStateChan chan<- bool) {

	var externalOrders OrderList
	var internalOrders OrderList
	completedOrders := make(map[int]Order)
	var allElevatorStates []fsm.ElevatorData_t
	var singleElevator bool = true
	handlingOrder := false
	var activeOrder Order
	activeOrder.Floor = -1
	heading := DirectionNone

	distributeOrderChan := make(chan Order)

	localId = network.GetLocalId()
	localElevatorData = fsm.GetElevatorData()
	localElevatorData.Id = localId

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
		activeOrder, _ = cost.GetNextOrder(internalOrders, localId)
		heading = getHeading(activeOrder)
		targetFloorChan <- activeOrder.Floor
	}

	// Goroutine to keep track of all elevator states for use in cost calculation
	go func() {
		for {
			select {
			case stateJson := <-stateRxChan:
				var state fsm.ElevatorData_t
				JsonToStruct(stateJson.Data, &state)
				allElevatorStates = fsm.UpdatePeerState(allElevatorStates, state)

			case peers := <-peerUpdateChan:
				var newPeerFlag, lostPeerFlag bool
				allElevatorStates, singleElevator, newPeerFlag, lostPeerFlag = updateLivePeers(peers, allElevatorStates)
				if newPeerFlag {
					resendStateChan <- true
				}
				if newPeerFlag || lostPeerFlag {
					externalOrders = reassignOrders(externalOrders, allElevatorStates)
					candidateOrder, _ := cost.GetNextOrder(externalOrders, localId)
					if (candidateOrder.Floor != -1) && (candidateOrder.AssignedTo == localId) && !handlingOrder {
						activeOrder = candidateOrder
						heading = getHeading(activeOrder)
						handlingOrder = true
						targetFloorChan <- activeOrder.Floor
					}
				}
			}
		}
	}()

	// Main loop for order handling
	for {
		select {
		case msg := <-orderRxChan:

			// Order received as JSON via UDP and stored to local queue
			var receivedOrder Order
			err := json.Unmarshal(msg.Data, &receivedOrder)
			if err == nil {
				if !OrderExists(externalOrders, receivedOrder) && receivedOrder.Done != true {
					driver.SetButtonLamp(driver.ButtonType(receivedOrder.Direction), receivedOrder.Floor, 1)

					// Calculating the received order cost for all connected elevators and assigning accordingly
					receivedOrder.AssignedTo = assignOrder(allElevatorStates, receivedOrder)
					externalOrders = AddOrder(externalOrders, receivedOrder)
					fmt.Println("ASSIGNED TO:", receivedOrder.AssignedTo)

					// Execute order immediately if it is assigned to this elevator
					if receivedOrder.AssignedTo == localId && !handlingOrder {
						activeOrder, handlingOrder = cost.GetNextOrder(externalOrders, localId)
						heading = getHeading(activeOrder)
						targetFloorChan <- activeOrder.Floor
					}
				} else {
					fmt.Println("External order already exists", receivedOrder)
				}
			} else {
				fmt.Println("External order message wrongly formatted:", err)
			}

		case order := <-distributeOrderChan:
			switch order.Type {
			case OrderInternal:
				if !OrderExists(internalOrders, order) {

					// 	Only internal orders for the same direction as the hall call indicated is taken
					if int(activeOrder.Direction) == getHeading(order) || activeOrder.Floor == -1 {
						internalOrders = AddOrder(internalOrders, order)

						// Sets received order floor as target only if it is on the way to current target or if idle
						if localElevatorData.State == fsm.Idle || cost.TakeOrderOnTheWay(order, activeOrder) {
							activeOrder = order
							heading = getHeading(activeOrder)
							targetFloorChan <- activeOrder.Floor
						}
					} else {
						driver.SetButtonLamp(driver.ButtonInternalOrder, order.Floor, 0)
					}
				} else {
					fmt.Println("Internal order already exists", order)
				}

			case OrderExternal:
				order.Origin = localId
				if singleElevator {
					fmt.Println("SINGLE")
					order.AssignedTo = localId
					externalOrders = AddOrder(externalOrders, order)
					if !handlingOrder || cost.TakeOrderOnTheWay(order, activeOrder) {
						handlingOrder = true
						activeOrder = order
						heading = getHeading(activeOrder)
						targetFloorChan <- activeOrder.Floor
					}
				} else {
				}
					go func() {
						orderJson, _ := json.Marshal(order)
						// Repeatedly sends new order to ensure it is not lost
						for i := 0; i < 15; i++ {
							orderTxChan <- network.UDPmessage{Type: network.MsgNewOrder, Data: orderJson}
							time.Sleep(200 * time.Millisecond)
						}
					}()
			}

		case floor := <-floorCompletedChan:

			// If the floor was the target floor in the active order, the order is considered completed
			if floor == activeOrder.Floor {
				activeOrder.Done = true
				if activeOrder.Type == OrderInternal {
					internalOrders = orderCompleted(internalOrders, activeOrder, orderDoneTxChan)
				} else if activeOrder.Type == OrderExternal {
					externalOrders = orderCompleted(externalOrders, activeOrder, orderDoneTxChan)
				}

				// Internal orders originating from an external order are handled before next external order is considered
				if len(internalOrders) > 0 {
					fmt.Println("Remaining internal orders", internalOrders)
					activeOrder, _ = cost.GetNextOrder(internalOrders, localId)
				} else {
					fmt.Println("Remaining external orders", externalOrders)
					activeOrder, _ = cost.GetNextOrder(externalOrders, localId)
					if activeOrder.Floor == -1 {
						handlingOrder = false
					}
				}
			}

			// If a new order is scheduled, it is executed. If not, elevator waits for next incoming order.
			if activeOrder.Floor != -1 {
				fmt.Println("Active order:", activeOrder)
				heading = getHeading(activeOrder)
				handlingOrder = true
				targetFloorChan <- activeOrder.Floor
			} else {
				handlingOrder = false
			}

		// Receiving message about completed orders from other elevators
		case orderMsg := <-orderDoneRxChan:
			var order Order
			JsonToStruct(orderMsg.Data, &order)
			if _, ok := completedOrders[order.Id]; !ok {
				completedOrders[order.Id] = order
				externalOrders = orderCompleted(externalOrders, order, orderDoneTxChan)
			}
		}
	}
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
				distributeOrderChan <- Order{Id: GetOrderId(),
					Type:      OrderExternal,
					Floor:     buttonEvent.Floor,
					Direction: DirectionUp}

			case driver.ButtonExternalDown:
				fmt.Println("External button event: DOWN from floor", buttonEvent.Floor)
				distributeOrderChan <- Order{Id: GetOrderId(),
					Type:      OrderExternal,
					Floor:     buttonEvent.Floor,
					Direction: DirectionDown}

			case driver.ButtonInternalOrder:
				fmt.Println("Internal button event: go to floor", buttonEvent.Floor)
				distributeOrderChan <- Order{Id: GetOrderId(),
					Type:       OrderInternal,
					Floor:      buttonEvent.Floor,
					AssignedTo: localId}
			}
		}
	}
}


// ** Helper functions	**//


// Function to calculate which elevator should be assigned to an order and returns ID of that elevator
func assignOrder(allElevatorStates []fsm.ElevatorData_t, order Order) string {
	orderCost := 0
	lowestCost := 100000
	var assignTo string
	order.AssignedTo = ""
	for _, elevator := range allElevatorStates {
		orderCost = cost.OrderCost(OrderList{}, order, elevator)
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
	fmt.Println("Orders reassigned")
	return orders
}

// Function that updates which elevators are currently connected to the network. Returns array of elevator states.
func updateLivePeers(	peers peers.PeerUpdate,
						allElevatorStates []fsm.ElevatorData_t) ([]fsm.ElevatorData_t, bool, bool, bool) {
	newPeerFlag := len(peers.New) > 0
	lostPeerFlag := len(peers.Lost) > 0
	var singleElevator bool
	fmt.Println("Connected peers:", peers.Peers)

	// Check if elevator is still on the network, and if so, parse received peer updates
	if network.IsConnected() {
		if lostPeerFlag {
			for i, storedPeer := range allElevatorStates {
				for _, lostPeer := range peers.Lost {
					if 	lostPeer == storedPeer.Id && len(allElevatorStates) > i &&
							lostPeer != localId  && lostPeer != "" {
						fmt.Println("Lost peer", peers.Lost)
						allElevatorStates = append(allElevatorStates[:i], allElevatorStates[i+1:]...)
					}
				}
			}
		}
		singleElevator = len(peers.Peers) == 1 && peers.Peers[0] == localId
		fmt.Println("Status of single elevator:", singleElevator)
	} else {
		fmt.Println("Disconnected from network. Finishing internal queue.")
		allElevatorStates = nil
		allElevatorStates = append(allElevatorStates, fsm.GetElevatorData())
		singleElevator = true
	}
	fmt.Println("Alle elevator states:", allElevatorStates)
	return allElevatorStates, singleElevator, newPeerFlag, lostPeerFlag
}

func getHeading(order Order) int {
	currentFloor := fsm.GetElevatorData().Floor
	diff := currentFloor - order.Floor
	if diff == 0 {
		return DirectionNone
	} else if diff < 0 {
		return DirectionUp
	}
	return DirectionDown
}

func orderCompleted(orders OrderList, order Order, orderDoneTxChan chan<- network.UDPmessage) OrderList {
	if order.Type == OrderInternal {
		orders = RemoveDoneOrders(orders, order)
		backup.SaveToFile(orders)
		driver.SetButtonLamp(driver.ButtonInternalOrder, order.Floor, 0)
	} else if order.Type == OrderExternal {
		driver.SetButtonLamp(driver.ButtonType(order.Direction), order.Floor, 0)
			orders = RemoveDoneOrders(orders, order)

		if order.AssignedTo == localId {
			// Goroutine that sends "order completed" message repeatedly for redundancy
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
