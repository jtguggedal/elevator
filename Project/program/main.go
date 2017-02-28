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
	flag.Int("sim_port", 15657, "Port used for simulator communications")
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
	currentFloorChannel := make(chan int)
	livePeersChannel := make(chan []string)

	go network.UDPinit(	id,
						stateRxChannel,
						stateTxChannel,
						orderRxChannel,
						orderTxChannel,
						UDPrxChannel,
						UDPtxChannel,
						peerStatusChannel)

	driver.ElevatorDriverInit(*simulator,
			buttonEventChannel,
			floorEventChannel)


	go fsm.Init(stateRxChannel,
				stateTxChannel,
				floorEventChannel,
				currentFloorChannel,
				peerStatusChannel,
				livePeersChannel)

	go order_handler.Init(	orderRxChannel,
							orderTxChannel,
							orderFinishedChannel,
							buttonEventChannel,
							currentFloorChannel	)


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
		}
	}
}
