package main

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"unsafe"

	"github.com/mi-wada/tcpip/ethernet"
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
			// TODO: Handle ARP
			log.Printf("ARP frame")
			util.Dump(payload)
		case ethernet.TypeIPv4:
			// TODO: Handle IPv4
			log.Printf("IPv4 frame")
			util.Dump(payload)
		case ethernet.TypeIPv6:
			// TODO: Handle IPv6
			log.Printf("IPv6 frame")
			util.Dump(payload)
		}
	}
}
