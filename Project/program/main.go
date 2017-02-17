package main

import (
	"./driver"
	"fmt"
	//"time"
	"./network"
	fsm "./state_machine"
)

func main() {
	fmt.Println("Starting...")

	currentFloor := 0
	nextFloor := 0

	// Create necessary channels
	buttonEventChannel := make(chan driver.ButtonEvent)
	floorEventChannel := make(chan int)

	// Initialize elevator driver
	driver.ElevatorDriverInit()

	// Start monitoring when elevator reaches a floor
	fsm.FloorMonitor(floorEventChannel)

	// Start button polling (all buttons)
	go driver.ButtonPoll(buttonEventChannel)

	go func() {
		for {
			select {
			case buttonEvent := <-buttonEventChannel:
				nextFloor = buttonEvent.Floor
				fmt.Println("Next order: ", buttonEvent.Floor)
				if nextFloor < currentFloor {
					driver.SetMotorDirection(driver.MOTOR_DIRECTION_DOWN)
				} else if nextFloor > currentFloor {
					driver.SetMotorDirection(driver.MOTOR_DIRECTION_UP)
				}
			case floor := <-floorEventChannel:
				currentFloor = floor
				if currentFloor == nextFloor {
					driver.SetMotorDirection(driver.MOTOR_DIRECTION_STOP)
				}
				fmt.Println("Floor: ", currentFloor+1)
			}
		}
	}()

	network.Init()
	go func() {
		var prevFloor int
		var currentFloor int
		for {
			currentFloor = driver.GetFloorSensorSignal()
			if currentFloor != prevFloor && currentFloor >= 0 {
				driver.SetFloorIndicator(driver.GetFloorSensorSignal())
				prevFloor = currentFloor
			}
		}
	}()

	go func() {
		for {
			if driver.GetFloorSensorSignal() >= 2 {
				driver.SetMotorDirection(driver.MOTOR_DIRECTION_STOP)
				return
			}
		}
	}()

	for {

	}
}
