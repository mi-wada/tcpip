package udp

import (
	"encoding/binary"
	"fmt"
)

func Handle(payload []byte) ([]byte, error) {
	p, err := unmarshal(payload)
	if err != nil {
		return nil, err
	}
	// とりあえずEchoするぞ
	d := p.data
	reply := packet{
		srcPort:  p.dstPort,
		dstPort:  p.srcPort,
		length:   uint16(8 + len(d)),
		checksum: 0,
		data:     d,
	}
	return reply.marshal(), nil
}

type packet struct {
	srcPort  uint16
	dstPort  uint16
	length   uint16
	checksum uint16
	data     []byte
}

func (p packet) marshal() []byte {
	buf := make([]byte, p.length)
	binary.BigEndian.PutUint16(buf[0:2], p.srcPort)
	binary.BigEndian.PutUint16(buf[2:4], p.dstPort)
	binary.BigEndian.PutUint16(buf[4:6], p.length)
	binary.BigEndian.PutUint16(buf[6:8], 0)
	copy(buf[8:], p.data)

	binary.BigEndian.PutUint16(buf[6:8], checksum(buf))
	return buf
}

func checksum(buf []byte) uint16 {
	// TODO: ちゃんとchecksum計算する。IPアドレスなどを含める必要があって面倒なのでスキップした。
	return 0

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
		return packet{}, fmt.Errorf("UDP packet too short: %d", len(payload))
	}

	return packet{
		srcPort:  binary.BigEndian.Uint16(payload[0:2]),
		dstPort:  binary.BigEndian.Uint16(payload[2:4]),
		length:   binary.BigEndian.Uint16(payload[4:6]),
		checksum: binary.BigEndian.Uint16(payload[6:8]),
		data:     payload[8:],
	}, nil
}
