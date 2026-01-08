package ip

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"sync/atomic"

	"github.com/mi-wada/tcpip/icmp"
	"github.com/mi-wada/tcpip/tcp"
	"github.com/mi-wada/tcpip/udp"
	"github.com/mi-wada/tcpip/util"
)

var (
	ErrEmpty = errors.New("empty")
)

func Handle(payload []byte) ([]byte, error) {
	h, payload, err := Unmarshal(payload)
	if err != nil {
		return nil, err
	}
	log.Printf("Received IP Header: %#v", h)

	var replyIPPacket []byte
	switch h.Protocol {
	case ProtocolICMP:
		reply, err := icmp.Handle(payload)
		if err != nil {
			panic(err)
		}
		fmt.Printf("ICMP reply: %#v\n", reply)
		ipHeader := Header{
			Version:        4,
			IHL:            5,
			TOS:            0,
			TotalLen:       uint16(20 + len(reply)),
			ID:             NewID(),
			Flags:          0b010,
			FragmentOffset: 0,
			TTL:            40,
			Protocol:       ProtocolICMP,
			Src:            h.Dst,
			Dst:            h.Src,
		}
		replyIPPacket = Marshal(ipHeader, reply)
	case ProtocolUDP:
		reply, err := udp.Handle(payload)
		if err != nil {
			panic(err)
		}
		ipHeader := Header{
			Version:        4,
			IHL:            5,
			TOS:            0,
			TotalLen:       uint16(20 + len(reply)),
			ID:             NewID(),
			Flags:          0b010,
			FragmentOffset: 0,
			TTL:            64,
			Protocol:       ProtocolUDP,
			Src:            h.Dst,
			Dst:            h.Src,
		}
		replyIPPacket = Marshal(ipHeader, reply)
	case ProtocolTCP:
		reply, err := tcp.Handle(payload, h.Src, h.Dst)
		if errors.Is(err, tcp.ErrEmpty) {
			return nil, ErrEmpty
		}
		if err != nil {
			panic(err)
		}
		ipHeader := Header{
			Version:        4,
			IHL:            5,
			TOS:            0,
			TotalLen:       uint16(20 + len(reply)),
			ID:             NewID(),
			Flags:          0b010,
			FragmentOffset: 0,
			TTL:            64,
			Protocol:       ProtocolTCP,
			Src:            h.Dst,
			Dst:            h.Src,
		}
		replyIPPacket = Marshal(ipHeader, reply)
	default:
		log.Printf("unsupported ip protocol: %d", h.Protocol)
		util.Dump(payload)
	}

	return replyIPPacket, nil
}

const (
	// https://datatracker.ietf.org/doc/html/rfc792
	ProtocolICMP = 1
	ProtocolTCP  = 6
	// https://datatracker.ietf.org/doc/html/rfc768
	ProtocolUDP = 17
)

//	0                   1                   2                   3
//	0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |Version|  IHL  |Type of Service|          Total Length         |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |         Identification        |Flags|      Fragment Offset    |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |  Time to Live |    Protocol   |         Header Checksum       |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                       Source Address                          |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                    Destination Address                        |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                    Options                    |    Padding    |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
type Header struct {
	Version        uint8
	IHL            uint8 // IHLは4bytesワード、つまりIHLが1とは4byteの意味。
	TOS            uint8
	TotalLen       uint16
	ID             uint16
	Flags          uint8
	FragmentOffset uint16
	TTL            uint8
	Protocol       uint8
	Checksum       uint16
	Src            [4]byte
	Dst            [4]byte
	Options        uint32
}

func (h Header) Marshal() []byte {
	buf := make([]byte, h.IHL*4)

	buf[0] = h.Version<<4 | (h.IHL & 0x0f)
	buf[1] = h.TOS
	binary.BigEndian.PutUint16(buf[2:4], h.TotalLen)

	binary.BigEndian.PutUint16(buf[4:6], h.ID)
	flagsAndOff := (uint16(h.Flags) << 13) | (h.FragmentOffset & 0x1fff)
	binary.BigEndian.PutUint16(buf[6:8], flagsAndOff)

	buf[8] = h.TTL
	buf[9] = h.Protocol
	// Set 仮の値 to Checksum
	binary.BigEndian.PutUint16(buf[10:12], 0)

	copy(buf[12:16], h.Src[:])
	copy(buf[16:20], h.Dst[:])
	// TODO: Options

	binary.BigEndian.PutUint16(buf[10:12], checksum(buf))
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

func Unmarshal(packet []byte) (Header, []byte, error) {
	if len(packet) < 20 {
		return Header{}, nil, errors.New("packet too short, ip header missing")
	}
	header := Header{}
	header.Version = packet[0] >> 4
	header.IHL = packet[0] & 0x0f
	if header.IHL == 6 {
		panic("need to implement to parse options")
	}
	header.TOS = packet[1]
	header.TotalLen = binary.BigEndian.Uint16(packet[2:4])
	header.ID = binary.BigEndian.Uint16(packet[4:6])
	header.Flags = packet[6] >> 5
	header.FragmentOffset = binary.BigEndian.Uint16([]byte{packet[6] & 0b00011111, packet[7]})
	header.TTL = packet[8]
	header.Protocol = packet[9]
	header.Checksum = binary.BigEndian.Uint16(packet[10:12])
	copy(header.Src[:], packet[12:16])
	copy(header.Dst[:], packet[16:20])

	// IHLは4バイトワード単位なので
	headerLen := int(header.IHL) * 4
	if len(packet) < headerLen {
		return Header{}, nil, errors.New("packet shorter than IHL")
	}

	return header, packet[headerLen:], nil
}

func Marshal(header Header, payload []byte) []byte {
	buf := make([]byte, header.TotalLen)
	copy(buf[:header.IHL*4], header.Marshal())
	copy(buf[header.IHL*4:], payload)
	return buf
}

var id atomic.Uint64

func NewID() uint16 {
	ret := uint16(id.Load() % math.MaxUint16)
	id.Add(1)
	return ret
}
