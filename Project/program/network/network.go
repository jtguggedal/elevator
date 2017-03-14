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

type ChannelPair struct {
	Rx 	chan UDPmessage
	Tx 	chan UDPmessage
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
				peerUpdateChan chan peers.PeerUpdate,
				orderChannels,
				orderDoneChannels,
				stateChannels ChannelPair) {

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

	go bcast.Receiver(orderPort, orderChannels.Rx)
	go bcast.Receiver(orderDonePort, orderDoneChannels.Rx)
	go bcast.Receiver(statePort, stateChannels.Rx)
	go bcast.Transmitter(orderDonePort, orderDoneChannels.Tx)
	go bcast.Transmitter(orderPort, orderChannels.Tx)
	go bcast.Transmitter(statePort, stateChannels.Tx)

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
