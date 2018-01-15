package common

import "encoding/hex"

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

func Hex2Bytes(str string) []byte {
	h, _ := hex.DecodeString(str)

	return h
}

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
