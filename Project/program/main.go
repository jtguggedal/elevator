package main

import (
	"./driver"
	fsm "./state_machine"
	"./network"
	"./order_handler"
	"fmt"
	//"time"
	"flag"
	//"encoding/json"
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
	stateRxChannel := make(chan network.UDPmessage)
	stateTxChannel := make(chan network.UDPmessage)
	orderRxChannel := make(chan network.UDPmessage)
	orderTxChannel := make(chan network.UDPmessage)
	orderFinishedChannel := make(chan network.UDPmessage)
	buttonEventChannel := make(chan driver.ButtonEvent)
	floorEventChannel := make(chan int)
	currentFloorChannel := make(chan int)

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
				floorEventChannel,
				currentFloorChannel)

	// Initialize order handler
	go order_handler.Init(	orderRxChannel,
							orderTxChannel,
							orderFinishedChannel,
							buttonEventChannel,
							currentFloorChannel	)


	for {
		select {

		// Route incoming UDP messages to the right module
		case msg := <-UDPrxChannel:
			switch msg.Type {
			case network.MsgState:
				stateRxChannel <- msg
			case network.MsgNewOrder:
				orderRxChannel <- msg
			case network.MsgFinishedOrder:
				orderFinishedChannel <- msg
			}
		}
	}
}
