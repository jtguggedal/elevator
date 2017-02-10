package main

import (
	"./network/bcast"
	"./network/localip"
	"./network/peers"
	"flag"
	"fmt"
	"os"
	"time"
	"net"
)

// We define some custom struct to send over the network.
// Note that all members we want to transmit must be public. Any private members
//  will be received as zero-values.
type HelloMsg struct {
	Message string
	Iter    int
}

func main() {




	// Our id can be anything. Here we pass it on the command line, using
	//  `go run main.go -id=our_id`
	var id string
	var connectedPeers peers.PeerUpdate
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()

	// ... or alternatively, we can use the local IP address.
	// (But since we can run multiple programs on the same PC, we also append the
	//  process ID)
	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}

	// We make a channel for receiving updates on the id's of the peers that are
	//  alive on the network
	peerUpdateCh := make(chan peers.PeerUpdate)
	// We can disable/enable the transmitter after it has been started.
	// This could be used to signal that we are somehow "unavailable".
	peerTxEnable := make(chan bool)
	go peers.Transmitter(30003, id, peerTxEnable)
	go peers.Receiver(30003, peerUpdateCh)

	// We make channels for sending and receiving our custom data types
	helloTx := make(chan HelloMsg)
	helloRx := make(chan HelloMsg)
	// ... and start the transmitter/receiver pair on some port
	// These functions can take any number of channels! It is also possible to
	//  start multiple transmitters/receivers on the same port.
	go bcast.Transmitter(31003, helloTx)
	go bcast.Receiver(31003, helloRx)

	// The example message. We just send one of these every second.
	/*go func() {
		helloMsg := HelloMsg{"Hello from " + id, 0}
		for {
			helloMsg.Iter++
			helloTx <- helloMsg
			time.Sleep(1 * time.Second)
		}
	}()*/


	timerChan := make(chan peers.PeerUpdate)
	startChan := make(chan bool)
	// Check after 10 seconds which nodes are online
	go func() {
		for {
			time.Sleep(3 * time.Second)
			//timerChan <- connectedPeers
		}
	}()

	fmt.Println("Started")
	for {
		select {
		case p := <-peerUpdateCh:
			connectedPeers = p
			fmt.Printf("%02d:%02d:%02d.%03d - Peer update:\n", time.Now().Hour(), time.Now().Minute(), time.Now().Second(), time.Now().UnixNano()%1e6/1e3)
			if len(p.New) > 0 {
				fmt.Printf("  New:\t\t%q\n", p.New)
			}
			if len(p.Lost) > 0 {
				fmt.Printf("  Lost:\t\t%q\n", p.Lost)
			}
			fmt.Printf("  Connected:\t%q\n\n", connectedPeers.Peers)
			if len(p.Lost) > 0  && false {
				fmt.Println("LOST A NODE :((")
				fmt.Println("Attempting TCP connection...")
			    conn, _ := net.Dial("tcp", "192.168.1.8:8081")
			    fmt.Fprintf(conn, "tjäna grabben\n")
			}
		case done := <-timerChan:
			fmt.Printf("Connected nodes:    %q\n", done.Peers)
			//startChan <- true
		//case a := <-helloRx:
		//	fmt.Printf("Received: %#v\n", a)
		case start := <- startChan:
			if start == true {
				fmt.Println("Elevators started. Have a nice day.")
			}
		}
	}
}
