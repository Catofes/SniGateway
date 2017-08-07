all:
	mkdir -p build
	go build -o build/SniGateway github.com/Catofes/SniGateway/gateway
	go build -o build/TLSServer github.com/Catofes/SniGateway/server
	go build -o build/TLSClient github.com/Catofes/SniGateway/client/PC
android:
	mkdir -p build
	gomobile build -target=android -o build/TLSClient.apk github.com/Catofes/SniGateway/client/Android
