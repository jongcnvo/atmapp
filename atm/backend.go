package atm

import (
	"../common"
	"../consensus"
	"../consensus/clique"
	"../core"
	"../core/bloombits"
	"../core/vm"
	"../db"
	"../event"
	"../log"
	"../node"
	"../p2p"
	"../params"
	"../rpc"
	"fmt"
	"math/big"
	"sync"
)

type ATM struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the ethereum
	stopDbUpgrade func() error // stop chain db sequential key upgrade

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	//lesServer       LesServer

	// DB interfaces
	chainDb db.Database // Block chain database

	eventMux *event.TypeMux
	engine   consensus.Engine
	//accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	//ApiBackend *ATMApiBackend

	//miner    *miner.Miner
	gasPrice *big.Int
	atmbase  common.Address

	networkId uint64
	//netRPCService *ethapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and etherbase)
}

// New creates a new Ethereum object (including the
// initialisation of the common Ethereum object)
func New(ctx *node.ServiceContext, config *Config) (*ATM, error) {
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	//stopDbUpgrade := upgradeDeduplicateData(chainDb)
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	eth := &ATM{
		config:      config,
		chainDb:     chainDb,
		chainConfig: chainConfig,
		eventMux:    ctx.EventMux,
		//accountManager: ctx.AccountManager,
		engine:       CreateConsensusEngine(ctx, chainConfig, chainDb),
		shutdownChan: make(chan bool),
		//stopDbUpgrade: stopDbUpgrade,
		networkId:     config.NetworkId,
		gasPrice:      config.GasPrice,
		bloomRequests: make(chan chan *bloombits.Retrieval),
		//bloomIndexer:  NewBloomIndexer(chainDb, params.BloomBitsBlocks),
	}

	log.Info("Initialising ATMChain protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run geth upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	}

	vmConfig := vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
	eth.blockchain, err = core.NewBlockChain(chainDb, eth.chainConfig, eth.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		eth.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	//eth.bloomIndexer.Start(eth.blockchain)

	//if config.TxPool.Journal != "" {
	//	config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	//}
	//eth.txPool = core.NewTxPool(config.TxPool, eth.chainConfig, eth.blockchain)

	//if eth.protocolManager, err = NewProtocolManager(eth.chainConfig, config.SyncMode, //config.NetworkId, eth.eventMux, eth.txPool, eth.engine, eth.blockchain, chainDb); //err != nil {
	//	return nil, err
	//}
	//eth.miner = miner.New(eth, eth.chainConfig, eth.EventMux(), eth.engine)
	//eth.miner.SetExtra(makeExtraData(config.ExtraData))

	//eth.ApiBackend = &EthApiBackend{eth, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	//eth.ApiBackend.gpo = gasprice.NewOracle(eth.ApiBackend, gpoParams)

	return eth, nil
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (db.Database, error) {
	atmdb, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if atmdb, ok := atmdb.(*db.LDBDatabase); ok {
		atmdb.Meter("atm/db/chaindata/")
	}
	return atmdb, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an Ethereum service
func CreateConsensusEngine(ctx *node.ServiceContext, chainConfig *params.ChainConfig, db db.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	return nil
}

// APIs returns the collection of RPC services the ATM package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *ATM) APIs() []rpc.API {
	//apis := ethapi.GetAPIs(s.ApiBackend)

	// Append any APIs exposed explicitly by the consensus engine
	//apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	/*return append(apis, []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicATMAPI(s),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
	*/
	return nil
}

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *ATM) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// protocol implementation.
func (s *ATM) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	//s.startBloomHandlers()

	// Start the RPC service
	//s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	/*maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		maxPeers -= s.config.LightPeers
		if maxPeers < srvr.MaxPeers/2 {
			maxPeers = srvr.MaxPeers / 2
		}
	}*/
	// Start the networking layer and the light server if requested
	//s.protocolManager.Start(maxPeers)
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// protocol.
func (s *ATM) Stop() error {
	/*if s.stopDbUpgrade != nil {
		s.stopDbUpgrade()
	}
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)*/

	return nil
}
