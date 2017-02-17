package state_machine

import (
	"./../driver"
)

type FsmDirection int

const (
	DIRECTION_STOP = 0
	DIRECTION_UP   = 1
	DIRECTION_DOWN = -1
)

type StateMsg struct {
	Id        string
	Direction FsmDirection
	Floor     int
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
