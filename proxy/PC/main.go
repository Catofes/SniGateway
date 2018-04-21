package main

import (
	"github.com/Catofes/SniGateway/proxy"
)

func main() {
	(&ProxyClient.ProxyClient{}).Init().Listen()
}
