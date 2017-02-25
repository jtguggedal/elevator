package network

import (
	fsm "./../state_machine"
	"./network/bcast"
	"./network/localip"
	"./network/peers"
	//"./../order_handler"
	//"flag"
	"fmt"
	//"net"
	"os"
	"time"
)

const (
	peerPort  = 32003
	bcastPort = 33003
)

// We define some custom struct to send over the network.
// Note that all members we want to transmit must be public. Any private members
//  will be received as zero-values.
type HelloMsg struct {
	Message string
	Iter    int
}

type UDPmessageType int

const (
	MsgState			= iota
	MsgNewOrder
	MsgFinishedOrder
)

type UDPmessage struct {
	Type 	UDPmessageType
	Data    []byte
}

/*
func UDPinit(txChannel, rxChannel chan UDPmessage) {

	var id string
	var connectedPeers peers.PeerUpdate
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()

	// We make a channel for receiving updates on the id's of the peers that are
	//  alive on the network
	peerUpdateCh := make(chan peers.PeerUpdate)

	// We can disable/enable the transmitter after it has been started.
	// This could be used to signal that we are somehow "unavailable".
	peerTxEnable := make(chan bool)
	go peers.Transmitter(peerPort, id, peerTxEnable)
	go peers.Receiver(peerPort, peerUpdateCh)

	// We make channels for sending and receiving our custom data types
	stateTx := make(chan fsm.StateMsg)
	stateRx := make(chan fsm.StateMsg)

	// ... and start the transmitter/receiver pair on some port
	// These functions can take any number of channels! It is also possible to
	//  start multiple transmitters/receivers on the same port.
	go bcast.Transmitter(bcastPort, stateTx)
	go bcast.Receiver(bcastPort, stateRx)

}*/

func UDPinit(	id string,
				stateRxChannel,
				stateTxChannel chan fsm.StateMsg,
				orderRxChannel,
				orderTxChannel chan UDPmessage,
				rxChannel chan UDPmessage,
				txChannel chan UDPmessage) {

	// Our id can be anything. Here we pass it on the command line, using
	//  `go run main.go -id=our_id`
	//var id string
	var connectedPeers peers.PeerUpdate
	/*flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()*/

	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}

	states := make([]fsm.ElevatorData, 0)

	// We make a channel for receiving updates on the id's of the peers that are
	//  alive on the network
	peerUpdateCh := make(chan peers.PeerUpdate)

	// We can disable/enable the transmitter after it has been started.
	// This could be used to signal that we are somehow "unavailable".
	peerTxEnable := make(chan bool)
	go peers.Transmitter(peerPort, id, peerTxEnable)
	go peers.Receiver(peerPort, peerUpdateCh)

	// We make channels for sending and receiving our custom data types
	// ... and start the transmitter/receiver pair on some port
	// These functions can take any number of channels! It is also possible to
	//  start multiple transmitters/receivers on the same port.
	go bcast.Transmitter(bcastPort, txChannel)
	go bcast.Receiver(bcastPort, rxChannel)


	timerChan := make(chan peers.PeerUpdate)
	startChan := make(chan bool)



	for {
		select {
		case p := <-peerUpdateCh:
			connectedPeers = p
			fmt.Printf("%02d:%02d:%02d.%03d - Peer update:\n", time.Now().Hour(), time.Now().Minute(), time.Now().Second(), time.Now().UnixNano()%1e6/1e3)
			if len(p.New) > 0 {
				var temp fsm.ElevatorData
				temp.Id = p.New
				fmt.Printf("  New:\t\t%q\n", p.New)
				states = append(states, temp)
				fmt.Println("States: ", states)
			}
			if len(p.Lost) > 0 {
				fmt.Printf("  Lost:\t\t%q\n", p.Lost)
				var temp []fsm.ElevatorData
				for _, element := range states {
					if element.Id != p.Lost[0] {
						temp = append(temp, element)
					}
				}
				states = temp
				fmt.Println("States: ", states)
			}
			fmt.Printf("  Connected:\t%q\n\n", connectedPeers.Peers)
		case order := <-orderTxChannel:
			txChannel <- order
		case done := <-timerChan:
			fmt.Printf("Connected nodes:    %q\n", done.Peers)
			//startChan <- true
		/*case a := <-stateRxChannel:

		// Passing on state changed to handler function in fsm pacakge
		fsm.StateChange(states, a)
		//fmt.Printf("States: %#v\n", states)*/
		case start := <-startChan:
			if start == true {
				fmt.Println("Elevators started. Have a nice day.")
			}
		}
	}

}
