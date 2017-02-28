package main

import (
	"./driver"
	"./fsm"
	"./network"
	"./order_handler"
	"fmt"
	"time"
	"flag"
	//"encoding/json"
)

func main() {
	var id string
	flag.StringVar(&id, "id", "", "ID of this peer")
	simulatorPort := flag.Int("sim_port", 15657, "Port used for simulator communications")
	simulator := flag.Bool("sim", false, "Run in simulator mode")
	flag.Parse()
	fmt.Println("Starting...")

	UDPrxChannel := make(chan network.UDPmessage)
	UDPtxChannel := make(chan network.UDPmessage)
	peerStatusChannel := make(chan network.PeerStatus)
	stateRxChannel := make(chan network.UDPmessage)
	stateTxChannel := make(chan network.UDPmessage)
	orderRxChannel := make(chan network.UDPmessage)
	orderTxChannel := make(chan network.UDPmessage)
	orderFinishedChannel := make(chan network.UDPmessage)
	buttonEventChannel := make(chan driver.ButtonEvent)
	floorEventChannel := make(chan int)
	targetFloorChannel := make(chan int)
	currentFloorChannel := make(chan int)
	//livePeersChannel := make(chan []string)

	go network.UDPinit(	id,
						stateRxChannel,
						stateTxChannel,
						orderRxChannel,
						orderTxChannel,
						UDPrxChannel,
						UDPtxChannel,
						peerStatusChannel)

	driver.ElevatorDriverInit(*simulator,
			*simulatorPort,
			buttonEventChannel,
			floorEventChannel)


	go fsm.Init(floorEventChannel,
				targetFloorChannel)

	go order_handler.Init(	orderRxChannel,
							orderTxChannel,
							orderFinishedChannel,
							buttonEventChannel,
							currentFloorChannel,
							targetFloorChannel	)


	for {
		select {
		//case msg := <- orderTx:
		//	UDPtxChannel<- msg

		// Route incoming UDP messages to the right module
		case msg := <-UDPrxChannel:
			switch msg.Type {
			case network.MsgState:
				stateRxChannel <- msg
			case network.MsgNewOrder:
				orderRxChannel <- msg
				fmt.Println("ds")
			case network.MsgFinishedOrder:
				orderFinishedChannel <- msg
			}
		case <- time.After(15 * time.Second):
			targetFloorChannel <- 3
		}
	}
}
