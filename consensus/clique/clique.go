package clique

import (
	"../../common"
	"../../core"
	"errors"
	lru "github.com/hashicorp/golang-lru"
)

var (
	extraSeal = 65 // Fixed number of extra-data suffix bytes reserved for signer seal

	errMissingSignature   = errors.New("extra-data 65 byte suffix signature missing")
	errInvalidVotingChain = errors.New("invalid voting chain")
	errUnauthorized       = errors.New("unauthorized")
)

// ecrecover extracts the Ethereum account address from a signed header.
func ecrecover(header *core.Header, sigcache *lru.ARCCache) (common.Address, error) {
	// If the signature's already cached, return that
	hash := header.Hash()
	if address, known := sigcache.Get(hash); known {
		return address.(common.Address), nil
	}
	// Retrieve the signature from the header extra-data
	if len(header.Extra) < extraSeal {
		return common.Address{}, errMissingSignature
	}
	signature := header.Extra[len(header.Extra)-extraSeal:]

	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(sigHash(header).Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	sigcache.Add(hash, signer)
	return signer, nil
}
