package fsm

import (
	"fmt"
	"time"

	"./../driver"
)

type state_t int

const (
	Idle       state_t = 0
	MovingUp   state_t = state_t(driver.DirectionUp)
	MovingDown state_t = state_t(driver.DirectionDown)
	DoorOpen   state_t = 3
	Stuck      state_t = 4
)

const doorOpenTime = 5
const TimeBetweenFloors = 6 * time.Second
const targetFloorReached = -1

type ElevatorData_t struct {
	Id    string
	State state_t
	Floor int
}

var elevatorData ElevatorData_t

func GetElevatorData() ElevatorData_t {
	return elevatorData
}

func Init(floorSignalChannel <-chan int,
	newTargetFloorChannel <-chan int,
	floorCompletedChannel chan<- int,
	distributeStateChannel chan<- ElevatorData_t) {

	var floor int
	var targetFloor int

	currentFloorChannel := make(chan int, 10)
	targetFloorChannel := make(chan int, 10)
	stateChangedChannel := make(chan ElevatorData_t, 10)


	//resendStateTimer := time.NewTicker(500 * time.Millisecond)

	// The state handler is the goroutine that holds the state machine of the elevator
	go stateHandler(stateChangedChannel, currentFloorChannel, targetFloorChannel, floorCompletedChannel)

	for {
		select {
		case newFloorSignal := <-floorSignalChannel:
			if newFloorSignal != -1 {
				floor = newFloorSignal
				currentFloorChannel <- floor
			}
		case targetFloor = <-newTargetFloorChannel:
			targetFloorChannel <- targetFloor
		case elevatorData = <-stateChangedChannel:
			distributeStateChannel <- elevatorData
			fmt.Println("SENT STATE")
		case <- time.After(500 * time.Millisecond):
			distributeStateChannel <- elevatorData
		}
	}
}

func stateHandler(	stateChangedChannel chan<- ElevatorData_t,
					currentFloorChannel,
					targetFloorChannel <-chan int,
					floorCompletedChannel chan<- int) {
	var elevatorData ElevatorData_t
	betweenFloorsTimer := time.NewTimer(TimeBetweenFloors)
	betweenFloorsTimer.Stop()
	elevatorData.State = Idle
	stateChangedChannel <- elevatorData
	targetFloor := targetFloorReached
	var direction driver.MotorDirection


	stateChan := make(chan ElevatorData_t, 3)

	go func() {
		for {
			select {
			case elevatorData.Floor = <-currentFloorChannel:
				stateChangedChannel <- elevatorData
				driver.SetFloorIndicator(elevatorData.Floor)
				fmt.Println("Floor:", elevatorData.Floor)
				betweenFloorsTimer.Reset(TimeBetweenFloors)

				stateChan <- elevatorData
			case targetFloor = <-targetFloorChannel:
				//stateChangedChannel <- elevatorData
				fmt.Println("New target floor:", targetFloor)
				stateChan <- elevatorData
			}
		}
	}()
	go func() {
		for {
			select {
			case state := <-stateChan:
				switch state.State {
				case Idle:
					betweenFloorsTimer.Stop()
					if targetFloor != targetFloorReached {
						if elevatorData.Floor < targetFloor {
							elevatorData.State = MovingUp
							stateChangedChannel <- elevatorData
							betweenFloorsTimer.Reset(TimeBetweenFloors)
						} else if elevatorData.Floor > targetFloor {
							elevatorData.State = MovingDown
							stateChangedChannel <- elevatorData
							betweenFloorsTimer.Reset(TimeBetweenFloors)
						} else if elevatorData.Floor == targetFloor {
							elevatorData.State = DoorOpen
							stateChangedChannel <- elevatorData
						}
						stateChan <- elevatorData
					}

				case MovingUp:
					if direction != driver.DirectionUp {
						betweenFloorsTimer.Reset(TimeBetweenFloors)
						driver.SetMotorDirection(driver.DirectionUp)
						direction = driver.DirectionUp
						fmt.Println("State: Moving up")
						stateChan <- elevatorData
					}

					if elevatorData.Floor == targetFloor {
						betweenFloorsTimer.Stop()
						targetFloor = targetFloorReached
						driver.SetMotorDirection(driver.DirectionStop)
						direction = driver.DirectionStop
						elevatorData.State = DoorOpen
						stateChangedChannel <- elevatorData
						stateChan <- elevatorData
					}

				case MovingDown:
					if direction != driver.DirectionDown {
						betweenFloorsTimer.Reset(TimeBetweenFloors)
						driver.SetMotorDirection(driver.DirectionDown)
						direction = driver.DirectionDown
						fmt.Println("State: Moving down")
						stateChan <- elevatorData
					}

					if elevatorData.Floor == targetFloor {
						betweenFloorsTimer.Stop()
						targetFloor = targetFloorReached
						driver.SetMotorDirection(driver.DirectionStop)
						direction = driver.DirectionStop
						elevatorData.State = DoorOpen
						stateChangedChannel <- elevatorData
						stateChan <- elevatorData
					}

				case DoorOpen:
					betweenFloorsTimer.Stop()
					targetFloor = targetFloorReached
					driver.SetDoorOpenLamp(1)
					fmt.Println("State: Door open")
					time.Sleep(doorOpenTime * time.Second)
					driver.SetDoorOpenLamp(0)
					fmt.Println("Door closed")

					if targetFloor != targetFloorReached {
						if elevatorData.Floor < targetFloor {
							elevatorData.State = MovingUp
						} else if elevatorData.Floor > targetFloor {
							elevatorData.State = MovingDown
						} else if elevatorData.Floor == targetFloor {
							elevatorData.State = DoorOpen
						} else {
							elevatorData.State = Idle
						}
					} else {
						elevatorData.State = Idle
					}
					floorCompletedChannel <- elevatorData.Floor
					stateChangedChannel <- elevatorData
					stateChan <- elevatorData
				case Stuck:
					driver.SetMotorDirection(driver.DirectionStop)
					stateChangedChannel <- elevatorData
				}
			case <-betweenFloorsTimer.C:
				// Timed out between floors -> probably stuck
				elevatorData.State = Stuck
				stateChan <- elevatorData
				fmt.Println("Elevator timed out between floors.")
			}
		}
	}()
}

// Function to update state of other elevators
func UpdatePeerState(	allElevatorStates []ElevatorData_t,
						state ElevatorData_t) []ElevatorData_t {
	var stateExists bool
	for key, data := range allElevatorStates {
		if state.Id == data.Id {
			stateExists = true
			allElevatorStates[key] = state
		}
	}
	if !stateExists && len(state.Id) > 3 {
		allElevatorStates = append(allElevatorStates, state)
		fmt.Println("Added state for", state.Id, state)
	}
	return allElevatorStates
}
