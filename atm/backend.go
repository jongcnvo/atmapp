package atm

import (
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/atmchain/atmapp/accounts"
	"github.com/atmchain/atmapp/common"
	"github.com/atmchain/atmapp/consensus"
	"github.com/atmchain/atmapp/consensus/clique"
	"github.com/atmchain/atmapp/core"
	"github.com/atmchain/atmapp/core/bloombits"
	"github.com/atmchain/atmapp/core/vm"
	"github.com/atmchain/atmapp/db"
	"github.com/atmchain/atmapp/event"
	"github.com/atmchain/atmapp/log"
	"github.com/atmchain/atmapp/miner"
	"github.com/atmchain/atmapp/node"
	"github.com/atmchain/atmapp/p2p"
	"github.com/atmchain/atmapp/params"
	"github.com/atmchain/atmapp/rlp"
	"github.com/atmchain/atmapp/rpc"
	"github.com/atmchain/atmapp/utils/atmapi"
)

type ATM struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the ATMChain
	stopDbUpgrade func() error // stop chain db sequential key upgrade

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager

	// DB interfaces
	chainDb db.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend *ATMApiBackend

	miner    *miner.Miner
	gasPrice *big.Int
	atmbase  common.Address

	networkId uint64
	//netRPCService *atmapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and etherbase)
}

// New creates a new ATM object (including the
// initialisation of the common ATMChain object)
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

	atm := &ATM{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
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
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run atmapp upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	}

	vmConfig := vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
	atm.blockchain, err = core.NewBlockChain(chainDb, atm.chainConfig, atm.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		atm.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	//eth.bloomIndexer.Start(eth.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	atm.txPool = core.NewTxPool(config.TxPool, atm.chainConfig, atm.blockchain)

	if atm.protocolManager, err = NewProtocolManager(atm.chainConfig, 0, config.NetworkId, atm.eventMux, atm.txPool, atm.engine, atm.blockchain, chainDb); err != nil {
		return nil, err
	}

	atm.miner = miner.New(atm, atm.chainConfig, atm.EventMux(), atm.engine)
	atm.miner.SetExtra(makeExtraData(config.ExtraData))

	atm.ApiBackend = &ATMApiBackend{atm}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	//eth.ApiBackend.gpo = gasprice.NewOracle(eth.ApiBackend, gpoParams)

	return atm, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"atm",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", common.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
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

// CreateConsensusEngine creates the required type of consensus engine instance for an ATMChain service
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
	apis := atmapi.GetAPIs(s.ApiBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		/*
			{
				Namespace: "atm",
				Version:   "1.0",
				Service:   NewPublicATMAPI(s),
				Public:    true,
			}, {
				Namespace: "atm",
				Version:   "1.0",
				Service:   NewPublicMinerAPI(s),
				Public:    true,
			}, {
				Namespace: "atm",
				Version:   "1.0",
				Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
				Public:    true,
			}, {
				Namespace: "atm",
				Version:   "1.0",
				Service:   NewPrivateMinerAPI(s),
				Public:    false,
			}, {
				Namespace: "atm",
				Version:   "1.0",
				Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
				Public:    true,
			}, */
		{
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		},
		/*, {
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
		},*/
	}...)
}

func (s *ATM) BlockChain() *core.BlockChain { return s.blockchain }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *ATM) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

func (s *ATM) ATMbase() (eb common.Address, err error) {
	s.lock.RLock()
	atmbase := s.atmbase
	s.lock.RUnlock()

	if atmbase != (common.Address{}) {
		return atmbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			etherbase := accounts[0].Address

			s.lock.Lock()
			s.atmbase = atmbase
			s.lock.Unlock()

			log.Info("ATMChain automatically configured", "address", etherbase)
			return etherbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("atmbase must be explicitly specified")
}

func (s *ATM) StartMining(local bool) error {
	eb, err := s.ATMbase()
	if err != nil {
		log.Error("Cannot start mining without atmbase", "err", err)
		return fmt.Errorf("atmrbase missing: %v", err)
	}
	if clique, ok := s.engine.(*clique.Clique); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("ATMbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		clique.Authorize(eb, wallet.SignHash)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so noone will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(eb)
	return nil
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

func (s *ATM) Engine() consensus.Engine          { return s.engine }
func (s *ATM) EventMux() *event.TypeMux          { return s.eventMux }
func (s *ATM) ChainDb() db.Database              { return s.chainDb }
func (s *ATM) TxPool() *core.TxPool              { return s.txPool }
func (s *ATM) AccountManager() *accounts.Manager { return s.accountManager }
