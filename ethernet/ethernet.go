package ethernet

import (
	"encoding/binary"
	"fmt"
)

type Header struct {
	Dst  [6]byte
	Src  [6]byte
	Type uint16
}

const (
	TypeARP  = 0x0806
	TypeIPv4 = 0x0800
	TypeIPv6 = 0x86dd
)

func Unmarshal(frame []byte) (Header, []byte, error) {
	if len(frame) < 14 {
		return Header{}, nil, fmt.Errorf("frame too short")
	}

	var h Header
	copy(h.Dst[:], frame[0:6])
	copy(h.Src[:], frame[6:12])
	h.Type = binary.BigEndian.Uint16(frame[12:14])

	return h, frame[14:], nil
}
