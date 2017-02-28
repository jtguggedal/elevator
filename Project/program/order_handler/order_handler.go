package order_handler

import (
	"./../driver"
	"./../network"
	"./../state_machine"
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
	Floor		int
	Direction	orderDirection
	AssignedTo	int
}

type orderList map[int]Order

func ReceiveOrder(orderRx chan Order) {

}


func Init(	orderRx <-chan network.UDPmessage,
			orderTx chan<- network.UDPmessage,
			orderFinishedChannel chan network.UDPmessage,
			buttonEventChannel <-chan  driver.ButtonEvent,
			currentFloorChannel <-chan int,
			targetFloorChannel chan<- int) {

	externalOrders := make(orderList)
	internalOrders := make(orderList)
	orderChannel := make(chan orderList)
	internalOrderFinished := make(chan Order)
	externalOrderFinished := make(chan Order)

	go executeOrders(	orderChannel,
						currentFloorChannel,
						orderFinishedChannel,
						internalOrderFinished,
						externalOrderFinished)

	for {
		select {
		case msg := <- orderRx:

			// Message contains order as JSON - converting to Order struct before using
			var receivedOrder Order
			err := json.Unmarshal(msg.Data, &receivedOrder)
			if err == nil {
				if !checkIfOrderExists(externalOrders, receivedOrder) {

					externalOrders[receivedOrder.Id] = receivedOrder
					internalOrders[receivedOrder.Id] = receivedOrder
					orderChannel <- internalOrders

					fmt.Println("External orders updated:", externalOrders)
					continue
				} else {
					fmt.Println("External order already exists", receivedOrder)
				}
			} else {
				fmt.Println("External order message wrongly formatted", receivedOrder)
			}
		case buttonEvent := <- buttonEventChannel:
			driver.SetButtonLamp(buttonEvent.Type, buttonEvent.Floor, 1)

			switch buttonEvent.Type  {
			case driver.ButtonExternalUp:

				// TODO: error handling
				fmt.Println("Received external button event: UP from floor", buttonEvent.Floor)
				newOrder := orderToJson(makeOrderId(), orderExternal, buttonEvent.Floor, directionUp)
				orderTx <- network.UDPmessage{Type: network.MsgNewOrder, Data: newOrder}

			case driver.ButtonExternalDown:

				// TODO: error handling
				fmt.Println("Received external button event: DOWN from floor", buttonEvent.Floor)
				newOrderJson := orderToJson(makeOrderId(), orderExternal, buttonEvent.Floor, directionDown)
				broadcastNewOrder(newOrderJson, orderTx)
				fmt.Println("Order sent")

			case driver.ButtonInternalOrder:

				// TODO: error handling
				fmt.Println("Received internal button event: go to floor", buttonEvent.Floor)
				newOrder := Order{	Id: int(time.Now().UnixNano()/1e8-1488*1e7),
									Type: orderInternal,
									Floor: buttonEvent.Floor}
				if !checkIfOrderExists(internalOrders, newOrder) {
					internalOrders[newOrder.Id] = newOrder
					orderChannel <- internalOrders
					fmt.Println("Internal orders updated:", internalOrders)
					continue
				} else {
					fmt.Println("Internal order already exists", newOrder)
				}
			}
		case orderFinished := <-internalOrderFinished:
			delete(internalOrders, orderFinished.Id)
			driver.SetButtonLamp(driver.ButtonInternalOrder, orderFinished.Floor, 0)
			fmt.Println("Order completed:", orderFinished, internalOrders)

		case orderFinished := <-externalOrderFinished:
			driver.SetButtonLamp(driver.ButtonType(orderFinished.Direction), orderFinished.Floor, 0)
			delete(externalOrders, orderFinished.Id)
			//orderJson, _ := json.Marshal(orderFinished)
			//orderTx <- network.UDPmessage{Type: network.MsgFinishedOrder, Data: orderJson}
			fmt.Println("External order completed:", orderFinished, externalOrders)
		}
	}
}


func executeOrders(	orderChannel <-chan orderList,
					currentFloorChannel <-chan int,
					orderFinishedChannel chan<- network.UDPmessage,
					internalOrderFinished chan<- Order,
					externalOrderFinished chan<- Order) {

	currentFloor := 0
	var orders orderList
	for {
		select {
		case currentFloor = <- currentFloorChannel:
			for _, order := range orders {
				if order.Floor == currentFloor {
					if order.Type == orderInternal {
						internalOrderFinished<- order
					} else if order.Type == orderExternal {
						externalOrderFinished<- order
					}
					delete(orders, order.Id)
					driver.SetMotorDirection(driver.DirectionStop)
					fmt.Println("Door open")
					state_machine.DoorOpen()
					fmt.Println("Door closed")
					fmt.Println("shortcut", orders)
				}
			}
		case orders = <- orderChannel:
			fmt.Println("Received for exectution:", orders)
		}
		if len(orders) > 0 {
			first := firstOrder(orders)
			order := orders[first]
			if order.Floor == currentFloor {
				driver.SetMotorDirection(driver.DirectionStop)
				internalOrderFinished<- order
			} else if order.Floor > currentFloor {
				driver.SetMotorDirection(driver.DirectionUp)
			} else if order.Floor < currentFloor {
				driver.SetMotorDirection(driver.DirectionDown)
			}
		}
	}
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



func checkIfOrderExists(orders map[int]Order, newOrder Order)(bool) {
	for _, order := range orders {
		if (order.Floor == newOrder.Floor) && (order.Direction == newOrder.Direction) {
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
