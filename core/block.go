package core

import (
	"../common"
	"../crypto"
	"../rlp"
	"math/big"
)

// Header represents a block header in the Ethereum blockchain.
type Header struct {
	ParentHash  common.Hash       `json:"parentHash"       gencodec:"required"`
	UncleHash   common.Hash       `json:"sha3Uncles"       gencodec:"required"`
	Coinbase    common.Address    `json:"miner"            gencodec:"required"`
	Root        common.Hash       `json:"stateRoot"        gencodec:"required"`
	TxHash      common.Hash       `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash common.Hash       `json:"receiptsRoot"     gencodec:"required"`
	Bloom       common.Bloom      `json:"logsBloom"        gencodec:"required"`
	Difficulty  *big.Int          `json:"difficulty"       gencodec:"required"`
	Number      *big.Int          `json:"number"           gencodec:"required"`
	GasLimit    uint64            `json:"gasLimit"         gencodec:"required"`
	GasUsed     uint64            `json:"gasUsed"          gencodec:"required"`
	Time        *big.Int          `json:"timestamp"        gencodec:"required"`
	Extra       []byte            `json:"extraData"        gencodec:"required"`
	MixDigest   common.Hash       `json:"mixHash"          gencodec:"required"`
	Nonce       common.BlockNonce `json:"nonce"            gencodec:"required"`
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
