package clique

import (
	"github.com/atmchain/atmapp/consensus"
)

// API is a user facing RPC API to allow controlling the signer and voting
// mechanisms of the proof-of-authority scheme.
type API struct {
	chain  consensus.ChainReader
	clique *Clique
}
