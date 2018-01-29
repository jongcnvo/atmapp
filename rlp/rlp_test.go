package rlp

import (
	"testing"
)

func TestEncode(t *testing.T) {
	var a byte

	// 1 byte 1~127
	a = 127
	data, err := EncodeToBytes(a)
	if err != nil {
		t.Error(err)
	}
	if len(data) != 1 || data[0] != a {
		t.Errorf("EncodeToBytes want 0x%x, but get 0x%x, len %d", data[0], a, len(data))
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

	a = 255
	data, err = EncodeToBytes(a)
	if err != nil {
		t.Error(err)
	}
	if len(data) != 2 || data[1] != a || data[0] != 129 {
		t.Errorf("EncodeToBytes want 0x%x, but get 0x%x, len %d", a, data, len(data))
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

	b = []byte{54: 0}
	data, err = EncodeToBytes(b)
	if err != nil {
		t.Error(err)
	}
	if len(data) != len(b)+1 || data[0] != 128+byte(len(b)) {
		t.Errorf("EncodeToBytes want 0x%x, but get 0x%x, len %d", b, data, len(data))
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

	b = []byte{99: 0}
	data, err = EncodeToBytes(b)
	if err != nil {
		t.Error(err)
	}
	if len(data) != len(b)+1+1 || data[0] != byte(0xb7+1) || data[1] != byte(len(b)) {
		t.Errorf("EncodeToBytes want 0x%x, but get 0x%x, len %d", b, data, len(data))
	}
}
