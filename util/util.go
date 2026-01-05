package util

import (
	"fmt"
	"unicode"
)

func Dump(data []byte) {
	fmt.Printf("--- Packet Dump (%d bytes) ---\n", len(data))

	for i := 0; i < len(data); i += 16 {
		fmt.Printf("%04x: ", i)

		for j := 0; j < 16; j++ {
			if i+j < len(data) {
				fmt.Printf("%02x ", data[i+j])
			} else {
				fmt.Printf("   ") // パケット末尾の空白埋め
			}
			if j == 7 {
				fmt.Printf(" ") // 8バイト目の区切り
			}
		}

		fmt.Printf("  ")

		for j := 0; j < 16; j++ {
			if i+j < len(data) {
				b := data[i+j]
				if unicode.IsPrint(rune(b)) && b < 128 {
					fmt.Printf("%c", b)
				} else {
					fmt.Printf(".")
				}
			}
		}
		fmt.Println()
	}
	fmt.Println("------------------------------")
}
