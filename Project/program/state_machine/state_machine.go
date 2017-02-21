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

func Init(stateRx, stateTx chan StateMsg /*, states []State*/) {
	go FloorMonitor(stateRx)
	go StateMonitor(stateRx)
	for {
		select {
		case receivedState := <-stateRx:
			//var states []State
			//states = append(states, State(receivedState))
			//StateChange(states, receivedState)
			fmt.Println(receivedState)
		}
	}
}

func FloorMonitor(channel chan StateMsg) {
	var prevFloor int
	var currentFloor int
	for {
		select {
		case a := <-channel:
			currentFloor = a.Floor
			if currentFloor != prevFloor && currentFloor >= 0 {
				driver.SetFloorIndicator(driver.GetFloorSensorSignal())
				prevFloor = currentFloor
			}
			if currentFloor == -1 {
				// Elevator between two floors
			}
		}
	}
}

func StateMonitor(receivedState StateMsg) {
	var states []State
	for i, element := range states {
		if element.Id == receivedState.Id {
			states[i].Direction = receivedState.Direction
			states[i].Floor = receivedState.Floor
		}
	}
	//fmt.Printf("States: %#v\n", states)
}
