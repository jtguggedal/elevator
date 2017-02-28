package fsm

import (
    "./../driver"
    "time"
    "fmt"
)

type state_t int

const (
    idle = iota
    movingUp
    movingDown
    doorOpen
)

const doorOpenTime = 3
const targetFloorReached = -1


func Init(floorSignalChannel <-chan int, newTargetFloorChannel <-chan int) {

    var floor int
    var targetFloor int

    currentFloorChannel := make(chan int)
    targetFloorChannel := make(chan int)

    // initialize to known floor somehow
    go stateHandler(currentFloorChannel, targetFloorChannel)

    for {
        select {
        case newFloorSignal := <- floorSignalChannel:
            if newFloorSignal != -1 {
                floor = newFloorSignal
                currentFloorChannel <- floor
                fmt.Println("Floor: ", floor)
            }
        case targetFloor = <- newTargetFloorChannel:
            targetFloorChannel <- targetFloor
            fmt.Println("New target: ", targetFloor)
        }
    }
}


func stateHandler(currentFloorChannel, targetFloorChannel <-chan int) {
    state := idle
    var current int
    target := targetFloorReached
    var direction driver.MotorDirection
    for {
        select {
        case current = <- currentFloorChannel:
        case target = <-targetFloorChannel:
        default:
            // Nothing to see here...
        }

        switch state {
        case idle:
            if target != targetFloorReached {
                if current < target {
                    state = movingUp
                }  else if current > target {
                    state = movingDown
                } else if current == target {
                    state = doorOpen
                }
            }

        case movingUp:
            if direction != driver.DirectionUp {
                driver.SetMotorDirection(driver.DirectionUp)
                direction = driver.DirectionUp
                fmt.Println("State: Moving up")
            }

            if current == target {
                target = targetFloorReached
                driver.SetMotorDirection(driver.DirectionStop)
                direction = driver.DirectionStop
                state = doorOpen
            }

        case movingDown:
            if direction != driver.DirectionDown {
                driver.SetMotorDirection(driver.DirectionDown)
                direction = driver.DirectionDown
                fmt.Println("State: Moving down")
            }

            if current == target {
                target = targetFloorReached
                driver.SetMotorDirection(driver.DirectionStop)
                direction = driver.DirectionStop
                state = doorOpen
            }

        case doorOpen:
            target = targetFloorReached
            driver.SetDoorOpenLamp(1)
            fmt.Println("State: Door open")
            time.Sleep(doorOpenTime * time.Second)
            driver.SetDoorOpenLamp(0)
            fmt.Println("Door closed")

            if target != targetFloorReached {
                if current < target {
                    state = movingUp
                }  else if current > target {
                    state = movingDown
                } else if current == target {
                    state = doorOpen
                } else {
                    state = idle
                }
            } else {
                state = idle
            }
        }
    }
}
