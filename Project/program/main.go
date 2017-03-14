package main

import (
	"./driver"
	"./fsm"
	"./network"
	"./network/peers"
	"./order_control"
	"fmt"
	"flag"
	"time"
	"encoding/json"
)

func main() {
	// Creating ID for elevator
	var localId string
	flag.StringVar(&localId, "id", "", "ID of this peer")

	// Setting simulator parameters
	simulatorPort := flag.Int("sim_port", 45657, "Port used for simulator communications")
	simulator := flag.Bool("sim", false, "Run in simulator mode")
	flag.Parse()
	fmt.Println("Starting elevator...")

	// Initializing all channels to be used for communication. All channels are one-way.
	buttonEventChan := make(chan driver.ButtonEvent)
	floorReachedChan := make(chan int)
	targetFloorChan := make(chan int)
	floorCompletedChan := make(chan int)
	distributeStateChan := make(chan fsm.ElevatorData_t)
	peerUpdateChan := make(chan peers.PeerUpdate)
	var orderChannels, orderDoneChannels, stateChannels  network.ChannelPair
	orderChannels.Rx = make(chan network.UDPmessage)
	orderChannels.Tx = make(chan network.UDPmessage)
	orderDoneChannels.Rx = make(chan network.UDPmessage)
	orderDoneChannels.Tx = make(chan network.UDPmessage)
	stateChannels.Rx = make(chan network.UDPmessage)
	stateChannels.Tx = make(chan network.UDPmessage)


	go network.UDPinit(	localId,
						peerUpdateChan,
						orderChannels,
						orderDoneChannels,
						stateChannels)

	driver.ElevatorDriverInit(	*simulator,
								*simulatorPort,
								buttonEventChan,
								floorReachedChan)

	go order_control.Init(	orderChannels,
							orderDoneChannels,
							buttonEventChan,
							targetFloorChan,
							floorCompletedChan,
							stateChannels.Rx,
							peerUpdateChan)

	go fsm.Init(floorReachedChan,
				targetFloorChan,
				floorCompletedChan,
				distributeStateChan)


	for {
		// Loop taking care of broadcasting the elevator state
		select {
		case elevatorData := <- distributeStateChan:
			elevatorData.Id = localId
			data, _ := json.Marshal(elevatorData)
			msg := network.UDPmessage{Type: network.MsgState, Data: data}
			for i:= 0; i < 10; i++ {
				stateChannels.Tx <- msg
				time.Sleep(100 * time.Millisecond)
			}
		}

	}
}
