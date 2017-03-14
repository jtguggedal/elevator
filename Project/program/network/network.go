package network

import (
	"fmt"
	"net"
	"os"

	"./bcast"
	"./localip"
	"./peers"
)

const (
	peerPort  		= 32155
	orderPort 		= 32152
	statePort 		= 32153
	orderDonePort 	= 32154
)

const (
	MsgState = iota
	MsgNewOrder
	MsgFinishedOrder
)

type HelloMsg struct {
	Message string
	Iter    int
}

type Ip string

type PeerStatus peers.PeerUpdate

type UDPmessageType int

type UDPmessage struct {
	Type UDPmessageType
	Data []byte
}

var id string

func UDPinit(	receivedId string,
				stateRxChan,
				stateTxChan chan UDPmessage,
				orderRxChan chan UDPmessage,
				orderTxChan chan UDPmessage,
				peerUpdateChan chan peers.PeerUpdate,
				orderDoneRxChan chan UDPmessage,
				orderDoneTxChan chan UDPmessage) {

	id = receivedId
	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}

	peerTxEnable := make(chan bool)

	go peers.Transmitter(peerPort, id, peerTxEnable)
	go peers.Receiver(peerPort, peerUpdateChan)

	go bcast.Receiver(orderPort, orderRxChan)
	go bcast.Receiver(statePort, stateRxChan)
	go bcast.Receiver(orderDonePort, orderDoneRxChan)
	go bcast.Transmitter(orderPort, orderTxChan)
	go bcast.Transmitter(statePort, stateTxChan)
	go bcast.Transmitter(orderDonePort, orderDoneTxChan)

}

func GetLocalId() string {
	return id
}

func IsConnected() bool {
	conn, err := net.Dial("tcp", "google.com:80")
	if err == nil {
		conn.Close()
		return true
	} else {
		return false
	}
}
