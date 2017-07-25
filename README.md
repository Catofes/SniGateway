## ZZSniGateway

This project is a simple SNI based TLS route program. You can bind SniGateway to your 443 port and forwart TLS connect to multi backend according to SNI part. In this way you can hide some tcp traffic under normal https traffic. 

**Noticei:** SNI do not be encrypted in TLS. MITM can distinguish the traffic if they want to.


