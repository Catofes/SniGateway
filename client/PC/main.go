package main

import (
	"github.com/Catofes/SniGateway/client"
	"math/rand"
	"github.com/Catofes/SniGateway/go/src/fmt"
)

func main() {
	(&TLSClient.TLSClient{}).Init().Listen()
}

func main() {
	queueSize := 10
	queue := make(chan int, queueSize)

	//producer
	for i := 0; i < 3; i++ {
		go func(queue chan<- int) {
			data := rand.New(nil).Int()
			queue <- data
		}(queue)
	}

	//customer
	for i := 0; i < 3; i++ {
		go func(queue <-chan int) {
			data := <-queue
			fmt.Print(data)
		}(queue)
	}

}
