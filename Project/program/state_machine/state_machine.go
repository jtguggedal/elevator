package state_machine

import (
	"./../driver"
	"fmt"
)

type Direction int

const (
	DIRECTION_STOP = 0
	DIRECTION_UP   = 1
	DIRECTION_DOWN = -1
)

type State struct {
	Id        string
	Direction Direction
	Floor     int
}

type StateMsg State

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
