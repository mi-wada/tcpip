package main

import (
	"fmt"

	"github.com/mi-wada/tcpip/net"
)

// Run server: nc -l localhost 8080
func main() {
	conn, err := net.Dial("tcp", "localhost")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	conn.Write([]byte("ping"))
	buf := make([]byte, 4)
	n, err := conn.Read(buf)
	if err != nil {
		panic(err)
	}
	if string(buf[:n]) == "pong" {
		fmt.Println("Received pong")
	} else {
		fmt.Printf("Received invalid: %s\n", string(buf[:n]))
	}
}
