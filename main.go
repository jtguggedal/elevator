package main

import "fmt"
import "net"

func main() {
	conn, err := net.Dial("tcp", "google.com:80")
	if err == nil {
		conn.Close()
		fmt.Println("Connected to network")
	} else {
		fmt.Println("No network")
	}
}
