package math

import (
	"math/big"
	"testing"
)

// Test big number operations
func TestBigOper(t *testing.T) {

	// test word length in math.big
	bw := big.Word(0)
	wordLen := 32 << (^bw >> 32 & 1)
	if wordLen != wordBits {
		t.Errorf("wordBits, want %d, but get %d", wordLen, wordBits)
	}

	// test big int (8 bytes) to byte array
	var bi big.Int
	var out [8]byte
	res := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	bi.SetInt64(0x0102030405060708)
	ReadBits(&bi, out[:])
	for i, r := range res {
		if r != out[i] {
			t.Errorf("ReadBits(), want 0x%x, but get 0x%x", r, out[i])
		}
	}

	// test big int to len 4 byte array, first 4 bytes trimmed
	bi.SetInt64(0x0102030405060708)
	var sout [4]byte
	sres := [4]byte{0x05, 0x06, 0x07, 0x08}
	ReadBits(&bi, sout[:])
	for i, r := range sres {
		if r != sout[i] {
			t.Errorf("ReadBits(), want 0x%x, but get 0x%x", r, sout[i])
		}
	}

	// test big int to len 10 byte array, first 4 bytes trimmed
	bi.SetInt64(0x0102030405060708)
	var lout [10]byte
	lres := [10]byte{0x0, 0x0, 0x01, 0x02, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8}
	ReadBits(&bi, lout[:])
	for i, r := range lres {
		if r != lout[i] {
			t.Errorf("ReadBits(), want 0x%x, but get 0x%x", r, lout[i])
		}
	}

	// test big int to len n array, padding len 0 < 8
	var p []byte
	p = PaddedBigBytes(&bi, 0)
	for i, pitem := range p {
		if pitem != res[i] {
			t.Errorf("PaddedBigBytes(), want 0x%x, but get 0x%x", res[i], pitem)
		}
	}

	// test big int to len n array, padding len 4 < 8
	p = PaddedBigBytes(&bi, 4)
	for i, pitem := range p {
		if pitem != res[i] {
			t.Errorf("PaddedBigBytes(), want 0x%x, but get 0x%x", res[i], pitem)
		}
	}

	// test big int to len n array, padding len 10 > 8
	p = PaddedBigBytes(&bi, 10)
	for i, pitem := range p {
		if pitem != lres[i] {
			t.Errorf("PaddedBigBytes(), want 0x%x, but get 0x%x", lres[i], pitem)
		}
	}
}
