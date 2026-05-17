package main

import (
	"fmt"
	"unsafe"
)

type TenInts struct {
	a, b, c, d, e, f, g int
	h, i, j int16
	//ends up being 64 bytes because of padding
}

func main() {
	var s TenInts
	fmt.Printf("Size of struct: %d bytes\n", unsafe.Sizeof(s))
}
