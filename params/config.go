package params

import (
	"math/big"
)

const (
	GenesisGasLimit uint64 = 4712388 // Gas limit of the Genesis block.
)

var (
	GenesisDifficulty        = big.NewInt(131072) // Difficulty of the Genesis block.
	AllCliqueProtocolChanges = &ChainConfig{big.NewInt(1337), &CliqueConfig{Period: 0, Epoch: 30000}}
)

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	ChainId *big.Int `json:"chainId"` // Chain id identifies the current chain and is used for replay protection

	Clique *CliqueConfig `json:"clique,omitempty"`
}

// CliqueConfig is the consensus engine configs for proof-of-authority based sealing.
type CliqueConfig struct {
	Period uint64 `json:"period"` // Number of seconds between blocks to enforce
	Epoch  uint64 `json:"epoch"`  // Epoch length to reset votes and checkpoint
}
