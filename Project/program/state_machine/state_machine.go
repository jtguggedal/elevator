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
type peerType string

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
			floorReachedChannel chan<- int,
			peerStatusChannel <-chan network.PeerStatus,
			livePeersChannel chan []string) {

	var livePeers []string
	var elevatorStates map[peerType]ElevatorData
	stateUpdateLocal := make(chan ElevatorData)

	go peerMonitor(peerStatusChannel, livePeersChannel)
	go floorMonitor(floorEventChannel, floorReachedChannel)
	go stateMonitor(stateRx, stateTx, stateUpdateLocal)


	for {
		select {
		case livePeers = <- livePeersChannel:
			fmt.Println("LIVE PEER UPDATE:", len(livePeers))

		case stateUpdate := <- stateRx:
			fmt.Println("STATE UPDATE FROM SFM:", elevatorStates, stateUpdate)

		}
	}
}

func floorMonitor(floorEventChannel <-chan int, floorReachedChannel chan<- int) {
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

func stateMonitor(stateRx <-chan network.UDPmessage, stateTx chan<- network.UDPmessage, stateUpdateLocal chan<- ElevatorData) {
	var states []ElevatorData
	for {
		select {
		case msg := <- stateRx:
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

func peerMonitor(peerStatusChannel <-chan network.PeerStatus, livePeersChannel chan<- []string) {
	for {
		select {
		case update := <- peerStatusChannel:
			var peerList []string
			for _, peer := range update.Peers {
				peerList = append(peerList, peer)
			}
			livePeersChannel <- peerList
		}
	}
}


func DoorOpen() {
	time.Sleep(doorOpenTime * time.Second)
}
