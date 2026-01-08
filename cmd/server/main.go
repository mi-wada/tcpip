package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"syscall"
	"unsafe"

	"github.com/mi-wada/tcpip/arp"
	"github.com/mi-wada/tcpip/ethernet"
	"github.com/mi-wada/tcpip/ip"
	"github.com/mi-wada/tcpip/util"
)

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

// ping 192.168.10.2
func main() {
	// 1. TUN/TAPクローンデバイスをオープン
	fd, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		log.Fatalf("デバイスオープン失敗: %v", err)
	}
	defer fd.Close()

	// 2. TAPデバイス(tap0)として登録するための設定
	var ifr ifreq
	copy(ifr.name[:], "tap0")
	ifr.flags = IFF_TAP | IFF_NO_PI

	// 3. ioctlでデバイスを作成/紐付け
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd.Fd(), uintptr(TUNSETIFF), uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		log.Fatalf("ioctl失敗: %v", errno)
	}

	fmt.Println("tap0 デバイスをリスン中... (Ctrl+Cで終了)")

	// 4. パケット読み込みループ
	frame := make([]byte, 1514) // イーサネット最大1500 + ヘッダ14
	for {
		n, err := fd.Read(frame)
		if err != nil {
			log.Printf("Readエラー: %v", err)
			continue
		}

		header, payload, err := ethernet.Unmarshal(frame[:n])
		if err != nil {
			panic(err)
		}

		switch header.Type {
		case ethernet.TypeARP:
			log.Print("ARP frame")
			util.Dump(payload)
			reply, err := arp.Handle(payload)
			if err != nil {
				panic(err)
			}
			header := ethernet.Header{
				Dst:  header.Src,
				Src:  [6]byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x02},
				Type: ethernet.TypeARP,
			}
			replyFrame := ethernet.Marshal(header, reply)
			fd.Write(replyFrame)
		case ethernet.TypeIPv4:
			log.Print("IPv4 frame")
			replyIPPacket, err := ip.Handle(payload)
			if errors.Is(err, ip.ErrEmpty) {
				continue
			}
			if err != nil {
				panic(err)
			}
			ethHeader := ethernet.Header{
				Dst:  header.Src,
				Src:  [6]byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x02},
				Type: ethernet.TypeIPv4,
			}
			replyFrame := ethernet.Marshal(ethHeader, replyIPPacket)
			fd.Write(replyFrame)
		case ethernet.TypeIPv6:
			// TODO: Handle IPv6
			log.Print("IPv6 frame")
			util.Dump(payload)
		}
	}
}
