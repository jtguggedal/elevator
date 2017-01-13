package main

import (
	"fmt"
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

	go inc()
	go dec()

	time.Sleep(1 * time.Second)
	fmt.Printf("Result: %d\n", i)
}
