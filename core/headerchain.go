package core

import (
	"../common"
	"../consensus"
	"../db"
	"../log"
	"../params"
	"./types"
	crand "crypto/rand"
	"math"
	"math/big"
	mrand "math/rand"

	"github.com/hashicorp/golang-lru"
)

const (
	headerCacheLimit = 512
	tdCacheLimit     = 1024
	numberCacheLimit = 2048
)

// HeaderChain implements the basic block header chain logic that is shared by
// core.BlockChain and light.LightChain. It is not usable in itself, only as
// a part of either structure.
// It is not thread safe either, the encapsulating chain structures should do
// the necessary mutex locking/unlocking.
type HeaderChain struct {
	config *params.ChainConfig

	chainDb       db.Database
	genesisHeader *types.Header

	currentHeader     *types.Header // Current head of the header chain (may be above the block chain!)
	currentHeaderHash common.Hash   // Hash of the current head of the header chain (prevent recomputing all the time)

	headerCache *lru.Cache // Cache for the most recent block headers
	tdCache     *lru.Cache // Cache for the most recent block total difficulties
	numberCache *lru.Cache // Cache for the most recent block numbers

	procInterrupt func() bool

	rand   *mrand.Rand
	engine consensus.Engine
}

// NewHeaderChain creates a new HeaderChain structure.
//  getValidator should return the parent's validator
//  procInterrupt points to the parent's interrupt semaphore
//  wg points to the parent's shutdown wait group
func NewHeaderChain(chainDb db.Database, config *params.ChainConfig, engine consensus.Engine, procInterrupt func() bool) (*HeaderChain, error) {
	headerCache, _ := lru.New(headerCacheLimit)
	tdCache, _ := lru.New(tdCacheLimit)
	numberCache, _ := lru.New(numberCacheLimit)

	// Seed a fast but crypto originating random generator
	seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, err
	}

	hc := &HeaderChain{
		config:        config,
		chainDb:       chainDb,
		headerCache:   headerCache,
		tdCache:       tdCache,
		numberCache:   numberCache,
		procInterrupt: procInterrupt,
		rand:          mrand.New(mrand.NewSource(seed.Int64())),
		engine:        engine,
	}

	hc.genesisHeader = hc.GetHeaderByNumber(0)
	if hc.genesisHeader == nil {
		return nil, ErrNoGenesis
	}

	hc.currentHeader = hc.genesisHeader
	if head := GetHeadBlockHash(chainDb); head != (common.Hash{}) {
		if chead := hc.GetHeaderByHash(head); chead != nil {
			hc.currentHeader = chead
		}
	}
	hc.currentHeaderHash = hc.currentHeader.Hash()

	return hc, nil
}

// SetCurrentHeader sets the current head header of the canonical chain.
func (hc *HeaderChain) SetCurrentHeader(head *types.Header) {
	if err := WriteHeadHeaderHash(hc.chainDb, head.Hash()); err != nil {
		log.Crit("Failed to insert head header hash", "err", err)
	}
	hc.currentHeader = head
	hc.currentHeaderHash = head.Hash()
}

// DeleteCallback is a callback function that is called by SetHead before
// each header is deleted.
type DeleteCallback func(common.Hash, uint64)

// SetHead rewinds the local chain to a new head. Everything above the new head
// will be deleted and the new one set.
func (hc *HeaderChain) SetHead(head uint64, delFn DeleteCallback) {
	height := uint64(0)
	if hc.currentHeader != nil {
		height = hc.currentHeader.Number.Uint64()
	}

	for hc.currentHeader != nil && hc.currentHeader.Number.Uint64() > head {
		hash := hc.currentHeader.Hash()
		num := hc.currentHeader.Number.Uint64()
		if delFn != nil {
			delFn(hash, num)
		}
		DeleteHeader(hc.chainDb, hash, num)
		DeleteTd(hc.chainDb, hash, num)
		hc.currentHeader = hc.GetHeader(hc.currentHeader.ParentHash, hc.currentHeader.Number.Uint64()-1)
	}
	// Roll back the canonical chain numbering
	for i := height; i > head; i-- {
		DeleteCanonicalHash(hc.chainDb, i)
	}
	// Clear out any stale content from the caches
	hc.headerCache.Purge()
	hc.tdCache.Purge()
	hc.numberCache.Purge()

	if hc.currentHeader == nil {
		hc.currentHeader = hc.genesisHeader
	}
	hc.currentHeaderHash = hc.currentHeader.Hash()

	if err := WriteHeadHeaderHash(hc.chainDb, hc.currentHeaderHash); err != nil {
		log.Crit("Failed to reset head header hash", "err", err)
	}
}

// CurrentHeader retrieves the current head header of the canonical chain. The
// header is retrieved from the HeaderChain's internal cache.
func (hc *HeaderChain) CurrentHeader() *types.Header {
	return hc.currentHeader
}

// WriteTd stores a block's total difficulty into the database, also caching it
// along the way.
func (hc *HeaderChain) WriteTd(hash common.Hash, number uint64, td *big.Int) error {
	if err := WriteTd(hc.chainDb, hash, number, td); err != nil {
		return err
	}
	hc.tdCache.Add(hash, new(big.Int).Set(td))
	return nil
}

// SetGenesis sets a new genesis block header for the chain
func (hc *HeaderChain) SetGenesis(head *types.Header) {
	hc.genesisHeader = head
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (hc *HeaderChain) GetHeaderByNumber(number uint64) *types.Header {
	hash := GetCanonicalHash(hc.chainDb, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return hc.GetHeader(hash, number)
}

// GetHeaderByHash retrieves a block header from the database by hash, caching it if
// found.
func (hc *HeaderChain) GetHeaderByHash(hash common.Hash) *types.Header {
	return hc.GetHeader(hash, hc.GetBlockNumber(hash))
}

// GetHeader retrieves a block header from the database by hash and number,
// caching it if found.
func (hc *HeaderChain) GetHeader(hash common.Hash, number uint64) *types.Header {
	// Short circuit if the header's already in the cache, retrieve otherwise
	if header, ok := hc.headerCache.Get(hash); ok {
		return header.(*types.Header)
	}
	header := GetHeader(hc.chainDb, hash, number)
	if header == nil {
		return nil
	}
	// Cache the found header for next time and return
	hc.headerCache.Add(hash, header)
	return header
}

// GetBlockNumber retrieves the block number belonging to the given hash
// from the cache or database
func (hc *HeaderChain) GetBlockNumber(hash common.Hash) uint64 {
	if cached, ok := hc.numberCache.Get(hash); ok {
		return cached.(uint64)
	}
	number := GetBlockNumber(hc.chainDb, hash)
	if number != missingNumber {
		hc.numberCache.Add(hash, number)
	}
	return number
}

// GetTd retrieves a block's total difficulty in the canonical chain from the
// database by hash and number, caching it if found.
func (hc *HeaderChain) GetTd(hash common.Hash, number uint64) *big.Int {
	// Short circuit if the td's already in the cache, retrieve otherwise
	if cached, ok := hc.tdCache.Get(hash); ok {
		return cached.(*big.Int)
	}
	td := GetTd(hc.chainDb, hash, number)
	if td == nil {
		return nil
	}
	// Cache the found body for next time and return
	hc.tdCache.Add(hash, td)
	return td
}
