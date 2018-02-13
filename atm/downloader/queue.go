package downloader

import (
	"../../common"
	"../../core/types"
	"../../log"
	"sync"
	"time"

	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

// queue represents hashes that are either need fetching or are being fetched
type queue struct {
	mode          SyncMode // Synchronisation mode to decide on the block parts to schedule for fetching
	fastSyncPivot uint64   // Block number where the fast sync pivots into archive synchronisation mode

	headerHead common.Hash // [eth/62] Hash of the last queued header to verify order

	// Headers are "special", they download in batches, supported by a skeleton chain
	headerTaskPool  map[uint64]*types.Header       // [eth/62] Pending header retrieval tasks, mapping starting indexes to skeleton headers
	headerTaskQueue *prque.Prque                   // [eth/62] Priority queue of the skeleton indexes to fetch the filling headers for
	headerPeerMiss  map[string]map[uint64]struct{} // [eth/62] Set of per-peer header batches known to be unavailable
	headerPendPool  map[string]*fetchRequest       // [eth/62] Currently pending header retrieval operations
	headerResults   []*types.Header                // [eth/62] Result cache accumulating the completed headers
	headerProced    int                            // [eth/62] Number of headers already processed from the results
	headerOffset    uint64                         // [eth/62] Number of the first header in the result cache
	headerContCh    chan bool                      // [eth/62] Channel to notify when header download finishes

	// All data retrievals below are based on an already assembles header chain
	blockTaskPool  map[common.Hash]*types.Header // [eth/62] Pending block (body) retrieval tasks, mapping hashes to headers
	blockTaskQueue *prque.Prque                  // [eth/62] Priority queue of the headers to fetch the blocks (bodies) for
	blockPendPool  map[string]*fetchRequest      // [eth/62] Currently pending block (body) retrieval operations
	blockDonePool  map[common.Hash]struct{}      // [eth/62] Set of the completed block (body) fetches

	receiptTaskPool  map[common.Hash]*types.Header // [eth/63] Pending receipt retrieval tasks, mapping hashes to headers
	receiptTaskQueue *prque.Prque                  // [eth/63] Priority queue of the headers to fetch the receipts for
	receiptPendPool  map[string]*fetchRequest      // [eth/63] Currently pending receipt retrieval operations
	receiptDonePool  map[common.Hash]struct{}      // [eth/63] Set of the completed receipt fetches

	resultCache  []*fetchResult // Downloaded but not yet delivered fetch results
	resultOffset uint64         // Offset of the first cached fetch result in the block chain

	lock   *sync.Mutex
	active *sync.Cond
	closed bool
}

// fetchRequest is a currently running data retrieval operation.
type fetchRequest struct {
	Peer    *peerConnection     // Peer to which the request was sent
	From    uint64              // [eth/62] Requested chain element index (used for skeleton fills only)
	Hashes  map[common.Hash]int // [eth/61] Requested hashes with their insertion index (priority)
	Headers []*types.Header     // [eth/62] Requested headers, sorted by request order
	Time    time.Time           // Time when the request was made
}

// fetchResult is a struct collecting partial results from data fetchers until
// all outstanding pieces complete and the result as a whole can be processed.
type fetchResult struct {
	Pending int // Number of data fetches still pending

	Header       *types.Header
	Uncles       []*types.Header
	Transactions types.Transactions
	Receipts     types.Receipts
}

// peerConnection represents an active peer from which hashes and blocks are retrieved.
type peerConnection struct {
	id string // Unique identifier of the peer

	headerIdle  int32 // Current header activity state of the peer (idle = 0, active = 1)
	blockIdle   int32 // Current block activity state of the peer (idle = 0, active = 1)
	receiptIdle int32 // Current receipt activity state of the peer (idle = 0, active = 1)
	stateIdle   int32 // Current node data activity state of the peer (idle = 0, active = 1)

	headerThroughput  float64 // Number of headers measured to be retrievable per second
	blockThroughput   float64 // Number of blocks (bodies) measured to be retrievable per second
	receiptThroughput float64 // Number of receipts measured to be retrievable per second
	stateThroughput   float64 // Number of node data pieces measured to be retrievable per second

	rtt time.Duration // Request round trip time to track responsiveness (QoS)

	headerStarted  time.Time // Time instance when the last header fetch was started
	blockStarted   time.Time // Time instance when the last block (body) fetch was started
	receiptStarted time.Time // Time instance when the last receipt fetch was started
	stateStarted   time.Time // Time instance when the last node data fetch was started

	lacking map[common.Hash]struct{} // Set of hashes not to request (didn't have previously)

	peer Peer

	version int        // Eth protocol version number to switch strategies
	log     log.Logger // Contextual logger to add extra infos to peer logs
	lock    sync.RWMutex
}
