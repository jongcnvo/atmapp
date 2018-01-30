package db

import (
	"../common"
	"errors"
	"sync"
)

type memdb struct {
	db   map[string][]byte
	lock sync.RWMutex
}

type kv struct{ k, v []byte }

type memBatch struct {
	db     *memdb
	writes []kv
	size   int
}

func (b *memBatch) Put(key, value []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), common.CopyBytes(value)})
	b.size += len(value)
	return nil
}

func (b *memBatch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, kv := range b.writes {
		b.db.db[string(kv.k)] = kv.v
	}
	return nil
}

func (b *memBatch) ValueSize() int {
	return b.size
}

func NewMemDB() (*memdb, error) {
	return &memdb{
		db: make(map[string][]byte),
	}, nil
}

func (mdb *memdb) NewBatch() Batch {
	return &memBatch{db: mdb}
}

func (mdb *memdb) Put(key []byte, value []byte) error {
	mdb.lock.Lock()
	defer mdb.lock.Unlock()

	mdb.db[string(key)] = common.CopyBytes(value)
	return nil
}

func (mdb *memdb) Has(key []byte) (bool, error) {
	mdb.lock.RLock()
	defer mdb.lock.RUnlock()

	_, ok := mdb.db[string(key)]
	return ok, nil
}

func (mdb *memdb) Get(key []byte) ([]byte, error) {
	mdb.lock.RLock()
	defer mdb.lock.RUnlock()

	if entry, ok := mdb.db[string(key)]; ok {
		return common.CopyBytes(entry), nil
	}
	return nil, errors.New("not found")
}

func (mdb *memdb) Delete(key []byte) error {
	mdb.lock.Lock()
	defer mdb.lock.Unlock()

	delete(mdb.db, string(key))
	return nil
}

func (mdb *memdb) Close() {}
