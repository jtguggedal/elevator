package main

import (
    "fmt"
	//"net"
	//"os"
	"os/exec"
	//"strconv"
	//"time"
    "flag"
)



type Counter struct {
	Count int
}

type UDPmessage struct {
	Data string
}


func listenUDP(listenChan chan UDPmessage) {
    //listenAddr, err := net.ResolveUDPAddr("udp", ":30031")

    // No error handling for now
}



func startBackup() {
    //arg := fmt.Sprintf("go run main.go %d", initCounter.State)

	//cmd := exec.Command("gnome-terminal", "-x", "sh", "-c", "go run main.go -backup")
    cmd := exec.Command("osascript -e 'tell app "Terminal" to do script ["terminal command"]")
	cmd.Run()
}

var backupFlag = flag.Bool("backup", false, "Is backup")


func main() {
    if(!*backupFlag) {
        startBackup()
        fmt.Println("I'm master")
    } else {
        fmt.Println("I'm backup")
    }

}
