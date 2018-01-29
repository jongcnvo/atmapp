package common

import (
	"encoding/json"
	"testing"
)

// Test hex string decode
func TestDecode(t *testing.T) {
	var err error
	var b []byte
	_, err = Decode("abc")
	if err != ErrMissingPrefix {
		t.Error(err)
	}

	_, err = Decode("")
	if err != ErrEmptyString {
		t.Error(err)
	}

	b, err = Decode("0x")
	if err != nil {
		t.Error(err)
	}

	b, err = Decode("0x12g0")
	if err != ErrSyntax {
		t.Error(err)
	}

	b, err = Decode("0x120")
	if err != ErrOddLength {
		t.Error(err)
	}

	b, err = Decode("0x010203040a")
	res := []byte{1, 2, 3, 4, 10}
	if err != nil {
		t.Error(err)
	} else {
		for i, bitem := range b {
			if bitem != res[i] {
				t.Errorf("Decode(), want 0x%x, but get 0x%x", res[i], bitem)
			}
		}
	}
}

// Test hash operations
func TestHash(t *testing.T) {

	h := HexToHash("0x0a7dc2333a3040eac9f35e6498bf23726f02a8dbbe51b59cea236a2af71b98f5")
	hbyte := FromHex("0a7dc2333a3040eac9f35e6498bf23726f02a8dbbe51b59cea236a2af71b98f5")
	if len(h) != HashLength {
		t.Errorf("HexToHash(), want len %d, but get %d", HashLength, len(h))
	}
	for i, b := range hbyte {
		if b != h.Bytes()[i] {
			t.Errorf("HexToHash(), want 0x%x, but get 0x%x", b, h[i])
		}
	}

	h1 := HexToHash("0a7dc2333a3040eac9f35e6498bf23726f02a8dbbe51b59cea236a2af71b98f5")
	if len(h1) != HashLength {
		t.Errorf("HexToHash(), want len %d, but get %d", HashLength, len(h1))
	}
	for i, b := range hbyte {
		if b != h1.Bytes()[i] {
			t.Errorf("HexToHash(), want 0x%x, but get 0x%x", b, h1[i])
		}
	}

	// trim the last part
	h2 := HexToHash("0a0b7dc2333a3040eac9f35e6498bf23726f02a8dbbe51b59cea236a2af71b98f5")
	if len(h2) != HashLength {
		t.Errorf("HexToHash(), want len %d, but get %d", HashLength, len(h2))
	}
	hbyte[0] = 11
	for i, b := range hbyte {
		if b != h2.Bytes()[i] {
			t.Errorf("HexToHash(), want 0x%x, but get 0x%x", b, h2[i])
		}
	}
	hbyte[0] = 10

	// put to the end
	h3 := HexToHash("7dc2333a3040eac9f35e6498bf23726f02a8dbbe51b59cea236a2af71b98f5")
	if len(h3) != HashLength {
		t.Errorf("HexToHash(), want len %d, but get %d", HashLength, len(h2))
	}
	hbyte[0] = 0
	for i, b := range hbyte {
		if b != h3.Bytes()[i] {
			t.Errorf("HexToHash(), want 0x%x, but get 0x%x", b, h3[i])
		}
	}
	hbyte[0] = 10
}

// Test address operations
func TestAddr(t *testing.T) {
	a := HexToAddress("0xDe1660B39610C17865449B99343D740E4c6D6253")
	abyte := FromHex("De1660B39610C17865449B99343D740E4c6D6253")
	if len(a) != AddressLength {
		t.Errorf("HexToAddress(), want len %d, but get %d", AddressLength, len(a))
	}
	for i, b := range abyte {
		if b != a.Bytes()[i] {
			t.Errorf("HexToAddress(), want 0x%x, but get 0x%x", b, a[i])
		}
	}

	a1 := HexToAddress("0xDe1660B39610C17865449B99343D740E4c6D6253")
	if len(a1) != AddressLength {
		t.Errorf("HexToAddress(), want len %d, but get %d", AddressLength, len(a1))
	}
	for i, b := range abyte {
		if b != a1.Bytes()[i] {
			t.Errorf("HexToAddress(), want 0x%x, but get 0x%x", b, a1[i])
		}
	}

	// trim the last part
	abyte[0] = 11
	a2 := HexToAddress("De0b1660B39610C17865449B99343D740E4c6D6253")
	if len(a2) != AddressLength {
		t.Errorf("HexToAddress(), want len %d, but get %d", AddressLength, len(a2))
	}
	for i, b := range abyte {
		if b != a2.Bytes()[i] {
			t.Errorf("HexToAddress(), want 0x%x, but get 0x%x", b, a2[i])
		}
	}
	abyte[0] = 222

	// put to the end
	abyte[0] = 0
	a3 := HexToAddress("1660B39610C17865449B99343D740E4c6D6253")
	if len(a3) != AddressLength {
		t.Errorf("HexToAddress(), want len %d, but get %d", AddressLength, len(a3))
	}
	for i, b := range abyte {
		if b != a3.Bytes()[i] {
			t.Errorf("HexToAddress(), want 0x%x, but get 0x%x", b, a3[i])
		}
	}
	abyte[0] = 222

	// marshal address to string(small letter)
	str, err := json.Marshal(a)
	if err != nil {
		t.Error(err)
	}
	strIn := "\"0xde1660b39610c17865449b99343d740e4c6d6253\""
	for i, s := range strIn {
		if byte(s) != str[i] {
			t.Errorf("MarshalText(), want 0x%x, but get 0x%x", byte(s), str[i])
		}
	}

	// unmarshall string to address
	var a4 Address
	err = json.Unmarshal([]byte(strIn), &a4)
	if err != nil {
		t.Error(err)
	}
	if len(a4) != AddressLength {
		t.Errorf("UnmarshalText(), want len %d, but get %d", AddressLength, len(a4))
	}
	for i, b := range abyte {
		if b != a4.Bytes()[i] {
			t.Errorf("UnmarshalText(), want 0x%x, but get 0x%x", b, a4[i])
		}
	}
}
