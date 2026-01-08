package icmp

import (
	"encoding/binary"
	"fmt"
)

func Handle(payload []byte) ([]byte, error) {
	p, err := Unmarshal(payload)
	if err != nil {
		return nil, err
	}
	reply := Packet{
		Type: TypeEchoReply,
		Code: 0,
		ID:   p.ID,
		Seq:  p.Seq,
		Data: p.Data,
	}
	return reply.Marshal(), nil
}

// Echo or Echo Reply Message
type Packet struct {
	Type     uint8
	Code     uint8
	Checksum uint16
	ID       uint16
	Seq      uint16
	Data     []byte
}

const (
	TypeEchoReply = 0
	TypeEcho      = 8
)

func (p Packet) Marshal() []byte {
	buf := make([]byte, 8+len(p.Data))
	buf[0] = p.Type
	buf[1] = p.Code
	binary.BigEndian.PutUint16(buf[2:4], 0)
	binary.BigEndian.PutUint16(buf[4:6], p.ID)
	binary.BigEndian.PutUint16(buf[6:8], p.Seq)
	copy(buf[8:], p.Data)

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

func Unmarshal(payload []byte) (Packet, error) {
	if len(payload) < 8 {
		return Packet{}, fmt.Errorf("ICMP packet too short: %d", len(payload))
	}

	return Packet{
		Type:     payload[0],
		Code:     payload[1],
		Checksum: binary.BigEndian.Uint16(payload[2:4]),
		ID:       binary.BigEndian.Uint16(payload[4:6]),
		Seq:      binary.BigEndian.Uint16(payload[6:8]),
		Data:     payload[8:],
	}, nil
}
