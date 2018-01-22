package clique

import (
	"../../core"
	"../../crypto"
	"crypto/ecdsa"
)

type testerVote struct {
	signer string
	voted  string
	auth   bool
}

// testerAccountPool is a pool to maintain currently active tester accounts,
// mapped from textual names used in the tests below to actual Ethereum private
// keys capable of signing transactions.
type testerAccountPool struct {
	accounts map[string]*ecdsa.PrivateKey
}

func newTesterAccountPool() *testerAccountPool {
	return &testerAccountPool{
		accounts: make(map[string]*ecdsa.PrivateKey),
	}
}

func (ap *testerAccountPool) sign(header *core.Header, signer string) {
	// Ensure we have a persistent key for the signer
	if ap.accounts[signer] == nil {
		ap.accounts[signer], _ = crypto.GenerateKey()
	}
	// Sign the header and embed the signature in extra data
	sig, _ := crypto.Sign(sigHash(header).Bytes(), ap.accounts[signer])
	copy(header.Extra[len(header.Extra)-65:], sig)
}
