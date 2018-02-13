package core

import (
	"../common"
	"../db"
	"../log"
	"../params"
	"../rlp"
	"./types"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
)

var (
	headHeaderKey = []byte("LastHeader")
	headBlockKey  = []byte("LastBlock")
	headFastKey   = []byte("LastFast")

	// Data item prefixes (use single byte to avoid mixing data types, avoid `i`).
	headerPrefix        = []byte("h")                // headerPrefix + num (uint64 big endian) + hash -> header
	tdSuffix            = []byte("t")                // headerPrefix + num (uint64 big endian) + hash +
	blockHashPrefix     = []byte("H")                // blockHashPrefix + hash -> num (uint64 big endian)
	bodyPrefix          = []byte("b")                // bodyPrefix + num (uint64 big endian) + hash -> block body
	blockReceiptsPrefix = []byte("r")                // blockReceiptsPrefix + num (uint64 big endian) + hash ->
	numSuffix           = []byte("n")                // headerPrefix + num (uint64 big endian) + numSuffix -> hash
	configPrefix        = []byte("atmchain-config-") // config prefix for the db

	lookupPrefix   = []byte("l")   // lookupPrefix + hash -> transaction/receipt lookup metadata
	preimagePrefix = "secure-key-" // preimagePrefix + hash -> preimage

	ErrChainConfigNotFound = errors.New("ChainConfig not found") // general config not found error
)

