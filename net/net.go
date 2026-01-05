package net

import "io"

type Conn struct{}

var _ io.ReadWriteCloser = &Conn{}

func (c *Conn) Read(b []byte) (int, error) {
	panic("TODO: implement later")
}

func (c *Conn) Write(b []byte) (int, error) {
	panic("TODO: implement later")
}

func (c *Conn) Close() error {
	panic("TODO: implement later")
}

func Dial(network, address string) (Conn, error) {
	panic("TODO: implement later")
}
