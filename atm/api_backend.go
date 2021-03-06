package atm

import (
	"context"
	"math/big"

	"github.com/atmchain/atmapp/accounts"
	"github.com/atmchain/atmapp/atm/gasprice"
	"github.com/atmchain/atmapp/common"
	"github.com/atmchain/atmapp/common/math"
	"github.com/atmchain/atmapp/core"
	"github.com/atmchain/atmapp/core/state"
	"github.com/atmchain/atmapp/core/types"
	"github.com/atmchain/atmapp/core/vm"
	"github.com/atmchain/atmapp/db"
	"github.com/atmchain/atmapp/event"
	"github.com/atmchain/atmapp/params"
	"github.com/atmchain/atmapp/rpc"
)

// ATMApiBackend implements atmapi.Backend for full nodes
type ATMApiBackend struct {
	atm *ATM
	gpo *gasprice.Oracle
}

func (b *ATMApiBackend) ChainConfig() *params.ChainConfig {
	return b.atm.chainConfig
}

func (b *ATMApiBackend) CurrentBlock() *types.Block {
	return b.atm.blockchain.CurrentBlock()
}

func (b *ATMApiBackend) SetHead(number uint64) {
	b.atm.protocolManager.downloader.Cancel()
	b.atm.blockchain.SetHead(number)
}

func (b *ATMApiBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.atm.txPool.Get(hash)
}

func (b *ATMApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.atm.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.atm.blockchain.CurrentBlock().Header(), nil
	}
	return b.atm.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *ATMApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.atm.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.atm.blockchain.CurrentBlock(), nil
	}
	return b.atm.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *ATMApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.atm.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.atm.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *ATMApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.atm.blockchain.GetBlockByHash(blockHash), nil
}

func (b *ATMApiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return core.GetBlockReceipts(b.atm.chainDb, blockHash, core.GetBlockNumber(b.atm.chainDb, blockHash)), nil
}

func (b *ATMApiBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.atm.blockchain.GetTdByHash(blockHash)
}

func (b *ATMApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.atm.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *ATMApiBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.atm.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *ATMApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.atm.txPool.AddLocal(signedTx)
}

func (b *ATMApiBackend) AccountManager() *accounts.Manager {
	return b.atm.AccountManager()
}

func (b *ATMApiBackend) ChainDb() db.Database {
	return b.atm.ChainDb()
}

func (b *ATMApiBackend) ProtocolVersion() int {
	return b.atm.ATMVersion()
}

func (b *ATMApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *ATMApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.atm.txPool.State().GetNonce(addr), nil
}

func (b *ATMApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.atm.BlockChain(), nil)
	return vm.NewEVM(context, state, b.atm.chainConfig, vmCfg), vmError, nil
}