// DatabaseDeleter wraps the Delete method of a backing data store.
type DatabaseDeleter interface {
	Delete(key []byte) error
}

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
func WriteBody(db db.Putter, hash common.Hash, number uint64, body *types.Body) error {
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
func WriteBlock(db db.Putter, block *types.Block) error {
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
func WriteBlockReceipts(db db.Putter, hash common.Hash, number uint64, receipts types.Receipts) error {
	// Convert the receipts into their storage form and serialize them
	storageReceipts := make([]*types.ReceiptForStorage, len(receipts))
	for i, receipt := range receipts {
		storageReceipts[i] = (*types.ReceiptForStorage)(receipt)
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
func WriteHeader(db db.Putter, header *types.Header) error {
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

// WriteHeadFastBlockHash stores the fast head block's hash.
func WriteHeadFastBlockHash(db db.Putter, hash common.Hash) error {
	if err := db.Put(headFastKey, hash.Bytes()); err != nil {
		log.Crit("Failed to store last fast block's hash", "err", err)
	}
	return nil
}

// GetChainConfig will fetch the network settings based on the given hash.
func GetChainConfig(db DatabaseReader, hash common.Hash) (*params.ChainConfig, error) {
	jsonChainConfig, _ := db.Get(append(configPrefix, hash[:]...))
	if len(jsonChainConfig) == 0 {
		return nil, ErrChainConfigNotFound
	}

	var config params.ChainConfig
	if err := json.Unmarshal(jsonChainConfig, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetCanonicalHash retrieves a hash assigned to a canonical block number.
func GetCanonicalHash(db DatabaseReader, number uint64) common.Hash {
	data, _ := db.Get(append(append(headerPrefix, encodeBlockNumber(number)...), numSuffix...))
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// missingNumber is returned by GetBlockNumber if no header with the
// given block hash has been stored in the database
const missingNumber = uint64(0xffffffffffffffff)

// GetBlockNumber returns the block number assigned to a block hash
// if the corresponding header is present in the database
func GetBlockNumber(db DatabaseReader, hash common.Hash) uint64 {
	data, _ := db.Get(append(blockHashPrefix, hash.Bytes()...))
	if len(data) != 8 {
		return missingNumber
	}
	return binary.BigEndian.Uint64(data)
}

// GetHeadHeaderHash retrieves the hash of the current canonical head block's
// header. The difference between this and GetHeadBlockHash is that whereas the
// last block hash is only updated upon a full block import, the last header
// hash is updated already at header import, allowing head tracking for the
// light synchronization mechanism.
func GetHeadHeaderHash(db DatabaseReader) common.Hash {
	data, _ := db.Get(headHeaderKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// GetHeader retrieves the block header corresponding to the hash, nil if none
// found.
func GetHeader(db DatabaseReader, hash common.Hash, number uint64) *types.Header {
	data := GetHeaderRLP(db, hash, number)
	if len(data) == 0 {
		return nil
	}
	header := new(types.Header)
	if err := rlp.Decode(bytes.NewReader(data), header); err != nil {
		log.Error("Invalid block header RLP", "hash", hash, "err", err)
		return nil
	}
	return header
}

// GetHeadBlockHash retrieves the hash of the current canonical head block.
func GetHeadBlockHash(db DatabaseReader) common.Hash {
	data, _ := db.Get(headBlockKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// GetHeadFastBlockHash retrieves the hash of the current canonical head block during
// fast synchronization. The difference between this and GetHeadBlockHash is that
// whereas the last block hash is only updated upon a full block import, the last
// fast hash is updated when importing pre-processed blocks.
func GetHeadFastBlockHash(db DatabaseReader) common.Hash {
	data, _ := db.Get(headFastKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
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

// GetBlockChainVersion reads the version number from db.
func GetBlockChainVersion(db DatabaseReader) int {
	var vsn uint
	enc, _ := db.Get([]byte("BlockchainVersion"))
	rlp.DecodeBytes(enc, &vsn)
	return int(vsn)
}

// WriteBlockChainVersion writes vsn as the version number to db.
func WriteBlockChainVersion(db db.Putter, vsn int) {
	enc, _ := rlp.EncodeToBytes(uint(vsn))
	db.Put([]byte("BlockchainVersion"), enc)
}

// DeleteBody removes all block body data associated with a hash.
func DeleteBody(db DatabaseDeleter, hash common.Hash, number uint64) {
	db.Delete(append(append(bodyPrefix, encodeBlockNumber(number)...), hash.Bytes()...))
}

// DeleteHeader removes all block header data associated with a hash.
func DeleteHeader(db DatabaseDeleter, hash common.Hash, number uint64) {
	db.Delete(append(blockHashPrefix, hash.Bytes()...))
	db.Delete(append(append(headerPrefix, encodeBlockNumber(number)...), hash.Bytes()...))
}

// DeleteTd removes all block total difficulty data associated with a hash.
func DeleteTd(db DatabaseDeleter, hash common.Hash, number uint64) {
	db.Delete(append(append(append(headerPrefix, encodeBlockNumber(number)...), hash.Bytes()...), tdSuffix...))
}

// DeleteCanonicalHash removes the number to hash canonical mapping.
func DeleteCanonicalHash(db DatabaseDeleter, number uint64) {
	db.Delete(append(append(headerPrefix, encodeBlockNumber(number)...), numSuffix...))
}

// GetBlock retrieves an entire block corresponding to the hash, assembling it
// back from the stored header and body. If either the header or body could not
// be retrieved nil is returned.
//
// Note, due to concurrent download of header and block body the header and thus
// canonical hash can be stored in the database but the body data not (yet).
func GetBlock(db DatabaseReader, hash common.Hash, number uint64) *types.Block {
	// Retrieve the block header and body contents
	header := GetHeader(db, hash, number)
	if header == nil {
		return nil
	}
	body := GetBody(db, hash, number)
	if body == nil {
		return nil
	}
	// Reassemble the block and return
	return types.NewBlockWithHeader(header).WithBody(body.Transactions, body.Uncles)
}

// GetTd retrieves a block's total difficulty corresponding to the hash, nil if
// none found.
func GetTd(db DatabaseReader, hash common.Hash, number uint64) *big.Int {
	data, _ := db.Get(append(append(append(headerPrefix, encodeBlockNumber(number)...), hash[:]...), tdSuffix...))
	if len(data) == 0 {
		return nil
	}
	td := new(big.Int)
	if err := rlp.Decode(bytes.NewReader(data), td); err != nil {
		log.Error("Invalid block total difficulty RLP", "hash", hash, "err", err)
		return nil
	}
	return td
}

// GetBody retrieves the block body (transactons, uncles) corresponding to the
// hash, nil if none found.
func GetBody(db DatabaseReader, hash common.Hash, number uint64) *types.Body {
	data := GetBodyRLP(db, hash, number)
	if len(data) == 0 {
		return nil
	}
	body := new(types.Body)
	if err := rlp.Decode(bytes.NewReader(data), body); err != nil {
		log.Error("Invalid block body RLP", "hash", hash, "err", err)
		return nil
	}
	return body
}

// TxLookupEntry is a positional metadata to help looking up the data content of
// a transaction or receipt given only its hash.
type TxLookupEntry struct {
	BlockHash  common.Hash
	BlockIndex uint64
	Index      uint64
}

// WriteTxLookupEntries stores a positional metadata for every transaction from
// a block, enabling hash based transaction and receipt lookups.
func WriteTxLookupEntries(db db.Putter, block *types.Block) error {
	// Iterate over each transaction and encode its metadata
	for i, tx := range block.Transactions() {
		entry := TxLookupEntry{
			BlockHash:  block.Hash(),
			BlockIndex: block.NumberU64(),
			Index:      uint64(i),
		}
		data, err := rlp.EncodeToBytes(entry)
		if err != nil {
			return err
		}
		if err := db.Put(append(lookupPrefix, tx.Hash().Bytes()...), data); err != nil {
			return err
		}
	}
	return nil
}

// PreimageTable returns a Database instance with the key prefix for preimage entries.
func PreimageTable(atmdb db.Database) db.Database {
	return db.NewTable(atmdb, preimagePrefix)
}

// WritePreimages writes the provided set of preimages to the database. `number` is the
// current block number, and is used for debug messages only.
func WritePreimages(db db.Database, number uint64, preimages map[common.Hash][]byte) error {
	table := PreimageTable(db)
	batch := table.NewBatch()
	hitCount := 0
	for hash, preimage := range preimages {
		if _, err := table.Get(hash.Bytes()); err != nil {
			batch.Put(hash.Bytes(), preimage)
			hitCount++
		}
	}
	//preimageCounter.Inc(int64(len(preimages)))
	//preimageHitCounter.Inc(int64(hitCount))
	if hitCount > 0 {
		if err := batch.Write(); err != nil {
			return fmt.Errorf("preimage write fail for block %d: %v", number, err)
		}
	}
	return nil
}

// GetBlockReceipts retrieves the receipts generated by the transactions included
// in a block given by its hash.
func GetBlockReceipts(db DatabaseReader, hash common.Hash, number uint64) types.Receipts {
	data, _ := db.Get(append(append(blockReceiptsPrefix, encodeBlockNumber(number)...), hash[:]...))
	if len(data) == 0 {
		return nil
	}
	storageReceipts := []*types.ReceiptForStorage{}
	if err := rlp.DecodeBytes(data, &storageReceipts); err != nil {
		log.Error("Invalid receipt array RLP", "hash", hash, "err", err)
		return nil
	}
	receipts := make(types.Receipts, len(storageReceipts))
	for i, receipt := range storageReceipts {
		receipts[i] = (*types.Receipt)(receipt)
	}
	return receipts
}

// DeleteTxLookupEntry removes all transaction data associated with a hash.
func DeleteTxLookupEntry(db DatabaseDeleter, hash common.Hash) {
	db.Delete(append(lookupPrefix, hash.Bytes()...))
}
