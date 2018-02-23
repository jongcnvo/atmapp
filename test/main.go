package main

import (
	"../rlp"
	"fmt"
	"math/big"
)

func main() {
	var td big.Int
	td.SetInt64(1)
	data, err := rlp.EncodeToBytes(td)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("0x%x", data)
}
