package fsm

import (
    "./../driver"
    "./../network"
    "time"
    "fmt"
)

type state_t int

const (
    Idle state_t = 0
    MovingUp state_t = state_t(driver.DirectionUp)
    MovingDown state_t = state_t(driver.DirectionDown)
    DoorOpen state_t = 3
)

const doorOpenTime = 3
const targetFloorReached = -1

type ElevatorData_t struct {
    Id      network.Ip
    State   state_t
    Floor   int
}

var elevatorData ElevatorData_t

func GetElevatorData() ElevatorData_t {
	return elevatorData
}

func Init(  floorSignalChannel <-chan int,
            newTargetFloorChannel <-chan int,
            floorCompletedChannel chan<- int,
            distributeStateChannel chan<- ElevatorData_t,
            resendStateChannel <-chan bool) {

    var floor int
    var targetFloor int


    currentFloorChannel := make(chan int)
    targetFloorChannel := make(chan int)
    stateChangedChannel := make(chan ElevatorData_t)

    go stateHandler(stateChangedChannel, currentFloorChannel, targetFloorChannel, floorCompletedChannel)

    for {
        select {
        case newFloorSignal := <- floorSignalChannel:
            if newFloorSignal != -1 {
                floor = newFloorSignal
                currentFloorChannel <- floor
            }
        case targetFloor = <- newTargetFloorChannel:
            targetFloorChannel <- targetFloor
        case elevatorData = <- stateChangedChannel:
            distributeStateChannel <- elevatorData
        case <- resendStateChannel:
            go func() {
                time.Sleep(1 * time.Second)
                distributeStateChannel <- elevatorData
            }()

        }
    }
}


func stateHandler(  stateChangedChannel chan<- ElevatorData_t,
                    currentFloorChannel,
                    targetFloorChannel <-chan int,
                    floorCompletedChannel chan<- int) {
    var elevatorData ElevatorData_t
    elevatorData.State = Idle
    stateChangedChannel <- elevatorData
    targetFloor := targetFloorReached
    var direction driver.MotorDirection
    for {
        select {
        case elevatorData.Floor = <- currentFloorChannel:
            stateChangedChannel <- elevatorData
            driver.SetFloorIndicator(elevatorData.Floor)
            fmt.Println("Floor:", elevatorData.Floor)
        case targetFloor = <-targetFloorChannel:
            stateChangedChannel <- elevatorData
            fmt.Println("New target floor:", targetFloor)
        default:
            // Nothing to see here...
        }

        switch elevatorData.State {
        case Idle:
            if targetFloor != targetFloorReached {
                if elevatorData.Floor < targetFloor {
                    elevatorData.State = MovingUp
                    stateChangedChannel <- elevatorData
                }  else if elevatorData.Floor > targetFloor {
                    elevatorData.State = MovingDown
                    stateChangedChannel <- elevatorData
                } else if elevatorData.Floor == targetFloor {
                    elevatorData.State = DoorOpen
                    stateChangedChannel <- elevatorData
                }
            }

        case MovingUp:
            if direction != driver.DirectionUp {
                driver.SetMotorDirection(driver.DirectionUp)
                direction = driver.DirectionUp
                fmt.Println("State: Moving up")
            }

            if elevatorData.Floor == targetFloor {
                targetFloor = targetFloorReached
                driver.SetMotorDirection(driver.DirectionStop)
                direction = driver.DirectionStop
                elevatorData.State = DoorOpen
                stateChangedChannel <- elevatorData
            }

        case MovingDown:
            if direction != driver.DirectionDown {
                driver.SetMotorDirection(driver.DirectionDown)
                direction = driver.DirectionDown
                fmt.Println("State: Moving down")
            }

            if elevatorData.Floor == targetFloor {
                targetFloor = targetFloorReached
                driver.SetMotorDirection(driver.DirectionStop)
                direction = driver.DirectionStop
                elevatorData.State = DoorOpen
                stateChangedChannel <- elevatorData
            }

        case DoorOpen:
            targetFloor = targetFloorReached
            driver.SetDoorOpenLamp(1)
            fmt.Println("State: Door open")
            time.Sleep(doorOpenTime * time.Second)
            driver.SetDoorOpenLamp(0)
            fmt.Println("Door closed")

            if targetFloor != targetFloorReached  {
                if elevatorData.Floor < targetFloor {
                    elevatorData.State = MovingUp
                    stateChangedChannel <- elevatorData
                }  else if elevatorData.Floor > targetFloor {
                    elevatorData.State = MovingDown
                    stateChangedChannel <- elevatorData
                } else if elevatorData.Floor == targetFloor {
                    elevatorData.State = DoorOpen
                    stateChangedChannel <- elevatorData
                } else {
                    elevatorData.State = Idle
                    stateChangedChannel <- elevatorData
                }
            } else {
                elevatorData.State = Idle
                stateChangedChannel <- elevatorData
            }

            floorCompletedChannel <- elevatorData.Floor
        }
    }
}
