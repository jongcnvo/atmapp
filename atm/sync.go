package atm

import (
	"../core/types"
)

type txsync struct {
	p   *peer
	txs []*types.Transaction
}
