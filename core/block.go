package core

import (
	"../common"
	"../crypto"
)

// Header represents a block header in the Ethereum blockchain.
type Header struct {
	ParentHash common.Hash `json:"parentHash"       gencodec:"required"`
}

// Hash returns the block hash of the header, which is simply the keccak256 hash of its
// RLP encoding.
func (h *Header) Hash() common.Hash {
	return rlpHash(h)
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := crypto.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}
