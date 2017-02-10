package state_machine

import( 
	"./../driver"
)



func FloorMonitor(channel chan int) {
	var prevFloor int
	var currentFloor int
	go func() {
		for {
			currentFloor = driver.GetFloorSensorSignal()
			if(currentFloor != prevFloor && currentFloor >= 0) {
				driver.SetFloorIndicator(driver.GetFloorSensorSignal())
				channel <- currentFloor
				prevFloor = currentFloor
			}
			if(currentFloor == -1) {
				// Elevator between two floors
			}
		}
	}()
}

