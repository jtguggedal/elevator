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

func Init(stateRx chan StateMsg, states []State) {
	go func() {
		for {
			select {
			case receivedState := <-stateRx:
				StateChange(states, receivedState)
			}
		}
	}()
}

func FloorMonitor(channel chan int) {
	var prevFloor int
	var currentFloor int
	go func() {
		for {
			currentFloor = driver.GetFloorSensorSignal()
			if currentFloor != prevFloor && currentFloor >= 0 {
				driver.SetFloorIndicator(driver.GetFloorSensorSignal())
				channel <- currentFloor
				prevFloor = currentFloor
			}
			if currentFloor == -1 {
				// Elevator between two floors
			}
		}
	}()
}

func StateChange(states []State, receivedState StateMsg) {
	for i, element := range states {
		if element.Id == receivedState.Id {
			states[i].Direction = receivedState.Direction
			states[i].Floor = receivedState.Floor
		}
	}
	fmt.Printf("States: %#v\n", states)
}
