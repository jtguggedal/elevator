package order_handler

import (
	"./../driver"
	"./../network"
	"./../fsm"
	"time"
	"fmt"
	"encoding/json"
)

const (
	directionUp		= driver.ButtonExternalUp
	directionDown	= driver.ButtonExternalDown
)

const (
	orderInternal = 1
	orderExternal = 2
)

type orderDirection int
type orderType int

type Order struct {
	Id			int
	Type		orderType
	Origin		network.Ip
	Floor		int
	Direction	orderDirection
	AssignedTo	int
	Done 		bool
}

type orderList []Order

func ReceiveOrder(orderRx chan Order) {

}


func Init(	localIp network.Ip,
			orderRx <-chan network.UDPmessage,
			orderTx chan<- network.UDPmessage,
			orderFinishedChannel chan network.UDPmessage,
			buttonEventChannel <-chan  driver.ButtonEvent,
			currentFloorChannel <-chan int,
			targetFloorChannel chan<- int,
			floorCompletedChannel <-chan int,
			getStateChannel <-chan fsm.ElevatorData_t) {

	var externalOrders orderList
	var internalOrders orderList
	var elevatorData fsm.ElevatorData_t
//	orderChannel := make(chan orderList)
	distributeOrderChannel := make(chan Order)

	newOrderChannel := make(chan bool)

//	var targetFloor int


	go buttonEventListener(	buttonEventChannel,
							distributeOrderChannel)

	go prioritizeOrders(newOrderChannel,
						&externalOrders,
						&internalOrders,
						targetFloorChannel)
	//go execute()

	for {
		select {
		case msg := <- orderRx:
			var receivedOrder Order
			err := json.Unmarshal(msg.Data, &receivedOrder)
			if err == nil {
				if !checkIfOrderExists(externalOrders, receivedOrder) {
					externalOrders = addOrder(externalOrders, receivedOrder)
					fmt.Println("External orders updated:", externalOrders)
					newOrderChannel <- true
				} else {
					fmt.Println("External order already exists", receivedOrder)
				}
			} else {
				fmt.Println("External order message wrongly formatted:", err)
			}

		case order := <- distributeOrderChannel:
			if order.Type == orderInternal {
				if !checkIfOrderExists(internalOrders, order) {
					internalOrders = addOrder(internalOrders, order)
					//newOrderChannel <- true
					fmt.Println("Internal orders updated:", internalOrders)
				} else {
					fmt.Println("Internal order already exists", order)
				}
			} else if order.Type == orderExternal {
				order.Origin = localIp
				orderJson, _ := json.Marshal(order)
				orderTx <- network.UDPmessage{Type: network.MsgNewOrder, Data: orderJson}
				fmt.Println("Order broadcasted" )
			}

		case <- floorCompletedChannel:
		case elevatorData = <- getStateChannel:
			fmt.Println("Elevator data:", elevatorData)
		case <- getStateChannel:
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

				// TODO: error handling
				fmt.Println("External button event: UP from floor", buttonEvent.Floor)
				distributeOrderChannel <- Order{Id: makeOrderId(),
												Type: orderExternal,
												Floor: buttonEvent.Floor,
												Direction: directionUp}

			case driver.ButtonExternalDown:

				// TODO: error handling
				fmt.Println("External button event: DOWN from floor", buttonEvent.Floor)
				distributeOrderChannel <- Order{Id: makeOrderId(),
												Type: orderExternal,
												Floor: buttonEvent.Floor,
												Direction: directionDown}

			case driver.ButtonInternalOrder:

				// TODO: error handling
				fmt.Println("Internal button event: go to floor", buttonEvent.Floor)
				distributeOrderChannel <- Order{Id: makeOrderId(),
												Type: orderInternal,
												Floor: buttonEvent.Floor}
			}
		}
	}
}


func prioritizeOrders(	newOrderSignal <-chan bool,
						externalOrders,
						internalOrders *orderList,
						targetFloorChannel chan<- int) {
	for {
		select {
		case <-newOrderSignal:
			fmt.Println("EXT", len(*externalOrders))
		}
	}
}



func addOrder(orders []Order, newOrder Order) []Order {
	if !checkIfOrderExists(orders, newOrder) {
		orders = append(orders, newOrder)
	}
	return orders
}

func removeDoneOrders(orders []Order, doneOrder Order) []Order {
	for key, order := range orders {
		if checkIfOrdersEqual(order, doneOrder) && doneOrder.Done == true {
			orders = append(orders[:key], orders[key+1:]...)
		}
	}
	return orders
}


func makeOrderId() int {
	return 	int(time.Now().UnixNano()/1e8-1488*1e7)
}

func orderToJson(id int, orderType orderType, floor int, direction orderDirection) []byte {
	ret, _ := json.Marshal(Order{
						Id: id,
						Type: orderType,
						Floor: floor,
						Direction: direction})
	return ret
}

func broadcastNewOrder(order []byte, broadcastChannel chan<- network.UDPmessage) {
	broadcastChannel <- network.UDPmessage{Type: network.MsgNewOrder, Data: order}
}


func checkIfOrdersEqual(o1, o2 Order) bool {
	if 	o1.Type == o2.Type 		&&
		o1.Floor == o2.Floor 	&&
		o1.Direction == o2.Direction {
		return true
	}
	return false
}

func checkIfOrderExists(orders []Order, candidateOrder Order)(bool) {
	for _, order := range orders {
		if 	(order.Type == candidateOrder.Type)				&&
			(order.Floor == candidateOrder.Floor) 			&&
			(order.Direction == candidateOrder.Direction) {
			return true
		}
	}
	return false
}

func firstOrder(input orderList)(int) {
	var ret  int = 10e15
	for id, _ := range input {
		if id < ret {
			ret = id
		}
	}
	return ret
}
