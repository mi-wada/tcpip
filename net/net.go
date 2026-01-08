package net

import "io"

type Conn interface {
	io.ReadWriteCloser
}

const (
	networkTCP      = "tcp"
	networkUDP      = "udp"
	networkIPV4     = "ip4"
	networkEthernet = "ethernet"
)

func Dial(network, address string) (Conn, error) {
	switch network {
	case networkTCP:
	case networkUDP:
	case networkIPV4:
		return NewIPV4Conn(address)
	}
	panic("TODO: implement later")
}
