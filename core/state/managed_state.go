package state

import (
	"../../common"
	"sync"
)

type account struct {
	stateObject *stateObject
	nstart      uint64
	nonces      []bool
}

type ManagedState struct {
	*StateDB
	mu       sync.RWMutex
	accounts map[common.Address]*account
}
