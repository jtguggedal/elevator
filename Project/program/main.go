package main

import (
	"./driver"
	"./fsm"
	"./network"
	"./network/peers"
	"./order_handler"
	"fmt"
	"flag"
	"encoding/json"
)

func main() {
	var id string
	flag.StringVar(&id, "id", "", "ID of this peer")
	simulatorPort := flag.Int("sim_port", 45657, "Port used for simulator communications")
	simulator := flag.Bool("sim", false, "Run in simulator mode")
	flag.Parse()
	fmt.Println("Starting...")

	UDPrxChannel := make(chan network.UDPmessage)
	UDPtxChannel := make(chan network.UDPmessage)
	stateRxChannel := make(chan network.UDPmessage)
	stateTxChannel := make(chan network.UDPmessage)
	orderRxChannel := make(chan network.UDPmessage)
	orderTxChannel := make(chan network.UDPmessage)
	orderDoneRxChannel := make(chan network.UDPmessage)
	orderDoneTxChannel := make(chan network.UDPmessage)
	currentFloorChannel := make(chan int)
	buttonEventChannel := make(chan driver.ButtonEvent)
	ipChannel := make(chan network.Ip)
	resendStateChannel := make(chan bool)

	floorReachedChannel := make(chan int)
	targetFloorChannel := make(chan int)
	floorCompletedChannel := make(chan int)
	distributeStateChannel := make(chan fsm.ElevatorData_t)
	peerUpdateChannel := make(chan peers.PeerUpdate)

	go network.UDPinit(	id,
						ipChannel,
						stateRxChannel,
						stateTxChannel,
						orderRxChannel,
						UDPrxChannel,
						UDPtxChannel,
						peerUpdateChannel,
						orderDoneRxChannel,
						orderDoneTxChannel)
	localIp := <- ipChannel


	driver.ElevatorDriverInit(	*simulator,
								*simulatorPort,
								buttonEventChannel,
								floorReachedChannel)

	go order_handler.Init(	orderRxChannel,
							orderTxChannel,
							orderDoneRxChannel,
							orderDoneTxChannel,
							buttonEventChannel,
							currentFloorChannel,
							targetFloorChannel,
							floorCompletedChannel,
							stateRxChannel,
							peerUpdateChannel,
							resendStateChannel)

	go fsm.Init(floorReachedChannel,
				targetFloorChannel,
				floorCompletedChannel,
				distributeStateChannel,
				resendStateChannel)


	for {
		select {
		case msg := <-orderTxChannel:
			UDPtxChannel <- msg
		case msg := <-UDPrxChannel:
			switch msg.Type {
			case network.MsgState:
			case network.MsgNewOrder:
				orderRxChannel <- msg
			}
		case elevatorData := <- distributeStateChannel:
			elevatorData.Id = localIp
			data, _ := json.Marshal(elevatorData)
			msg := network.UDPmessage{Type: network.MsgState, Data: data}
			stateTxChannel <- msg
		}

	}
}
