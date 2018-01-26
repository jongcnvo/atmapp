package common

import (
	"testing"
)

// Tests big number operations
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
