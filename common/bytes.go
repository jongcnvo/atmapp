package common

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
)

const uintBits = 32 << (uint64(^uint(0)) >> 63)

var (
	//ErrEmptyString string is empty
	ErrEmptyString = &decError{"empty hex string"}
	//ErrSyntax string format is not hex
	ErrSyntax = &decError{"invalid hex string"}
	//ErrMissingPrefix string for hex without 0x prefix
	ErrMissingPrefix = &decError{"hex string without 0x prefix"}
	//ErrOddLength string length is odd not even
	ErrOddLength = &decError{"hex string of odd length"}
	//ErrEmptyNumber string only contains 0x prefix
	ErrEmptyNumber = &decError{"hex string \"0x\""}
	//ErrLeadingZero hex number with zero beginning
	ErrLeadingZero = &decError{"hex number with leading zero digits"}
	//ErrUint64Range hex number is larger than 64 bit len
	ErrUint64Range = &decError{"hex number > 64 bits"}
	//ErrUintRange hex number is larger than uint len
	ErrUintRange = &decError{fmt.Sprintf("hex number > %d bits", uintBits)}
	//ErrBig256Range hex number is larger than 256 bit len
	ErrBig256Range = &decError{"hex number > 256 bits"}
)

type decError struct{ msg string }

func (err decError) Error() string { return err.msg }

//CopyBytes returns a duplication of self byte array
//Input - copiedBytes: pointer to destination
//Output - copiedBytes: copy byte array to destination
func CopyBytes(b []byte) (copiedBytes []byte) {
	if b == nil {
		return nil
	}
	copiedBytes = make([]byte, len(b))
	copy(copiedBytes, b)
	return
}

//Hex2Bytes converts a hex string to byte array
func Hex2Bytes(str string) []byte {
	h, _ := hex.DecodeString(str)
	return h
}

//FromHex converts a hex string with 0x prefix to byte array
func FromHex(s string) []byte {
	if len(s) > 1 {
		if s[0:2] == "0x" || s[0:2] == "0X" {
			s = s[2:]
		}
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return Hex2Bytes(s)
}

func has0xPrefix(input string) bool {
	return len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X')
}

// Decode decodes a hex string with 0x prefix.
func Decode(input string) ([]byte, error) {
	if len(input) == 0 {
		return nil, ErrEmptyString
	}
	if !has0xPrefix(input) {
		return nil, ErrMissingPrefix
	}
	b, err := hex.DecodeString(input[2:])
	if err != nil {
		err = mapError(err)
	}
	return b, err
}

func mapError(err error) error {
	if err, ok := err.(*strconv.NumError); ok {
		switch err.Err {
		case strconv.ErrRange:
			return ErrUint64Range
		case strconv.ErrSyntax:
			return ErrSyntax
		}
	}
	if _, ok := err.(hex.InvalidByteError); ok {
		return ErrSyntax
	}
	if err == hex.ErrLength {
		return ErrOddLength
	}
	return err
}

// MustDecode decodes a hex string with 0x prefix. It panics for invalid input.
func MustDecode(input string) []byte {
	dec, err := Decode(input)
	if err != nil {
		panic(err)
	}
	return dec
}

// EncodeBig encodes bigint as a hex string with 0x prefix.
// The sign of the integer is ignored.
func EncodeBig(bigint *big.Int) string {
	nbits := bigint.BitLen()
	if nbits == 0 {
		return "0x0"
	}
	return fmt.Sprintf("%#x", bigint)
}

func RightPadBytes(slice []byte, l int) []byte {
	if l <= len(slice) {
		return slice
	}

	padded := make([]byte, l)
	copy(padded, slice)

	return padded
}

func LeftPadBytes(slice []byte, l int) []byte {
	if l <= len(slice) {
		return slice
	}

	padded := make([]byte, l)
	copy(padded[l-len(slice):], slice)

	return padded
}
