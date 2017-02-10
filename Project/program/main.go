package main

import (
	"fmt"
	"./driver"
	//"time"
	"./state_machine"
)


func main() {
	fmt.Println("Starting...")

	currentFloor := 0
	nextFloor := 0

	buttonEventChan := make(chan driver.ButtonEvent)
	floorEventChan := make(chan int)


	driver.ElevatorDriverInit()
	state_machine.FloorMonitor(floorEventChan)


	driver.ButtonPoll(buttonEventChan)

	go func() {
		for {
			select {
			case buttonEvent := <- buttonEventChan:
				nextFloor = buttonEvent.Floor
				fmt.Println("Next order: ", buttonEvent.Floor)
				if(nextFloor < currentFloor) {
					driver.SetMotorDirection(driver.MOTOR_DIRECTION_DOWN)
				} else if(nextFloor > currentFloor) {
					driver.SetMotorDirection(driver.MOTOR_DIRECTION_UP)
				}
			case floor := <- floorEventChan:
				currentFloor = floor
				if(currentFloor == nextFloor) {
					driver.SetMotorDirection(driver.MOTOR_DIRECTION_STOP)
				}
				fmt.Println("Floor: ", currentFloor + 1)
			}

		}
	}()

	go func() {
		var prevFloor int
		var currentFloor int
		for {
			currentFloor = driver.GetFloorSensorSignal()
			if(currentFloor != prevFloor && currentFloor >= 0) {
				driver.SetFloorIndicator(driver.GetFloorSensorSignal())
				prevFloor = currentFloor
			}
		}
	}()




	go func() {
		for {
			if(driver.GetFloorSensorSignal() >= 2) {
				driver.SetMotorDirection(driver.MOTOR_DIRECTION_STOP)
				return
			}
		}
	}()

	for {

	}
}


