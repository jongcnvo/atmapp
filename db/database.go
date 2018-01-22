package db

import (
	"../log"
	gometrics "github.com/rcrowley/go-metrics"
	"github.com/syndtr/goleveldb/leveldb"
	"sync"
	"time"
)

// Putter wraps the database write operation supported by both batches and regular databases.
type Putter interface {
	Put(key []byte, value []byte) error
}

// Database wraps all database operations. All methods are safe for concurrent use.
type Database interface {
	Putter
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
	Delete(key []byte) error
	Close()
	NewBatch() Batch
}

// Batch is a write-only database that commits changes to its host database
// when Write is called. Batch cannot be used concurrently.
type Batch interface {
	Putter
	ValueSize() int // amount of data in the batch
	Write() error
}

type LDBDatabase struct {
	fn string      // filename for reporting
	db *leveldb.DB // LevelDB instance

	getTimer       gometrics.Timer // Timer for measuring the database get request counts and latencies
	putTimer       gometrics.Timer // Timer for measuring the database put request counts and latencies
	delTimer       gometrics.Timer // Timer for measuring the database delete request counts and latencies
	missMeter      gometrics.Meter // Meter for measuring the missed database get requests
	readMeter      gometrics.Meter // Meter for measuring the database get request data usage
	writeMeter     gometrics.Meter // Meter for measuring the database put request data usage
	compTimeMeter  gometrics.Meter // Meter for measuring the total time spent in database compaction
	compReadMeter  gometrics.Meter // Meter for measuring the data read during compaction
	compWriteMeter gometrics.Meter // Meter for measuring the data written during compaction

	quitLock sync.Mutex      // Mutex protecting the quit channel access
	quitChan chan chan error // Quit channel to stop the metrics collection before closing the database

	log log.Logger // Contextual logger tracking the database path
}

// Get returns the given key if it's present.
func (db *LDBDatabase) Get(key []byte) ([]byte, error) {
	// Measure the database get latency, if requested
	if db.getTimer != nil {
		defer db.getTimer.UpdateSince(time.Now())
	}
	// Retrieve the key and increment the miss counter if not found
	dat, err := db.db.Get(key, nil)
	if err != nil {
		if db.missMeter != nil {
			db.missMeter.Mark(1)
		}
		return nil, err
	}
	// Otherwise update the actually retrieved amount of data
	if db.readMeter != nil {
		db.readMeter.Mark(int64(len(dat)))
	}
	return dat, nil
	//return rle.Decompress(dat)
}
