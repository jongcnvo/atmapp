package accounts

import (
	"reflect"
	"sync"

	"github.com/atmchain/atmapp/event"
)

// Manager is an overarching account manager that can communicate with various
// backends for signing transactions.
type Manager struct {
	backends map[reflect.Type][]Backend // Index of backends currently registered
	updaters []event.Subscription       // Wallet update subscriptions for all backends
	updates  chan WalletEvent           // Subscription sink for backend wallet changes
	wallets  []Wallet                   // Cache of all wallets from all registered backends

	feed event.Feed // Wallet feed notifying of arrivals/departures

	quit chan chan error
	lock sync.RWMutex
}
