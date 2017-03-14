package main

import (
	"./driver"
	"./fsm"
	"./network"
	"./network/peers"
	"./order_handler"
	"fmt"
	"flag"
	"time"
	"encoding/json"
)

func main() {
	var localId string
	flag.StringVar(&localId, "id", "", "ID of this peer")
	simulatorPort := flag.Int("sim_port", 45657, "Port used for simulator communications")
	simulator := flag.Bool("sim", false, "Run in simulator mode")
	flag.Parse()
	fmt.Println("Starting...")

	stateRxChan := make(chan network.UDPmessage)
	stateTxChan := make(chan network.UDPmessage)
	orderRxChan := make(chan network.UDPmessage)
	orderTxChan := make(chan network.UDPmessage)
	orderDoneRxChan := make(chan network.UDPmessage)
	orderDoneTxChan := make(chan network.UDPmessage)
	buttonEventChan := make(chan driver.ButtonEvent)
	resendStateChan := make(chan bool)

	floorReachedChan := make(chan int)
	targetFloorChan := make(chan int)
	floorCompletedChan := make(chan int)
	distributeStateChan := make(chan fsm.ElevatorData_t)
	peerUpdateChan := make(chan peers.PeerUpdate)

	go network.UDPinit(	localId,
						stateRxChan,
						stateTxChan,
						orderRxChan,
						orderTxChan,
						peerUpdateChan,
						orderDoneRxChan,
						orderDoneTxChan)

	fmt.Println("ID:", localId)

	time.Sleep(1 * time.Second)

	driver.ElevatorDriverInit(	*simulator,
								*simulatorPort,
								buttonEventChan,
								floorReachedChan)

	go order_handler.Init(	orderRxChan,
							orderTxChan,
							orderDoneRxChan,
							orderDoneTxChan,
							buttonEventChan,
							targetFloorChan,
							floorCompletedChan,
							stateRxChan,
							peerUpdateChan,
							resendStateChan)

	go fsm.Init(floorReachedChan,
				targetFloorChan,
				floorCompletedChan,
				distributeStateChan,
				resendStateChan)


	for {
		select {
		case elevatorData := <- distributeStateChan:
			elevatorData.Id = localId
			data, _ := json.Marshal(elevatorData)
			msg := network.UDPmessage{Type: network.MsgState, Data: data}
			for i:= 0; i < 10; i++ {
				stateTxChan <- msg
				time.Sleep(100 * time.Millisecond)
			}
		}

	}
}
