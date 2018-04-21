all:
	mkdir -p build
	go build -o build/SniGateway github.com/Catofes/SniGateway/gateway
	go build -o build/TLSServer github.com/Catofes/SniGateway/server
	go build -o build/TLSClient github.com/Catofes/SniGateway/client/PC
	go build -o build/ProxyClient github.com/Catofes/SniGateway/proxy/PC
	go build -o build/TencentProxyClient github.com/Catofes/SniGateway/tencentProxy/PC
android:
	bash make.sh 26 client
	bash make.sh 26 proxy
	bash make.sh 26 tencentProxy
