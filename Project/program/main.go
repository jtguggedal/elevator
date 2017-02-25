package main

import (
	"./driver"
	fsm "./state_machine"
	"./network"
	"./order_handler"
	"fmt"
	//"time"
	"flag"
	"encoding/json"
)

func main() {
	var id string
	flag.StringVar(&id, "id", "", "ID of this peer")
	simulator := flag.Bool("sim", false, "Run in simulator mode")
	flag.Parse()
	fmt.Println("Starting...")

	// Initialize network
	UDPrxChannel := make(chan network.UDPmessage)
	UDPtxChannel := make(chan network.UDPmessage)
	stateRxChannel := make(chan fsm.StateMsg)
	stateTxChannel := make(chan fsm.StateMsg)
	orderRxChannel := make(chan network.UDPmessage)
	orderTxChannel := make(chan network.UDPmessage)
	orderFinishedChannel := make(chan network.UDPmessage)
	buttonEventChannel := make(chan driver.ButtonEvent)
	floorEventChannel := make(chan int)

	go network.UDPinit(	id,
						stateRxChannel,
						stateTxChannel,
						orderRxChannel,
						orderTxChannel,
						UDPrxChannel,
						UDPtxChannel)

	// Initialize elevator driver
	driver.ElevatorDriverInit(*simulator)

	// Start event listener
	go driver.EventListener(buttonEventChannel, floorEventChannel)

	// Initialize state machine
	go fsm.Init(stateRxChannel,
				stateTxChannel,
				floorEventChannel)

	// Initialize order handler
	go order_handler.Init(	orderRxChannel,
							orderTxChannel,
							orderFinishedChannel,
							buttonEventChannel)


	for {
		select {

		case msg := <-UDPrxChannel:

				receivedData := msg
			switch msg.Type {
			/*case network.MsgState:

				// State message
				// Proper way to assert interface:
				//receivedData, ok := msg.Data.(fsm.StateMsg)
				ok := true
				if ok {
					stateRxChannel <- receivedData
					fmt.Println("Received state message: ", receivedData)
					continue
				}
				fmt.Println("Wrongly formatted state message received: ", msg.Data)*/
			case network.MsgNewOrder:

				// New order message
				//receivedData, ok := msg.Data.(order_handler.Order)
				ok := true
				if ok {
					orderRxChannel <- msg
					//fmt.Println("Received new order message: ", receivedData)
					continue
				}
				//fmt.Println("Wrongly formatted new order message received: ", msg.Data, receivedData)
				//orderRxChannel <- msg
			case network.MsgFinishedOrder:

				// Order is handled
				//receivedData, ok := msg.Data.(order_handler.Order)
				ok := true
				if ok {
					orderFinishedChannel <- msg
					fmt.Println("Received finished order message: ", receivedData)
					continue
				}
				var data order_handler.Order
				json.Unmarshal([]byte(msg.Data), &data)
				fmt.Println("Wrongly formatted finished order message received: ", data)
			}


		//case updatedFloor := <-floorEventChannel:
		//	stateRxChannel <- fsm.StateMsg{Id: id, Direction: 1, Floor: updatedFloor}
		//	fmt.Println("Arrived at floor:", updatedFloor)
		//case button := <-buttonEventChannel:
		//	fmt.Println("Button pressed: ", button)
		}
	}
}
