package icmp

import (
	"encoding/binary"
	"fmt"
)

func Handle(payload []byte) ([]byte, error) {
	p, err := unmarshal(payload)
	if err != nil {
		return nil, err
	}
	reply := packet{
		t:        typeEchoReply,
		code:     0,
		checksum: 0,
		id:       p.id,
		seq:      p.seq,
		data:     p.data,
	}
	return reply.marshal(), nil
}

// Echo or Echo Reply Message
type packet struct {
	t        uint8
	code     uint8
	checksum uint16
	id       uint16
	seq      uint16
	data     []byte
}

const (
	typeEchoReply = 0
	typeEcho      = 8
)

func (p packet) marshal() []byte {
	buf := make([]byte, 8+len(p.data))
	buf[0] = p.t
	buf[1] = p.code
	binary.BigEndian.PutUint16(buf[2:4], 0)
	binary.BigEndian.PutUint16(buf[4:6], p.id)
	binary.BigEndian.PutUint16(buf[6:8], p.seq)
	copy(buf[8:], p.data)

	binary.BigEndian.PutUint16(buf[2:4], checksum(buf))
	return buf
}

func checksum(buf []byte) uint16 {
	var sum uint32
	for i := 0; i < len(buf); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(buf[i : i+2]))
	}
	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum)
}

func unmarshal(payload []byte) (packet, error) {
	if len(payload) < 8 {
		return packet{}, fmt.Errorf("ICMP packet too short: %d", len(payload))
	}

	return packet{
		t:        payload[0],
		code:     payload[1],
		checksum: binary.BigEndian.Uint16(payload[2:4]),
		id:       binary.BigEndian.Uint16(payload[4:6]),
		seq:      binary.BigEndian.Uint16(payload[6:8]),
		data:     payload[8:],
	}, nil
}
