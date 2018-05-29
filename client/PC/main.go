package main

import (
	"github.com/Catofes/SniGateway/client"
)

func main() {
	(&TLSClient.TLSClient{}).Init().Listen()
}
