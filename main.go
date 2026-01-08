package main

import (
	"github.com/mi-wada/tcpip/net"
)

func main() {
	address := [4]byte{192, 168, 10, 1}
	ethernetConn, err := net.NewEthernetConn(address)
	if err != nil {
		panic(err)
	}
	defer ethernetConn.Close()
	if err := ethernetConn.RRR(); err != nil {
		panic(err)
	}
}
