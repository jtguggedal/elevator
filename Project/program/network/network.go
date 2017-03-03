package network

import (
	"./network/bcast"
	"./network/localip"
	"./network/peers"
	"fmt"
	"os"
)

const (
	peerPort  = 32151
	bcastPort = 32152
	statePort = 32153
	orderPort = 32154
)

const (
	MsgState			= iota
	MsgNewOrder
	MsgFinishedOrder
)

// We define some custom struct to send over the network.
// Note that all members we want to transmit must be public. Any private members
//  will be received as zero-values.
type HelloMsg struct {
	Message string
	Iter    int
}

type Ip string

type PeerStatus peers.PeerUpdate

type UDPmessageType int


type UDPmessage struct {
	Type 	UDPmessageType
	Data    []byte
}

func UDPinit(	id string,
				ipChannel chan<- Ip,
				stateRxChannel,
				stateTxChannel chan UDPmessage,
				orderRxChannel,
				orderTxChannel chan UDPmessage,
				rxChannel chan UDPmessage,
				txChannel chan UDPmessage,
				peerStatusChannel chan PeerStatus,
				orderDoneRxChannel chan UDPmessage,
				orderDoneTxChannel chan UDPmessage) {

	// Our id can be anything. Here we pass it on the command line, using
	//  `go run main.go -id=our_id`
	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}

	ipChannel <- Ip(id)


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
	go bcast.Transmitter(statePort, stateTxChannel)
	go bcast.Transmitter(orderPort, orderDoneTxChannel)
	go bcast.Receiver(bcastPort, rxChannel)
	go bcast.Receiver(statePort, stateRxChannel)
	go bcast.Receiver(orderPort, orderDoneRxChannel)

	for {
		select {
		case peers := <-peerUpdateCh:
			peerStatusChannel <- PeerStatus(peers)
		case order := <-orderTxChannel:
			txChannel <- order
		}
	}

}
