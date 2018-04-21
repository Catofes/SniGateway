package main

import (
	"github.com/Catofes/SniGateway/tencentProxy"
)

func main() {
	(&ProxyClient.ProxyClient{}).Init().Listen()
}
