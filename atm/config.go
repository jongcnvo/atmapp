package atm

import (
	"../common"
	"../core"
	"../params"
	"math/big"
)

type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the Ethereum main net block is used.
	Genesis *core.Genesis `toml:",omitempty"`

	// Protocol options
	NetworkId uint64 // Network ID to use for selecting peers to connect to

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int

	// Mining-related options
	ATMBase   common.Address `toml:",omitempty"`
	ExtraData []byte         `toml:",omitempty"`
	GasPrice  *big.Int

	// Transaction pool options
	TxPool core.TxPoolConfig

	// Gas Price Oracle options
	GPO GasConfig

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Miscellaneous options
	DocRoot string `toml:"-"`
}

type GasConfig struct {
	Blocks     int
	Percentile int
	Default    *big.Int `toml:",omitempty"`
}

// DefaultConfig contains default settings for use on the Ethereum main net.
var DefaultConfig = Config{
	NetworkId:     1,
	DatabaseCache: 128,
	GasPrice:      big.NewInt(18 * params.Shannon),
	TxPool:        core.DefaultTxPoolConfig,
}
