package rlp

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"
	"time"
)

func TestCodec(t *testing.T) {
	var a byte

	// 1 byte 0
	a = 0
	data, err := EncodeToBytes(a)
	if err != nil {
		t.Error(err)
	}
	if len(data) != 1 || data[0] != 128+a {
		t.Errorf("EncodeToBytes want 0x%x, but get 0x%x, len %d", data[0], a, len(data))
	}

	// 1 byte 1~127
	a = 127
	data, err = EncodeToBytes(a)
	if err != nil {
		t.Error(err)
	}
	if len(data) != 1 || data[0] != a {
		t.Errorf("EncodeToBytes want 0x%x, but get 0x%x, len %d", data[0], a, len(data))
	}

	var res byte
	err = Decode(bytes.NewReader(data), &res)
	if err != nil {
		t.Error(err)
	}
	if res != 127 {
		t.Errorf("Decode want 0x%x, but get 0x%x", a, res)
	}

	// 1 byte 128~255
	a = 128
	data, err = EncodeToBytes(a)
	if err != nil {
		t.Error(err)
	}
	if len(data) != 2 || data[1] != a || data[0] != 129 {
		t.Errorf("EncodeToBytes want 0x%x, but get 0x%x, len %d", a, data, len(data))
	}
	err = Decode(bytes.NewReader(data), &res)
	if err != nil {
		t.Error(err)
	}
	if res != 128 {
		t.Errorf("Decode want 0x%x, but get 0x%x", a, res)
	}

	a = 255
	data, err = EncodeToBytes(a)
	if err != nil {
		t.Error(err)
	}
	if len(data) != 2 || data[1] != a || data[0] != 129 {
		t.Errorf("EncodeToBytes want 0x%x, but get 0x%x, len %d", a, data, len(data))
	}

	err = Decode(bytes.NewReader(data), &res)
	if err != nil {
		t.Error(err)
	}
	if res != 255 {
		t.Errorf("Decode want 0x%x, but get 0x%x", a, res)
	}

	// byte array lenght < 56
	b := []byte{9: 0}
	data, err = EncodeToBytes(b)
	if err != nil {
		t.Error(err)
	}
	if len(data) != len(b)+1 || data[0] != 128+byte(len(b)) {
		t.Errorf("EncodeToBytes want 0x%x, but get 0x%x, len %d", b, data, len(data))
	}

	var resArray []byte
	err = Decode(bytes.NewReader(data), &resArray)
	if err != nil {
		t.Error(err)
	}
	if len(resArray) != len(b) {
		t.Errorf("Decode length want 0x%x, but get 0x%x", len(b), len(resArray))
	}
	for i, r := range resArray {
		if r != b[i] {
			t.Errorf("Decode want 0x%x, but get 0x%x", b[i], r)
		}
	}

	b = []byte{54: 0}
	data, err = EncodeToBytes(b)
	if err != nil {
		t.Error(err)
	}
	if len(data) != len(b)+1 || data[0] != 128+byte(len(b)) {
		t.Errorf("EncodeToBytes want 0x%x, but get 0x%x, len %d", b, data, len(data))
	}

	err = Decode(bytes.NewReader(data), &resArray)
	if err != nil {
		t.Error(err)
	}
	if len(resArray) != len(b) {
		t.Errorf("Decode length want 0x%x, but get 0x%x", len(b), len(resArray))
	}
	for i, r := range resArray {
		if r != b[i] {
			t.Errorf("Decode want 0x%x, but get 0x%x", b[i], r)
		}
	}

	// byte array lenght >= 56
	b = []byte{55: 0}
	data, err = EncodeToBytes(b)
	if err != nil {
		t.Error(err)
	}
	if len(data) != len(b)+1+1 || data[0] != byte(0xb7+1) || data[1] != byte(len(b)) {
		t.Errorf("EncodeToBytes want 0x%x, but get 0x%x, len %d", b, data, len(data))
	}

	err = Decode(bytes.NewReader(data), &resArray)
	if err != nil {
		t.Error(err)
	}
	if len(resArray) != len(b) {
		t.Errorf("Decode length want 0x%x, but get 0x%x", len(b), len(resArray))
	}
	for i, r := range resArray {
		if r != b[i] {
			t.Errorf("Decode want 0x%x, but get 0x%x", b[i], r)
		}
	}

	b = []byte{99: 0}
	data, err = EncodeToBytes(b)
	if err != nil {
		t.Error(err)
	}
	if len(data) != len(b)+1+1 || data[0] != byte(0xb7+1) || data[1] != byte(len(b)) {
		t.Errorf("EncodeToBytes want 0x%x, but get 0x%x, len %d", b, data, len(data))
	}

	err = Decode(bytes.NewReader(data), &resArray)
	if err != nil {
		t.Error(err)
	}
	if len(resArray) != len(b) {
		t.Errorf("Decode length want 0x%x, but get 0x%x", len(b), len(resArray))
	}
	for i, r := range resArray {
		if r != b[i] {
			t.Errorf("Decode want 0x%x, but get 0x%x", b[i], r)
		}
	}
}

type Msg struct {
	Code       uint64
	Size       uint32 // size of the paylod
	ReceivedAt time.Time
}

func TestTimeCodec(t *testing.T) {
	m := Msg{Code: 1, Size: 2, ReceivedAt: time.Now()}
	o, err := EncodeToBytes(m)
	if err != nil {
		t.Errorf("Encode timer err")
	} else {
		t.Errorf("Encode timer ok 1")
		//o, err := ioutil.ReadAll(r)
		if err == nil {
			t.Errorf("Encode timer ok 2")
			fmt.Printf("%d\n", len(o))

			var m1 Msg
			Decode(bytes.NewReader(o), &m1)
			fmt.Printf("time 2", m1.ReceivedAt)
		}
	}
}

func TestBigCodec(t *testing.T) {
	var b big.Int
	o, err := EncodeToBytes(big.NewInt(100))
	if err != nil {
		t.Errorf("Encode big number err")
	}
	fmt.Printf("0x%x\n", o)

	err = DecodeBytes(o, &b)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf(b.String())
}
