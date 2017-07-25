all:
	mkdir -p build
	go build -o build/SniGateway github.com/Catofes/SniGateway/gateway
	go build -o build/TLSServer github.com/Catofes/SniGateway/server
	go build -o build/TLSClient github.com/Catofes/SniGateway/client
