package state_machine

import (
	"./../driver"
	"./../network"
	"fmt"
	"encoding/json"
	"time"
)

type elevatorDirection driver.MotorDirection
type elevatorState int

const (
	idle elevatorState = iota
	moving
	doorOpen
)

const doorOpenTime = 1

type ElevatorData struct {
	Id        	string
	Direction 	elevatorDirection
	Floor     	int
	State		elevatorState
}

type StateMsg ElevatorData

func Init(	stateRx, stateTx chan network.UDPmessage,
			floorEventChannel <-chan int,
			floorReachedChannel chan<- int /*, states []State*/) {

	go FloorMonitor(floorEventChannel, floorReachedChannel)
	go StateMonitor(stateRx)
	/*for {
		select {
		case receivedState := <-stateRx:
			//var states []State
			//states = append(states, State(receivedState))
			//StateChange(states, receivedState)
			fmt.Println(receivedState)
		}
	}*/
}

func FloorMonitor(floorEventChannel <-chan int, floorReachedChannel chan<- int) {
	var floorSignal int
	for {
		select {
		case floorSignal = <-floorEventChannel:
			if floorSignal >= 0 {
				floorReachedChannel <- floorSignal
				if(floorSignal >= driver.NUMBER_OF_FLOORS) {
					driver.SetMotorDirection(driver.DirectionStop)
				}
				driver.SetFloorIndicator(floorSignal)
				fmt.Println("Arrived at floor", floorSignal)
			} else if floorSignal == -1 {
				fmt.Println("Moving...")
			}
		}
	}
}

func StateMonitor(stateChannel <-chan network.UDPmessage) {
	var states []ElevatorData
	for {
		select {
		case msg := <- stateChannel:
			var receivedState StateMsg
			json.Unmarshal(msg.Data, &receivedState)
			for i, element := range states {
				if element.Id == receivedState.Id {
					states[i].Direction = receivedState.Direction
					states[i].Floor = receivedState.Floor
				}
			}
		}
	}
	fmt.Printf("States: %#v\n", states)
}


func DoorOpen() {
	time.Sleep(doorOpenTime * time.Second)
}
