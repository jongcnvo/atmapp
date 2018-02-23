package state

import (
	"sync"

	"github.com/atmchain/atmapp/common"
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

// ManagedState returns a new managed state with the statedb as it's backing layer
func ManageState(statedb *StateDB) *ManagedState {
	return &ManagedState{
		StateDB:  statedb.Copy(),
		accounts: make(map[common.Address]*account),
	}
}

// GetNonce returns the canonical nonce for the managed or unmanaged account.
//
// Because GetNonce mutates the DB, we must take a write lock.
func (ms *ManagedState) GetNonce(addr common.Address) uint64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.hasAccount(addr) {
		account := ms.getAccount(addr)
		return uint64(len(account.nonces)) + account.nstart
	} else {
		return ms.StateDB.GetNonce(addr)
	}
}

func (ms *ManagedState) hasAccount(addr common.Address) bool {
	_, ok := ms.accounts[addr]
	return ok
}

// populate the managed state
func (ms *ManagedState) getAccount(addr common.Address) *account {
	if account, ok := ms.accounts[addr]; !ok {
		so := ms.GetOrNewStateObject(addr)
		ms.accounts[addr] = newAccount(so)
	} else {
		// Always make sure the state account nonce isn't actually higher
		// than the tracked one.
		so := ms.StateDB.getStateObject(addr)
		if so != nil && uint64(len(account.nonces))+account.nstart < so.Nonce() {
			ms.accounts[addr] = newAccount(so)
		}

	}

	return ms.accounts[addr]
}

func newAccount(so *stateObject) *account {
	return &account{so, so.Nonce(), nil}
}
