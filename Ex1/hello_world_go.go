package main

import (
	"fmt"
	"runtime"
	"time"
)

var i int = 0

func inc(){
	for j := 0; j < 1000000; j++ {
    	i++
    }
}

func dec(){
	for j := 0; j < 1000000; j++ {
    	i--
    }
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	go inc()
	go dec()

	time.Sleep(100 * time.Millisecond)
	fmt.Printf("Result: %d\n", i)
}
