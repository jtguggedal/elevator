package order_handler

import (
	"./../driver"
	"./../network/peers"
	"./../network"
	"./../fsm"
	// "./../order_cost"
	"time"
	"fmt"
	"encoding/json"
	"math"
)

const (
	directionUp		= driver.ButtonExternalUp
	directionDown	= driver.ButtonExternalDown
)

const (
	orderInternal = 1
	orderExternal = 2
)

const (
	costIdle				= 2
	costMovingSameDir		= 1
	costMovingOpositeDir	= 6
	costPerFloor			= 3
)


type orderDirection int
type orderType int

type Order struct {
	Id			int
	Type		orderType
	Origin		network.Ip
	Floor		int
	Direction	orderDirection
	AssignedTo	network.Ip
	Done 		bool
}

type orderList []Order

var localId network.Ip

func Init(	orderRx <-chan network.UDPmessage,
			orderTx chan<- network.UDPmessage,
			orderDonedRxChannel <-chan network.UDPmessage,
			orderDoneTxChannel chan<- network.UDPmessage,
			buttonEventChannel <-chan  driver.ButtonEvent,
			currentFloorChannel <-chan int,
			targetFloorChannel chan<- int,
			floorCompletedChannel <-chan int,
			stateRxChannel <-chan network.UDPmessage,
			peerUpdateChannel <-chan peers.PeerUpdate,
			resendStateChannel chan<- bool) {

	var externalOrders orderList
	var internalOrders orderList
	var orderQueue orderList
	var allElevatorStates []fsm.ElevatorData_t

	var singleElevator bool = true
	busy := false
	var activeOrder Order

	localId := network.GetLocalId()

	distributeOrderChannel := make(chan Order)

	go buttonEventListener(	buttonEventChannel,
							distributeOrderChannel)

	for {
		select {
		case <- currentFloorChannel:
		case msg := <- orderRx:

			// Order received via UDP
			var receivedOrder Order
			err := json.Unmarshal(msg.Data, &receivedOrder)
			if err == nil {
				if !orderExists(externalOrders, receivedOrder) {
					if singleElevator == true {
						receivedOrder.AssignedTo = localId
						externalOrders = addOrder(externalOrders, receivedOrder)
						orderQueue = addOrder(orderQueue, receivedOrder)
						candidateOrder := getNextOrder(orderQueue)
						if candidateOrder.Floor != -1 && !busy {
							activeOrder = candidateOrder
							busy = true
							targetFloorChannel <- activeOrder.Floor
						}
					} else if singleElevator == false {
						driver.SetButtonLamp(driver.ButtonType(receivedOrder.Direction), receivedOrder.Floor, 1)
						receivedOrder.AssignedTo = assignOrder(allElevatorStates, receivedOrder)
						externalOrders = addOrder(externalOrders, receivedOrder)
						fmt.Printf("Order assigned to %s:\tFloor: %d\n", receivedOrder.AssignedTo, receivedOrder.Floor)
						if receivedOrder.AssignedTo == localId {
							orderQueue = addOrder(orderQueue, receivedOrder)
							activeOrder = getNextOrder(orderQueue)
							if activeOrder.Floor != -1 && !busy {
								busy = true
								targetFloorChannel <- activeOrder.Floor
							}
						}
						fmt.Println("External orders updated:", externalOrders)
					}
				} else {
					fmt.Println("External order already exists", receivedOrder)
				}
			} else {
				fmt.Println("External order message wrongly formatted:", err)
			}

		case order := <- distributeOrderChannel:
			switch  order.Type {
			case orderInternal:
				if !orderExists(internalOrders, order) {
					internalOrders = addOrder(internalOrders, order)
					orderQueue = addOrder(orderQueue, order)
					fmt.Println("Internal orders updated:", internalOrders)
					candidateOrder := getNextOrder(orderQueue)
					if candidateOrder.Floor != -1 && !busy {
						activeOrder = candidateOrder
						busy = true
						targetFloorChannel <- activeOrder.Floor
					}
				} else {
					fmt.Println("Internal order already exists", order)
				}

			case orderExternal:
				order.Origin = localId
				orderJson, _ := json.Marshal(order)
				orderTx <- network.UDPmessage{Type: network.MsgNewOrder, Data: orderJson}
			}

		case floor := <- floorCompletedChannel:
			if floor == activeOrder.Floor {
				busy = false
				activeOrder.Done = true
				orderQueue = orderCompleted(orderQueue, activeOrder)
				if activeOrder.Type == orderInternal {
					removeDoneOrders(internalOrders, activeOrder)
				}
				if !singleElevator {
					orderJson, _ := json.Marshal(activeOrder)
					orderDoneTxChannel <- network.UDPmessage{Type: network.MsgFinishedOrder, Data: orderJson}
				}
			}
			activeOrder = getNextOrder(orderQueue)
			if activeOrder.Floor != -1 && !busy {
				busy = true
				fmt.Println("Next order", activeOrder)
				targetFloorChannel <- activeOrder.Floor
			}

		case stateJson := <- stateRxChannel:
			var state fsm.ElevatorData_t
			jsonToStruct(stateJson.Data, &state)
			allElevatorStates = updatePeerState(allElevatorStates, state)

		case peers := <- peerUpdateChannel:
			var newPeer bool
			allElevatorStates, singleElevator, newPeer = updateLivePeers(peers, allElevatorStates)
			if newPeer {
				resendStateChannel <- true
			}

		case orderMsg := <- orderDonedRxChannel:
			var order Order
			jsonToStruct(orderMsg.Data, &order)
			externalOrders = orderCompleted(externalOrders, order)
			fmt.Println("Remaining external orders: ", externalOrders)
		}
	}
}

