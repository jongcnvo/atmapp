package clique

import (
	"../../common"
	"../../core"
	"../../crypto"
	"../../db"
	"../../params"
	"bytes"
	"crypto/ecdsa"
	"math/big"
	"testing"
)

type testerVote struct {
	signer string
	voted  string
	auth   bool
}

// testerAccountPool is a pool to maintain currently active tester accounts,
// mapped from textual names used in the tests below to actual private
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

func (ap *testerAccountPool) address(account string) common.Address {
	// Ensure we have a persistent key for the account
	if ap.accounts[account] == nil {
		ap.accounts[account], _ = crypto.GenerateKey()
	}
	// Resolve and return the Ethereum address
	return crypto.PubkeyToAddress(ap.accounts[account].PublicKey)
}

// testerChainReader implements consensus.ChainReader to access the genesis
// block. All other methods and requests will panic.
type testerChainReader struct {
	db db.Database
}

func (r *testerChainReader) Config() *params.ChainConfig                { return params.AllCliqueProtocolChanges }
func (r *testerChainReader) CurrentHeader() *core.Header                { panic("not supported") }
func (r *testerChainReader) GetHeader(common.Hash, uint64) *core.Header { panic("not supported") }
func (r *testerChainReader) GetBlock(common.Hash, uint64) *core.Block   { panic("not supported") }
func (r *testerChainReader) GetHeaderByHash(common.Hash) *core.Header   { panic("not supported") }
func (r *testerChainReader) GetHeaderByNumber(number uint64) *core.Header {
	if number == 0 {
		return core.GetHeader(r.db, core.GetCanonicalHash(r.db, 0), 0)
	}
	panic("not supported")
}

// Tests that voting is evaluated correctly for various simple and complex scenarios.
func TestVoting(t *testing.T) {

	// Define the various voting scenarios to test
	tests := []struct {
		epoch   uint64
		signers []string
		votes   []testerVote
		results []string
	}{
		{
			// Single signer, no votes cast
			signers: []string{"A"},
			votes:   []testerVote{{signer: "A"}},
			results: []string{"A"},
		},
	}
	// Run through the scenarios and test them
	for i, tt := range tests {
		// Create the account pool and generate the initial set of signers
		accounts := newTesterAccountPool()

		signers := make([]common.Address, len(tt.signers))
		for j, signer := range tt.signers {
			signers[j] = accounts.address(signer)
		}
		for j := 0; j < len(signers); j++ {
			for k := j + 1; k < len(signers); k++ {
				if bytes.Compare(signers[j][:], signers[k][:]) > 0 {
					signers[j], signers[k] = signers[k], signers[j]
				}
			}
		}
		// Create the genesis block with the initial set of signers
		genesis := &core.Genesis{
			ExtraData: make([]byte, extraVanity+common.AddressLength*len(signers)+extraSeal),
		}
		for j, signer := range signers {
			copy(genesis.ExtraData[extraVanity+j*common.AddressLength:], signer[:])
		}
		// Create a pristine blockchain with the genesis injected
		db, _ := db.NewMemDB()
		genesis.Commit(db)

		// Assemble a chain of headers from the cast votes
		headers := make([]*core.Header, len(tt.votes))
		for j, vote := range tt.votes {
			headers[j] = &core.Header{
				Number:   big.NewInt(int64(j) + 1),
				Time:     big.NewInt(int64(j) * int64(blockPeriod)),
				Coinbase: accounts.address(vote.voted),
				Extra:    make([]byte, extraVanity+extraSeal),
			}
			if j > 0 {
				headers[j].ParentHash = headers[j-1].Hash()
			}
			if vote.auth {
				copy(headers[j].Nonce[:], nonceAuthVote)
			}
			accounts.sign(headers[j], vote.signer)
		}
		// Pass all the headers through clique and ensure tallying succeeds
		head := headers[len(headers)-1]

		snap, err := New(&params.CliqueConfig{Epoch: tt.epoch}, db).snapshot(&testerChainReader{db: db}, head.Number.Uint64(), head.Hash(), headers)
		if err != nil {
			t.Errorf("test %d: failed to create voting snapshot: %v", i, err)
			continue
		}
		// Verify the final list of signers against the expected ones
		signers = make([]common.Address, len(tt.results))
		for j, signer := range tt.results {
			signers[j] = accounts.address(signer)
		}
		for j := 0; j < len(signers); j++ {
			for k := j + 1; k < len(signers); k++ {
				if bytes.Compare(signers[j][:], signers[k][:]) > 0 {
					signers[j], signers[k] = signers[k], signers[j]
				}
			}
		}
		result := snap.signers()
		if len(result) != len(signers) {
			t.Errorf("test %d: signers mismatch: have %x, want %x", i, result, signers)
			continue
		}
		for j := 0; j < len(result); j++ {
			if !bytes.Equal(result[j][:], signers[j][:]) {
				t.Errorf("test %d, signer %d: signer mismatch: have %x, want %x", i, j, result[j], signers[j])
			}
		}
	}
}
