package order_handler

import (
	"./../driver"
	"./../network"
	"time"
	"fmt"
	"encoding/json"
)

const (
	DirectionUp		= 0
	DirectionDown	= 1
)

type OrderDirection int

type Order struct {
	Id			int
	Floor		int
	Direction	OrderDirection
	AssignedTo	int
}

func ReceiveOrder(orderRx chan Order) {

}


func Init(	orderRx <-chan network.UDPmessage,
			orderTx chan<- network.UDPmessage,
			orderFinished <-chan network.UDPmessage,
			buttonEventChannel <-chan  driver.ButtonEvent) {

	externalOrders := make(map[int]Order)
	//var internalOrders map[int]Order

	for {
		select {
		case msg := <- orderRx:
			var receivedOrder Order
			json.Unmarshal(msg.Data, &receivedOrder)
		//	if !err {
				externalOrders[receivedOrder.Id] = receivedOrder
				fmt.Println("External orders updated:", externalOrders)
				//continue
		//	}
			fmt.Println("External order message wrongly formatted", receivedOrder)
		case buttonEvent := <- buttonEventChannel:
			driver.SetButtonLamp(buttonEvent.Type, buttonEvent.Floor, 1)
			switch buttonEvent.Type  {
			case driver.ButtonExternalUp:
				fmt.Println("Received button event: UP from floor", buttonEvent.Floor)
				newOrder, _ := json.Marshal( Order{Id: int(time.Now().UnixNano()/1e8-1488*1e7), Floor: buttonEvent.Floor, Direction: DirectionUp, AssignedTo: 0})
				orderTx <- network.UDPmessage{Type: network.MsgNewOrder, Data: newOrder}
			}
		}
	}
}