func buttonEventListener(	buttonEventChannel <-chan driver.ButtonEvent,
							distributeOrderChannel chan<- Order) {
	for {
		select {
		case buttonEvent := <- buttonEventChannel:
			driver.SetButtonLamp(buttonEvent.Type, buttonEvent.Floor, 1)

			switch buttonEvent.Type  {
			case driver.ButtonExternalUp:
				fmt.Println("External button event: UP from floor", buttonEvent.Floor)
				distributeOrderChannel <- Order{Id: getOrderId(),
												Type: orderExternal,
												Floor: buttonEvent.Floor,
												Direction: directionUp}

			case driver.ButtonExternalDown:
				fmt.Println("External button event: DOWN from floor", buttonEvent.Floor)
				distributeOrderChannel <- Order{Id: getOrderId(),
												Type: orderExternal,
												Floor: buttonEvent.Floor,
												Direction: directionDown}

			case driver.ButtonInternalOrder:
				fmt.Println("Internal button event: go to floor", buttonEvent.Floor)
				distributeOrderChannel <- Order{Id: getOrderId(),
												Type: orderInternal,
												Floor: buttonEvent.Floor}
			}
		}
	}
}



func orderCost(o Order, e fsm.ElevatorData_t) int {
	var cost int
	distance := o.Floor - e.Floor

	idle := e.State == fsm.Idle
	movingSameDir := (int(o.Direction) == int(e.State)) && !idle
	movingOpositeDir := !movingSameDir && !idle

	targetAbove := distance > 0
	targetBelow := distance < 0
	distance = int(math.Abs(float64(distance)))

	if targetAbove  || targetBelow {
		cost += distance * costPerFloor
	}
	if movingSameDir {
		cost += costMovingSameDir
	} else if movingOpositeDir {
		cost += costMovingOpositeDir
	} else if idle {
		cost += costIdle
	}
	return cost
}

func assignOrder(allElevatorStates []fsm.ElevatorData_t, order Order) (network.Ip) {
	cost := 0
	lowestCost := 100000
	var assignTo network.Ip
	for _, elevator := range allElevatorStates {
		cost = orderCost(order, elevator)
		fmt.Printf("Cost for %s: %d\n", elevator.Id, cost)
		if cost < lowestCost {
			lowestCost = cost
			assignTo = elevator.Id
		} else if cost == lowestCost && elevator.Id > assignTo {
			fmt.Println("Assigned to and new", elevator.Id, assignTo)
			assignTo = elevator.Id
		}
	}
	return assignTo
}


func updateLivePeers(	peers peers.PeerUpdate,
	 					allElevatorStates []fsm.ElevatorData_t) ([]fsm.ElevatorData_t, bool, bool) {
	newPeer := len(peers.New) > 0 && peers.Peers[0] != string(localId)
	for i, storedPeer := range allElevatorStates {
		for _, lostPeer := range peers.Lost {
			fmt.Println("Lost peer", lostPeer)
			if storedPeer.Id == "" || network.Ip(lostPeer) == storedPeer.Id {
				allElevatorStates = append(allElevatorStates[:i], allElevatorStates[i+1:]...)
			}
		}
	}
	singleElevator := len(peers.Peers) == 1
	return allElevatorStates, singleElevator, newPeer
}


func updatePeerState(	allElevatorStates []fsm.ElevatorData_t,
						state fsm.ElevatorData_t) ([]fsm.ElevatorData_t){
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


func isOrderOnTheWay(order, activeOrder Order) (bool) {
	elevatorState := fsm.GetElevatorData()
	if (elevatorState.Floor > order.Floor) && (order.Floor > activeOrder.Floor) {
		return true
	} else if elevatorState.Floor < order.Floor && order.Floor < activeOrder.Floor {
		return true
	}
	return false
}

func getNextOrder(orders orderList) (Order) {
	if len(orders) == 0 {
		return Order{Floor: -1}
	}
	oldestEntryTime := int(10e15)
	var nextOrder Order
	//var orderDir orderDirection
	var internal bool
	nextOrder.Floor = -1

	// Serving internal orders first
	for _, order := range orders {
		if order.Type == orderInternal {
			nextOrder = order
			internal = true
		}
	}
	if internal {
		return nextOrder
	}

	for _, order := range orders {
		if order.Id < oldestEntryTime {
			oldestEntryTime = order.Id
			nextOrder = order
		}
	}
	for _, order := range orders {
		if isOrderOnTheWay(order, nextOrder) {
			nextOrder = order
		}
	}
	return nextOrder
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
	return 	int(time.Now().UnixNano()/1e8-1488*1e7)
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
						Id: id,
						Type: orderType,
						Floor: floor,
						Direction: direction})
	return ret
}

func ordersEqual(o1, o2 Order) bool {
	if 	o1.Type == o2.Type 		&&
		o1.Floor == o2.Floor 	&&
		o1.Direction == o2.Direction {
		return true
	}
	return false
}

func orderExists(orders []Order, candidateOrder Order)(bool) {
	for _, order := range orders {
		if 	(order.Type == candidateOrder.Type)				&&
			(order.Floor == candidateOrder.Floor) 			&&
			(order.Direction == candidateOrder.Direction) {
			return true
		}
	}
	return false
}
