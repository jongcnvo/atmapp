package core

import (
	"../common"
	"../db"
	"../log"
	"../params"
	"../rlp"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math/big"
)

var (
	headHeaderKey = []byte("LastHeader")
	headBlockKey  = []byte("LastBlock")

	// Data item prefixes (use single byte to avoid mixing data types, avoid `i`).
	headerPrefix        = []byte("h")                // headerPrefix + num (uint64 big endian) + hash -> header
	tdSuffix            = []byte("t")                // headerPrefix + num (uint64 big endian) + hash +
	blockHashPrefix     = []byte("H")                // blockHashPrefix + hash -> num (uint64 big endian)
	bodyPrefix          = []byte("b")                // bodyPrefix + num (uint64 big endian) + hash -> block body
	blockReceiptsPrefix = []byte("r")                // blockReceiptsPrefix + num (uint64 big endian) + hash ->
	numSuffix           = []byte("n")                // headerPrefix + num (uint64 big endian) + numSuffix -> hash
	configPrefix        = []byte("atmchain-config-") // config prefix for the db
)

// DatabaseReader wraps the Get method of a backing data store.
type DatabaseReader interface {
	Get(key []byte) (value []byte, err error)
}

// encodeBlockNumber encodes a block number as big endian uint64
func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

// WriteBody serializes the body of a block into the database.
func WriteBody(db db.Putter, hash common.Hash, number uint64, body *Body) error {
	data, err := rlp.EncodeToBytes(body)
	if err != nil {
		return err
	}
	return WriteBodyRLP(db, hash, number, data)
}

// WriteTd serializes the total difficulty of a block into the database.
func WriteTd(db db.Putter, hash common.Hash, number uint64, td *big.Int) error {
	data, err := rlp.EncodeToBytes(td)
	if err != nil {
		return err
	}
	key := append(append(append(headerPrefix, encodeBlockNumber(number)...), hash.Bytes()...), tdSuffix...)
	if err := db.Put(key, data); err != nil {
		log.Crit("Failed to store block total difficulty", "err", err)
	}
	return nil
}

// WriteBlock serializes a block into the database, header and body separately.
func WriteBlock(db db.Putter, block *Block) error {
	// Store the body first to retain database consistency
	if err := WriteBody(db, block.Hash(), block.NumberU64(), block.Body()); err != nil {
		return err
	}
	// Store the header too, signaling full block ownership
	if err := WriteHeader(db, block.Header()); err != nil {
		return err
	}
	return nil
}

// WriteBlockReceipts stores all the transaction receipts belonging to a block
// as a single receipt slice. This is used during chain reorganisations for
// rescheduling dropped transactions.
func WriteBlockReceipts(db db.Putter, hash common.Hash, number uint64, receipts Receipts) error {
	// Convert the receipts into their storage form and serialize them
	storageReceipts := make([]*ReceiptForStorage, len(receipts))
	for i, receipt := range receipts {
		storageReceipts[i] = (*ReceiptForStorage)(receipt)
	}
	bytes, err := rlp.EncodeToBytes(storageReceipts)
	if err != nil {
		return err
	}
	// Store the flattened receipt slice
	key := append(append(blockReceiptsPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
	if err := db.Put(key, bytes); err != nil {
		log.Crit("Failed to store block receipts", "err", err)
	}
	return nil
}

// WriteHeader serializes a block header into the database.
func WriteHeader(db db.Putter, header *Header) error {
	data, err := rlp.EncodeToBytes(header)
	if err != nil {
		return err
	}
	hash := header.Hash().Bytes()
	num := header.Number.Uint64()
	encNum := encodeBlockNumber(num)
	key := append(blockHashPrefix, hash...)
	if err := db.Put(key, encNum); err != nil {
		log.Crit("Failed to store hash to number mapping", "err", err)
	}
	key = append(append(headerPrefix, encNum...), hash...)
	if err := db.Put(key, data); err != nil {
		log.Crit("Failed to store header", "err", err)
	}
	return nil
}

// WriteBodyRLP writes a serialized body of a block into the database.
func WriteBodyRLP(db db.Putter, hash common.Hash, number uint64, rlp rlp.RawValue) error {
	key := append(append(bodyPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
	if err := db.Put(key, rlp); err != nil {
		log.Crit("Failed to store block body", "err", err)
	}
	return nil
}

// WriteCanonicalHash stores the canonical hash for the given block number.
func WriteCanonicalHash(db db.Putter, hash common.Hash, number uint64) error {
	key := append(append(headerPrefix, encodeBlockNumber(number)...), numSuffix...)
	if err := db.Put(key, hash.Bytes()); err != nil {
		log.Crit("Failed to store number to hash mapping", "err", err)
	}
	return nil
}

// WriteHeadHeaderHash stores the head header's hash.
func WriteHeadHeaderHash(db db.Putter, hash common.Hash) error {
	if err := db.Put(headHeaderKey, hash.Bytes()); err != nil {
		log.Crit("Failed to store last header's hash", "err", err)
	}
	return nil
}

// WriteHeadBlockHash stores the head block's hash.
func WriteHeadBlockHash(db db.Putter, hash common.Hash) error {
	if err := db.Put(headBlockKey, hash.Bytes()); err != nil {
		log.Crit("Failed to store last block's hash", "err", err)
	}
	return nil
}

// WriteChainConfig writes the chain config settings to the database.
func WriteChainConfig(db db.Putter, hash common.Hash, cfg *params.ChainConfig) error {
	// short circuit and ignore if nil config. GetChainConfig
	// will return a default.
	if cfg == nil {
		return nil
	}

	jsonChainConfig, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	return db.Put(append(configPrefix, hash[:]...), jsonChainConfig)
}

// GetCanonicalHash retrieves a hash assigned to a canonical block number.
func GetCanonicalHash(db DatabaseReader, number uint64) common.Hash {
	data, _ := db.Get(append(append(headerPrefix, encodeBlockNumber(number)...), numSuffix...))
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// GetHeader retrieves the block header corresponding to the hash, nil if none
// found.
func GetHeader(db DatabaseReader, hash common.Hash, number uint64) *Header {
	data := GetHeaderRLP(db, hash, number)
	if len(data) == 0 {
		return nil
	}
	header := new(Header)
	if err := rlp.Decode(bytes.NewReader(data), header); err != nil {
		log.Error("Invalid block header RLP", "hash", hash, "err", err)
		return nil
	}
	return header
}

// GetHeaderRLP retrieves a block header in its raw RLP database encoding, or nil
// if the header's not found.
func GetHeaderRLP(db DatabaseReader, hash common.Hash, number uint64) rlp.RawValue {
	data, _ := db.Get(headerKey(hash, number))
	return data
}

func headerKey(hash common.Hash, number uint64) []byte {
	return append(append(headerPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

// GetBodyRLP retrieves the block body (transactions and uncles) in RLP encoding.
func GetBodyRLP(db DatabaseReader, hash common.Hash, number uint64) rlp.RawValue {
	data, _ := db.Get(blockBodyKey(hash, number))
	return data
}

func blockBodyKey(hash common.Hash, number uint64) []byte {
	return append(append(bodyPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}
