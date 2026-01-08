package tcp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
)

type table struct {
	mu          sync.RWMutex
	connections map[string]*state
}

var globalTable = table{
	connections: make(map[string]*state),
}

func toTableKey(srcAddress [4]byte, srcPort uint16, dstAddress [4]byte, dstPort uint16) string {
	ipToStr := func(a [4]byte) string {
		return fmt.Sprintf("%d.%d.%d.%d", a[0], a[1], a[2], a[3])
	}
	return fmt.Sprintf("%s:%d->%s:%d", ipToStr(srcAddress), srcPort, ipToStr(dstAddress), dstPort)
}

type state struct {
	t   stateType
	irs uint32 // initial rcv seq number
	iss uint32 // initial snd seq number
	nrs uint32 // next rcv seq number
	nss uint32 // next snd seq number
}

type stateType string

const (
	tSYN     stateType = "SYN"
	tSYNRCVD stateType = "SYNRCVD"
	tESTAB   stateType = "ESTAB"
)

var (
	ErrEmpty = errors.New("empty")
)

func Handle(payload []byte, src, dst [4]byte) ([]byte, error) {
	p, err := unmarshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal TCP packet: %w", err)
	}
	fmt.Printf("TCP Packet: %#v\n", p)
	tableKey := toTableKey(src, p.srcPort, dst, p.dstPort)
	globalTable.mu.RLock()
	s := globalTable.connections[tableKey]
	globalTable.mu.RUnlock()
	if s == nil {
		globalTable.mu.Lock()
		s = &state{t: tSYN}
		globalTable.connections[tableKey] = s
		globalTable.mu.Unlock()
	}
	switch s.t {
	case tSYN:
		iss := rand.Uint32()
		reply := packet{
			srcPort:   p.dstPort,
			dstPort:   p.srcPort,
			seqNumber: iss,
			ackNumber: p.seqNumber + 1,
			dOffset:   5,
			ack:       true,
			syn:       true,
			window:    0xffff,
		}
		globalTable.mu.Lock()
		globalTable.connections[tableKey].irs = p.seqNumber
		globalTable.connections[tableKey].iss = iss
		globalTable.connections[tableKey].nrs = globalTable.connections[tableKey].irs + 1
		globalTable.connections[tableKey].nss = globalTable.connections[tableKey].iss + 1
		globalTable.connections[tableKey].t = tSYNRCVD
		globalTable.mu.Unlock()
		return reply.marshal(src, dst), nil
	case tSYNRCVD:
		// globalTable.mu.RLock()
		// if p.ack && p.ackNumber == globalTable.connections[tableKey].nss {
		// 	// ok
		// } else {
		// 	fmt.Printf("p: %#v\n", p)
		// 	panic("invalid ACK")
		// }
		// globalTable.mu.RUnlock()

		globalTable.mu.Lock()
		globalTable.connections[tableKey].t = tESTAB
		globalTable.mu.Unlock()

		return nil, ErrEmpty
	case tESTAB:
		globalTable.mu.Lock()
		globalTable.connections[tableKey].nrs += uint32(len(p.data))
		globalTable.mu.Unlock()

		reply := packet{
			srcPort:   p.dstPort,
			dstPort:   p.srcPort,
			seqNumber: globalTable.connections[tableKey].nss,
			ackNumber: globalTable.connections[tableKey].nrs,
			dOffset:   5,
			window:    0xffff,
			ack:       true,
			psh:       true,
			data:      p.data,
		}

		globalTable.mu.Lock()
		globalTable.connections[tableKey].nss += uint32(len(p.data))
		globalTable.mu.Unlock()

		return reply.marshal(src, dst), nil
	default:
		return nil, fmt.Errorf("unknown state type: %s", s.t)
	}
}

//	0                   1                   2                   3
//	0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |          Source Port          |       Destination Port        |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                        Sequence Number                        |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                    Acknowledgment Number                      |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |  Data |       |C|E|U|A|P|R|S|F|                               |
// | Offset| Rsrvd |W|C|R|C|S|S|Y|I|            Window             |
// |       |       |R|E|G|K|H|T|N|N|                               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |           Checksum            |         Urgent Pointer        |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                           [Options]                           |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                                                               :
// :                             Data                              :
// :                                                               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
type packet struct {
	srcPort       uint16
	dstPort       uint16
	seqNumber     uint32
	ackNumber     uint32
	dOffset       uint8
	rsrvd         uint8
	cwr           bool
	ece           bool
	urg           bool
	ack           bool
	psh           bool
	rst           bool
	syn           bool
	fin           bool
	window        uint16
	checksum      uint16
	urgentPointer uint16
	options       []byte
	data          []byte
}

