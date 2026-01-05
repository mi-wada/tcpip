package arp

import (
	"encoding/binary"
	"fmt"
)

func Handle(payload []byte) ([]byte, error) {
	p, err := unmarshal(payload)
	if err != nil {
		return nil, err
	}

	myIP := [4]byte{192, 168, 10, 2}
	if p.op != opRequest || p.tpa != myIP {
		return nil, nil
	}

	reply := packet{
		htype: 0x0001, // Ethernet
		ptype: 0x0800, // IPv4
		hlen:  6,
		plen:  4,
		op:    opReply,
		// // 02:00:00:00:00:02
		sha: [6]byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x02},
		spa: myIP,
		tha: p.sha,
		tpa: p.spa,
	}

	return marshal(reply), nil
}

const (
	opRequest uint16 = 1
	opReply   uint16 = 2
)

type packet struct {
	htype uint16
	ptype uint16
	hlen  byte
	plen  byte
	op    uint16
	sha   [6]byte
	spa   [4]byte
	tha   [6]byte
	tpa   [4]byte
}

func unmarshal(payload []byte) (packet, error) {
	// ARPパケット(IPv4 over Ethernet)は28バイト固定
	if len(payload) < 28 {
		return packet{}, fmt.Errorf("arp packet too short: %d", len(payload))
	}

	var p packet
	p.htype = binary.BigEndian.Uint16(payload[0:2])
	p.ptype = binary.BigEndian.Uint16(payload[2:4])
	p.hlen = payload[4]
	p.plen = payload[5]
	p.op = binary.BigEndian.Uint16(payload[6:8])
	copy(p.sha[:], payload[8:14])
	copy(p.spa[:], payload[14:18])
	copy(p.tha[:], payload[18:24])
	copy(p.tpa[:], payload[24:28])

	return p, nil
}

func marshal(p packet) []byte {
	buf := make([]byte, 28)

	binary.BigEndian.PutUint16(buf[0:2], p.htype)
	binary.BigEndian.PutUint16(buf[2:4], p.ptype)
	buf[4] = p.hlen
	buf[5] = p.plen
	binary.BigEndian.PutUint16(buf[6:8], p.op)
	copy(buf[8:14], p.sha[:])
	copy(buf[14:18], p.spa[:])
	copy(buf[18:24], p.tha[:])
	copy(buf[24:28], p.tpa[:])

	return buf
}
