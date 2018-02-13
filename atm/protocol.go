package atm

import (
	"../common"
	"../core"
	"../core/types"
	"../event"
)

// Constants to match up protocol versions and messages
const (
	atm1 = 1
)

// Supported versions of the eth protocol (first is primary).
var ProtocolVersions = []uint{atm1}

type txPool interface {
	// AddRemotes should add the given transactions to the pool.
	AddRemotes([]*types.Transaction) []error

	// Pending should return pending transactions.
	// The slice should be modifiable by the caller.
	Pending() (map[common.Address]types.Transactions, error)

	// SubscribeTxPreEvent should return an event subscription of
	// TxPreEvent and send events to the given channel.
	SubscribeTxPreEvent(chan<- core.TxPreEvent) event.Subscription
}
