package main

import (
	"fmt"
	"runtime"
	"time"
)

var i int = 0

func inc(shared_value chan int, available chan bool){
	for j := 0; j < 1000000; j++ {
		counter := <- shared_value
		counter++
		i = counter
		shared_value <- counter
    }
	available <- true
}

func dec(shared_value chan int, available chan bool){
	for j := 0; j < 1000002; j++ {
		counter := <- shared_value
		counter--
		i = counter
		shared_value <- counter
    }
	available <- true
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	shared_value := make(chan int, 1)
	shared_value <- i

	available := make(chan bool, 1)

	go inc(shared_value, available)
	go dec(shared_value, available)

	<- available


	time.Sleep(100 * time.Millisecond)
	fmt.Printf("Result: %d\n", i)
}
