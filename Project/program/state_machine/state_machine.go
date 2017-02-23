package state_machine

import (
	"./../driver"
	"fmt"
)

type elevatorDirection driver.MotorDirection
type elevatorState int

const (
	idle elevatorState = iota
	moving
	doorOpen
)

type ElevatorData struct {
	Id        	string
	Direction 	elevatorDirection
	Floor     	int
	State		elevatorState
}

type StateMsg ElevatorData

func Init(	stateRx, stateTx chan StateMsg, 
			buttonEventChannel chan driver.ButtonEvent, 
			floorEventChannel chan int /*, states []State*/) {

	go FloorMonitor(floorEventChannel)
	go StateMonitor(stateRx)
	/*for {
		select {
		case receivedState := <-stateRx:
			//var states []State
			//states = append(states, State(receivedState))
			//StateChange(states, receivedState)
			fmt.Println(receivedState)
		}
	}*/
}

func FloorMonitor(channel chan int) {
	var floorSignal int
	for {
		select {
		case floorSignal = <-channel:
			if floorSignal >= 0 {
				if(floorSignal >= driver.NUMBER_OF_FLOORS) {
					driver.SetMotorDirection(driver.DirectionStop)
				}
				driver.SetFloorIndicator(floorSignal)
				fmt.Println("Arrived at floor", floorSignal)
			} else if floorSignal == -1 {
				fmt.Println("Moving...")
			}
		}
	}
}

func StateMonitor(stateChannel <-chan StateMsg) {
	var states []ElevatorData
	for {
		select {
		case receivedState := <- stateChannel:
			for i, element := range states {
				if element.Id == receivedState.Id {
					states[i].Direction = receivedState.Direction
					states[i].Floor = receivedState.Floor
				}
			}
		}
	}
	//fmt.Printf("States: %#v\n", states)
}
