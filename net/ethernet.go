package net

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"unsafe"

	"github.com/mi-wada/tcpip/arp"
	"github.com/mi-wada/tcpip/ethernet"
)

type EthernetConn struct {
	fd           *os.File
	dst          *[6]byte
	dstIPAddress [4]byte
}

var _ Conn = &EthernetConn{}

var defaultEthernetSrc = [6]byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x02}

func NewEthernetConn(ipAddress [4]byte) (*EthernetConn, error) {
	// Linuxの再定義定数
	const (
		TUNSETIFF = 0x400454ca // /usr/include/linux/if_tun.h より
		IFF_TAP   = 0x0002
		IFF_NO_PI = 0x1000 // パケット情報ヘッダを付与しない設定
	)

	type ifreq struct {
		name  [16]byte
		flags uint16
	}

	// 1. TUN/TAPクローンデバイスをオープン
	fd, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		log.Fatalf("デバイスオープン失敗: %v", err)
	}

	// 2. TAPデバイス(tap0)として登録するための設定
	var ifr ifreq
	copy(ifr.name[:], "tap0")
	ifr.flags = IFF_TAP | IFF_NO_PI

	// 3. ioctlでデバイスを作成/紐付け
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd.Fd(), uintptr(TUNSETIFF), uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		log.Fatalf("ioctl失敗: %v", errno)
	}

	return &EthernetConn{fd: fd, dstIPAddress: ipAddress}, nil
}

func (c *EthernetConn) Read(p []byte) (n int, err error) {
	frame := make([]byte, 1514) // イーサネット最大1500 + ヘッダ14
	var payload []byte
	for {
		if n, err = c.fd.Read(frame); err != nil {
			return 0, err
		}
		var header ethernet.Header
		header, payload, err = ethernet.Unmarshal(frame[:n])
		if err != nil {
			return 0, err
		}
		if header.Dst != defaultEthernetSrc {
			continue
		}
		break
	}
	return copy(p, payload), nil
}

var broadcastMacAddress = [6]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

func (c *EthernetConn) Write(p []byte) (n int, err error) {
	if c.dst == nil {
		if err := c.resolveARP(); err != nil {
			return 0, err
		}
	}

	header := ethernet.Header{
		Dst:  *c.dst,
		Src:  defaultEthernetSrc,
		Type: ethernet.TypeIPv4, // TODO: pass through via New...
	}
	frame := ethernet.Marshal(header, p)
	if _, err = c.fd.Write(frame); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *EthernetConn) resolveARP() error {
	header := ethernet.Header{
		Dst:  broadcastMacAddress,
		Src:  defaultEthernetSrc,
		Type: ethernet.TypeARP,
	}
	payload := arp.ARPRequestPayload(c.dstIPAddress)
	frame := ethernet.Marshal(header, payload)
	if _, err := c.fd.Write(frame); err != nil {
		return fmt.Errorf("failed to resolve ARP: %w", err)
	}
	replyFrame := make([]byte, 1514)
	n, err := c.Read(replyFrame)
	if err != nil {
		return fmt.Errorf("failed to read ARP reply: %w", err)
	}
	arpPacket, err := arp.Unmarshal(replyFrame[:n])
	if err != nil {
		return fmt.Errorf("failed to unmarshal ARP reply: %w", err)
	}
	c.dst = &arpPacket.Sha
	fmt.Printf("c.dst: %v\n", c.dst)
	return nil
}

func (c *EthernetConn) RRR() error {
	return c.resolveARP()
}

func (c *EthernetConn) Close() error {
	return c.fd.Close()
}
