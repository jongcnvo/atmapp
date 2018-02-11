package core

import (
	"../common"
	"math/big"
)

// SyncMode represents the synchronisation mode of the downloader.
type SyncMode int

const (
	FullSync  SyncMode = iota // Synchronise the entire blockchain history from full blocks
	FastSync                  // Quickly download the headers, full sync only at the chain head
	LightSync                 // Download only the headers and terminate afterwards
)

type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the main net block is used.
	Genesis *Genesis `toml:",omitempty"`

	// Protocol options
	NetworkId uint64 // Network ID to use for selecting peers to connect to
	SyncMode  SyncMode

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int

	// Mining-related options
	ATMbase   common.Address `toml:",omitempty"`
	ExtraData []byte         `toml:",omitempty"`
	GasPrice  *big.Int

	// Transaction pool options
	TxPool TxPoolConfig

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Miscellaneous options
	DocRoot string `toml:"-"`
}

const (
	// These are the multipliers for ether denominations.
	// Example: To get the wei value of an amount in 'douglas', use
	//
	//    new(big.Int).Mul(value, big.NewInt(params.Douglas))
	//
	Wei      = 1
	Ada      = 1e3
	Babbage  = 1e6
	Shannon  = 1e9
	Szabo    = 1e12
	Finney   = 1e15
	Ether    = 1e18
	Einstein = 1e21
	Douglas  = 1e42
)

// DefaultConfig contains default settings for use on the Ethereum main net.
var DefaultConfig = Config{
	SyncMode:      FastSync,
	NetworkId:     1,
	DatabaseCache: 128,
	GasPrice:      big.NewInt(18 * Shannon),

	TxPool: DefaultTxPoolConfig,
}
