package main

import (
	"github.com/mi-wada/tcpip/net"
)

// Run server: nc -u -l 8081
func main() {
	conn, err := net.Dial("udp", "localhost:8081")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte("hi\n"))
	if err != nil {
		panic(err)
	}
}
