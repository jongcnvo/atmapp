package crypto

import (
	"fmt"
	"testing"
)

func TestSign(t *testing.T) {
	hw := NewKeccak256()
	fmt.Println("hw:", hw)
}
