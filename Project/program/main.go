package main

import (
	"./driver"
	"fmt"
	//"time"
	"./network"
	fsm "./state_machine"
	"flag"
)

func main() {
	var id string
	flag.StringVar(&id, "id", "", "ID of this peer")
	flag.Parse()
	fmt.Println("Starting...")

	// Initialize network
	stateRxChannel := make(chan fsm.StateMsg)
	stateTxChannel := make(chan fsm.StateMsg)
	go network.UDPinit(id, stateRxChannel, stateTxChannel)

	// Initialize elevator driver
	driver.ElevatorDriverInit()

	// Start event listener
	buttonEventChannel := make(chan driver.ButtonEvent)
	floorEventChannel := make(chan int)
	go driver.EventListener(buttonEventChannel, floorEventChannel)

	// Initialize state machine
	go fsm.Init(stateRxChannel, stateTxChannel)

	for {
		select {
		case updatedFloor := <-floorEventChannel:
			stateRxChannel <- fsm.StateMsg{Id: id, Direction: 1, Floor: updatedFloor}
			fmt.Println("received", updatedFloor)
		case button := <-buttonEventChannel:
			fmt.Println(button)
		}
	}
}
