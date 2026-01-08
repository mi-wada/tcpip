package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/mi-wada/tcpip/icmp"
)

var data = []byte("0123456789")

func main() {
	c := flag.Int("c", 0, "ping count. default is endless.")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "host is required")
		os.Exit(1)
	}
	host := args[0]

	log.Printf("c: %v, host: %v\n", *c, host)

	// TODO: mynet.Dialを実装して、Read, Write, CloseするConnを返す。内部で一番下ではIPアドレスを使ってARPでmacアドレスを逆引きする。IPでのsrcは192.168.10.1になる。server/main.goでしているみたいにtun/tapをopenする感じかしら。
	conn, err := net.Dial("ip4:1", host)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	for i := range *c {
		echo := icmp.Packet{
			Type: icmp.TypeEcho,
			Code: 0,
			ID:   10,
			Seq:  uint16(i),
			Data: []byte("0123456789"),
		}
		_, err := conn.Write(echo.Marshal())
		if err != nil {
			panic(err)
		}
		buf := make([]byte, 20+8+len(data))
		if _, err = conn.Read(buf); err != nil {
			panic(err)
		}
		buf = buf[20:]
		echoReply, err := icmp.Unmarshal(buf)
		if err != nil {
			panic(err)
		}
		fmt.Printf("echoReply: %#v\n", echoReply)
		fmt.Printf("%d bytes from %s: icmp_seq=%d ttl=%d time=%f ms\n", len(buf), host, echoReply.Seq, 10, 10.0)
		time.Sleep(1 * time.Second)
	}
}
