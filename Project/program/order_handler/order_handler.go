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
	DirectionUp		= driver.ButtonExternalUp
	DirectionDown	= driver.ButtonExternalDown
)

const (
	orderInternal = 1
	orderExternal = 2
)

type OrderDirection int
type OrderType int

type Order struct {
	Id			int
	Type		OrderType
	Floor		int
	Direction	OrderDirection
	AssignedTo	int
}

type OrderList map[int]Order

func ReceiveOrder(orderRx chan Order) {

}


func Init(	orderRx <-chan network.UDPmessage,
			orderTx chan<- network.UDPmessage,
			orderFinishedChannel chan network.UDPmessage,
			buttonEventChannel <-chan  driver.ButtonEvent,
			currentFloorChannel <-chan int) {

	externalOrders := make(OrderList)
	internalOrders := make(OrderList)
	orderChannel := make(chan OrderList)
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
				newOrder, _ := json.Marshal( Order{
									Id: int(time.Now().UnixNano()/1e8-1488*1e7),
									Type: orderExternal,
									Floor: buttonEvent.Floor,
									Direction: DirectionUp})
				orderTx <- network.UDPmessage{Type: network.MsgNewOrder, Data: newOrder}

			case driver.ButtonExternalDown:

				// TODO: error handling
				fmt.Println("Received external button event: DOWN from floor", buttonEvent.Floor)
				newOrder, _ := json.Marshal( Order{
									Id: int(time.Now().UnixNano()/1e8-1488*1e7),
									Type: orderExternal,
									Floor: buttonEvent.Floor,
									Direction: DirectionDown})
				orderTx <- network.UDPmessage{Type: network.MsgNewOrder, Data: newOrder}

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
			//orderChannel <- internalOrders
			driver.SetButtonLamp(driver.ButtonInternalOrder, orderFinished.Floor, 0)
			fmt.Println("Order completed:", orderFinished, internalOrders)

		case orderFinished := <-externalOrderFinished:
			delete(externalOrders, orderFinished.Id)
			driver.SetButtonLamp(driver.ButtonType(orderFinished.Direction), orderFinished.Floor, 0)
			fmt.Println("Order completed:", orderFinished, externalOrders)
		}
	}
}


func executeOrders(	orderChannel <-chan OrderList,
					currentFloorChannel <-chan int,
					orderFinishedChannel chan<- network.UDPmessage,
					internalOrderFinished chan<- Order,
					externalOrderFinished chan<- Order) {

	currentFloor := 0
	var orders OrderList
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
					//fmt.Println("shortcut", orders)
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

func checkIfOrderExists(orders map[int]Order, newOrder Order)(bool) {
	for _, order := range orders {
		if (order.Floor == newOrder.Floor) && (order.Direction == newOrder.Direction) {
			return true
		}
	}
	return false
}

func firstOrder(input OrderList)(int) {
	var ret  int = 10e15
	for id, _ := range input {
		if id < ret {
			ret = id
		}
	}
	return ret
}
