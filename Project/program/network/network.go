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
	peerPort  = 32151
	bcastPort = 32152
	statePort = 32153
	orderPort = 32154
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

func UDPinit(receivedId string,
	stateRxChannel,
	stateTxChannel chan UDPmessage,
	orderRxChannel chan UDPmessage,
	rxChannel chan UDPmessage,
	txChannel chan UDPmessage,
	peerUpdateChannel chan peers.PeerUpdate,
	orderDoneRxChannel chan UDPmessage,
	orderDoneTxChannel chan UDPmessage) {

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
	go peers.Receiver(peerPort, peerUpdateChannel)

	go bcast.Transmitter(bcastPort, txChannel)
	go bcast.Transmitter(statePort, stateTxChannel)
	go bcast.Transmitter(orderPort, orderDoneTxChannel)
	go bcast.Receiver(bcastPort, rxChannel)
	go bcast.Receiver(statePort, stateRxChannel)
	go bcast.Receiver(orderPort, orderDoneRxChannel)

}

func GetLocalId() Ip {
	return Ip(id)
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
