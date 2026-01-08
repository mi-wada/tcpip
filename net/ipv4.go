package net

type IPV4Conn struct {
	ethernetConn *EthernetConn
}

var _ Conn = &IPV4Conn{}

func NewIPV4Conn(address string) (*IPV4Conn, error) {
	panic("TODO: implement later")
	// ethernetConn, err := NewEthernetConn(address)
	// if err != nil {
	// 	return nil, err
	// }
	// return &IPV4Conn{
	// 	ethernetConn: ethernetConn,
	// }, nil
}

func (c *IPV4Conn) Read(p []byte) (n int, err error) {
	panic("not implemented") // TODO: Implement
}

func (c *IPV4Conn) Write(p []byte) (n int, err error) {
	panic("not implemented") // TODO: Implement
}

func (c *IPV4Conn) Close() error {
	return c.ethernetConn.Close()
}