func (p packet) marshal(src, dst [4]byte) []byte {
	buf := make([]byte, 20+len(p.options)+len(p.data))
	binary.BigEndian.PutUint16(buf[0:2], p.srcPort)
	binary.BigEndian.PutUint16(buf[2:4], p.dstPort)
	binary.BigEndian.PutUint32(buf[4:8], p.seqNumber)
	binary.BigEndian.PutUint32(buf[8:12], p.ackNumber)
	buf[12] = p.dOffset<<4 | p.rsrvd
	if p.cwr {
		buf[13] = buf[13] | 1<<7
	}
	if p.ece {
		buf[13] = buf[13] | 1<<6
	}
	if p.urg {
		buf[13] = buf[13] | 1<<5
	}
	if p.ack {
		buf[13] = buf[13] | 1<<4
	}
	if p.psh {
		buf[13] = buf[13] | 1<<3
	}
	if p.rst {
		buf[13] = buf[13] | 1<<2
	}
	if p.syn {
		buf[13] = buf[13] | 1<<1
	}
	if p.fin {
		buf[13] = buf[13] | 1<<0
	}
	binary.BigEndian.PutUint16(buf[14:16], p.window)
	binary.BigEndian.PutUint16(buf[16:18], 0)
	binary.BigEndian.PutUint16(buf[18:20], p.urgentPointer)
	dOffsetBytes := p.dOffset * 4
	copy(buf[20:dOffsetBytes], p.options)
	copy(buf[dOffsetBytes:], p.data)

	p.checksum = checksum(src, dst, buf)
	binary.BigEndian.PutUint16(buf[16:18], p.checksum)

	return buf
}

func checksum(src, dst [4]byte, tcpPayload []byte) uint16 {
	var sum uint32

	// 1. 擬似ヘッダの計算 (IPv4)
	// Source IP
	sum += uint32(binary.BigEndian.Uint16(src[0:2]))
	sum += uint32(binary.BigEndian.Uint16(src[2:4]))
	// Destination IP
	sum += uint32(binary.BigEndian.Uint16(dst[0:2]))
	sum += uint32(binary.BigEndian.Uint16(dst[2:4]))
	// Protocol (TCP = 6) & TCP Length (Header + Data)
	sum += uint32(6)
	sum += uint32(len(tcpPayload))

	// 2. TCPヘッダ + データの計算
	// ※計算時、チェックサムフィールド自体は 0 として扱う必要があります
	for i := 0; i < len(tcpPayload)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(tcpPayload[i : i+2]))
	}

	// 奇数バイトの場合の処理
	if len(tcpPayload)%2 == 1 {
		sum += uint32(tcpPayload[len(tcpPayload)-1]) << 8
	}

	// 3. 溢れた桁（キャリー）を足し戻す (16ビットに収める)
	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	// 4. 最後にビット反転
	return uint16(^sum)
}

func unmarshal(payload []byte) (packet, error) {
	if len(payload) < 20 {
		return packet{}, fmt.Errorf("TCP packet too short: %d", len(payload))
	}

	dOffset := (payload[12] >> 4) & 0x0f
	dOffsetBytes := dOffset * 4
	return packet{
		srcPort:       binary.BigEndian.Uint16(payload[0:2]),
		dstPort:       binary.BigEndian.Uint16(payload[2:4]),
		seqNumber:     binary.BigEndian.Uint32(payload[4:8]),
		ackNumber:     binary.BigEndian.Uint32(payload[8:12]),
		dOffset:       dOffset,
		rsrvd:         payload[12] & 0x0f,
		cwr:           payload[13]>>7&1 == 1,
		ece:           payload[13]>>6&1 == 1,
		urg:           payload[13]>>5&1 == 1,
		ack:           payload[13]>>4&1 == 1,
		psh:           payload[13]>>3&1 == 1,
		rst:           payload[13]>>2&1 == 1,
		syn:           payload[13]>>1&1 == 1,
		fin:           payload[13]>>0&1 == 1,
		window:        binary.BigEndian.Uint16(payload[14:16]),
		checksum:      binary.BigEndian.Uint16(payload[16:18]),
		urgentPointer: binary.BigEndian.Uint16(payload[18:20]),
		options:       payload[20:dOffsetBytes],
		data:          payload[dOffsetBytes:],
	}, nil
}
