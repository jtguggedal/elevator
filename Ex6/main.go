package main

import (
    "fmt"
	"net"
	//"os"
	"os/exec"
	"strconv"
	"time"
    "flag"
    "log"
)



var masterFlag bool
var portOffset int = 0

type Counter struct {
	Count int
}

type UDPmessage struct {
	Data string
}


func listenUDP(listenChannel chan Counter) {
	port := fmt.Sprintf(":3333%d", portOffset)
    listenAddr, err := net.ResolveUDPAddr("udp", port)
    if err != nil {
		log.Fatal(err)
	}


	conn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}

	for {
		if masterFlag  {
			conn.Close()
			break
		}
		buffer := make([]byte, 1024)
		n, _, _ := conn.ReadFromUDP(buffer)

		var counter Counter
		counter.Count, _ = strconv.Atoi(string(buffer[:n]))

		listenChannel <- counter
	}
    // No error handling for now
}

	
func broadcastServer(outputChannel chan UDPmessage) {

	// Setting up ports. Note, no error handling implemented
	listenAddr, _ := net.ResolveUDPAddr("udp", ":30031")


	addr := fmt.Sprintf("255.255.255.255:3333%d", portOffset)
	broadcastAddr, _ := net.ResolveUDPAddr("udp", addr)

	conn, _ := net.ListenUDP("udp", listenAddr)

	for {
		msg := <- outputChannel
		conn.WriteToUDP([]byte(msg.Data), broadcastAddr)
	}
}



func startBackup() {
	expression := fmt.Sprintf("go run main.go -backup -portOffset=%d", portOffset)
	cmd := exec.Command("gnome-terminal", "-x", "sh", "-c", expression)
    //cmd := exec.Command("osascript -e 'tell app "Terminal" to do script ["terminal command"]")
	cmd.Run()
}

func becomeMaster(outputChannel chan UDPmessage) {
	masterFlag = true
    startBackup()
	go broadcastServer(outputChannel)

    fmt.Println("I'm master")
}

func becomeBackup(listenChannel chan Counter, receivedPortOffset int) {
	masterFlag = false
	portOffset = receivedPortOffset
	go listenUDP(listenChannel)
    fmt.Println("I'm backup")
}


func main() {
	var backupFlag = flag.Bool("backup", false, "Is backup")
	var receivedPortOffset = flag.Int("portOffset", 0, "Port offset")
	flag.Parse()

    outputChannel := make(chan UDPmessage)

    listenChannel := make(chan Counter)


	// Checking if master or backup
    if !*backupFlag {
    	becomeMaster(outputChannel)
    } else {
    	becomeBackup(listenChannel, *receivedPortOffset)
    }


	counter := Counter{0}

	
	for {
		if masterFlag {
			fmt.Printf("Count: %d\n", counter.Count)

			outputChannel <- UDPmessage{strconv.Itoa(counter.Count)}

			counter.Count++
			time.Sleep(1 * time.Second)
		} else {
			select {
			case <-time.After(2 * time.Second):
				fmt.Println("Timer has time out - becoming master...")
   			 	portOffset++
   			 	counter.Count++
				becomeMaster(outputChannel)
			case msg := <- listenChannel:
				counter.Count = msg.Count
				fmt.Printf("Received count: %d\n", counter.Count)
			}
		}



	}

}
