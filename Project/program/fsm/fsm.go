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
const sendStateInterval = 400000 * time.Millisecond

type ElevatorData_t struct {
    Id      network.Ip
    State   state_t
    Floor   int
}


func Init(  floorSignalChannel <-chan int,
            newTargetFloorChannel <-chan int,
            floorCompletedChannel chan<- int,
            distributeStateChannel chan<- ElevatorData_t) {

    var floor int
    var targetFloor int
    var elevatorData ElevatorData_t

    //sendDataTick := time.Tick(sendStateInterval)

    currentFloorChannel := make(chan int)
    targetFloorChannel := make(chan int)
    stateChangedChannel := make(chan ElevatorData_t)

    // initialize to known floor somehow
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
            fmt.Println("New state:", elevatorData)
            distributeStateChannel <- elevatorData
        }
    }
}


func stateHandler(  stateChangedChannel chan<- ElevatorData_t,
                    currentFloorChannel,
                    targetFloorChannel <-chan int,
                    floorCompletedChannel chan<- int) {
    var e ElevatorData_t
    e.State = Idle
    stateChangedChannel <- e
    targetFloor := targetFloorReached
    var direction driver.MotorDirection
    for {
        select {
        case e.Floor = <- currentFloorChannel:
            stateChangedChannel <- e
            driver.SetFloorIndicator(e.Floor)
            fmt.Println("Floor:", e.Floor)
        case targetFloor = <-targetFloorChannel:
            stateChangedChannel <- e
            fmt.Println("New target floor:", targetFloor)
        default:
            // Nothing to see here...
        }

        switch e.State {
        case Idle:
            if targetFloor != targetFloorReached {
                if e.Floor < targetFloor {
                    e.State = MovingUp
                    stateChangedChannel <- e
                }  else if e.Floor > targetFloor {
                    e.State = MovingDown
                    stateChangedChannel <- e
                } else if e.Floor == targetFloor {
                    e.State = DoorOpen
                    stateChangedChannel <- e
                }
            }

        case MovingUp:
            if direction != driver.DirectionUp {
                driver.SetMotorDirection(driver.DirectionUp)
                direction = driver.DirectionUp
                fmt.Println("State: Moving up")
            }

            if e.Floor == targetFloor {
                targetFloor = targetFloorReached
                driver.SetMotorDirection(driver.DirectionStop)
                direction = driver.DirectionStop
                e.State = DoorOpen
                stateChangedChannel <- e
            }

        case MovingDown:
            if direction != driver.DirectionDown {
                driver.SetMotorDirection(driver.DirectionDown)
                direction = driver.DirectionDown
                fmt.Println("State: Moving down")
            }

            if e.Floor == targetFloor {
                targetFloor = targetFloorReached
                driver.SetMotorDirection(driver.DirectionStop)
                direction = driver.DirectionStop
                e.State = DoorOpen
                stateChangedChannel <- e
            }

        case DoorOpen:
            floorCompletedChannel <- e.Floor
            targetFloor = targetFloorReached
            driver.SetDoorOpenLamp(1)
            fmt.Println("State: Door open")
            time.Sleep(doorOpenTime * time.Second)
            driver.SetDoorOpenLamp(0)
            fmt.Println("Door closed")

            if targetFloor != targetFloorReached {
                if e.Floor < targetFloor {
                    e.State = MovingUp
                    stateChangedChannel <- e
                }  else if e.Floor > targetFloor {
                    e.State = MovingDown
                    stateChangedChannel <- e
                } else if e.Floor == targetFloor {
                    e.State = DoorOpen
                    stateChangedChannel <- e
                } else {
                    e.State = Idle
                    stateChangedChannel <- e
                }
            } else {
                e.State = Idle
                stateChangedChannel <- e
            }
        }
    }
}
