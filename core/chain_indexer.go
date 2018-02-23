package core

import (
	"github.com/atmchain/atmapp/core/types"
	"github.com/atmchain/atmapp/db"
	"github.com/atmchain/atmapp/log"
	"sync"
	"time"

	"github.com/atmchain/atmapp/common"
)

// ChainIndexerBackend defines the methods needed to process chain segments in
// the background and write the segment results into the database. These can be
// used to create filter blooms or CHTs.
type ChainIndexerBackend interface {
	// Reset initiates the processing of a new chain segment, potentially terminating
	// any partially completed operations (in case of a reorg).
	Reset(section uint64, prevHead common.Hash) error

	// Process crunches through the next header in the chain segment. The caller
	// will ensure a sequential order of headers.
	Process(header *types.Header)

	// Commit finalizes the section metadata and stores it into the database.
	Commit() error
}

// ChainIndexer does a post-processing job for equally sized sections of the
// canonical chain (like BlooomBits and CHT structures). A ChainIndexer is
// connected to the blockchain through the event system by starting a
// ChainEventLoop in a goroutine.
//
// Further child ChainIndexers can be added which use the output of the parent
// section indexer. These child indexers receive new head notifications only
// after an entire section has been finished or in case of rollbacks that might
// affect already finished sections.
type ChainIndexer struct {
	chainDb  db.Database         // Chain database to index the data from
	indexDb  db.Database         // Prefixed table-view of the db to write index metadata into
	backend  ChainIndexerBackend // Background processor generating the index data content
	children []*ChainIndexer     // Child indexers to cascade chain updates to

	active uint32          // Flag whether the event loop was started
	update chan struct{}   // Notification channel that headers should be processed
	quit   chan chan error // Quit channel to tear down running goroutines

	sectionSize uint64 // Number of blocks in a single chain segment to process
	confirmsReq uint64 // Number of confirmations before processing a completed segment

	storedSections uint64 // Number of sections successfully indexed into the database
	knownSections  uint64 // Number of sections known to be complete (block wise)
	cascadedHead   uint64 // Block number of the last completed section cascaded to subindexers

	throttling time.Duration // Disk throttling to prevent a heavy upgrade from hogging resources

	log  log.Logger
	lock sync.RWMutex
}
