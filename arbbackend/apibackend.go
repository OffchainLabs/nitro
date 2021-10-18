package arbbackend

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/bloombits"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

type ArbAPIBackend struct {
	b *ArbBackend
}

func createRegisterAPIBackend(backend *ArbBackend) {
	backend.apiBackend = &ArbAPIBackend{
		b: backend,
	}
	backend.stack.RegisterAPIs(backend.apiBackend.getAPIs())
}

func (a *ArbAPIBackend) getAPIs() []rpc.API {
	return eth.GetAPIsForBackend(a)
}

// General Ethereum API
func (a *ArbAPIBackend) SyncProgress() ethereum.SyncProgress {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) FeeHistory(ctx context.Context, blockCount int, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*big.Int, [][]*big.Int, []*big.Int, []float64, error) {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) ChainDb() ethdb.Database {
	return a.b.ethDatabase
}

func (a *ArbAPIBackend) AccountManager() *accounts.Manager {
	return a.b.stack.AccountManager()
}

func (a *ArbAPIBackend) ExtRPCEnabled() bool {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) RPCGasCap() uint64 {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) RPCTxFeeCap() float64 {
	return a.b.ethConfig.RPCTxFeeCap
}

func (a *ArbAPIBackend) UnprotectedAllowed() bool {
	return true // TODO: is that true?
}

// Blockchain API
func (a *ArbAPIBackend) SetHead(number uint64) {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	if number == rpc.LatestBlockNumber {
		return a.b.blockChain.CurrentBlock().Header(), nil
	}
	return a.b.blockChain.GetHeaderByNumber(uint64(number.Int64())), nil
}

func (a *ArbAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return a.b.blockChain.GetHeaderByHash(hash), nil
}

func (a *ArbAPIBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	number, isnum := blockNrOrHash.Number()
	if isnum {
		return a.HeaderByNumber(ctx, number)
	}
	hash, ishash := blockNrOrHash.Hash()
	if ishash {
		return a.HeaderByHash(ctx, hash)
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (a *ArbAPIBackend) CurrentHeader() *types.Header {
	return a.b.blockChain.CurrentHeader()
}

func (a *ArbAPIBackend) CurrentBlock() *types.Block {
	return a.b.blockChain.CurrentBlock()
}

func (a *ArbAPIBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	return a.b.blockChain.GetBlockByNumber(uint64(number.Int64())), nil
}

func (a *ArbAPIBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return a.b.blockChain.GetBlockByHash(hash), nil
}

func (a *ArbAPIBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	number, isnum := blockNrOrHash.Number()
	if isnum {
		return a.BlockByNumber(ctx, number)
	}
	hash, ishash := blockNrOrHash.Hash()
	if ishash {
		return a.BlockByHash(ctx, hash)
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (a *ArbAPIBackend) stateAndHeaderFromHeader(header *types.Header, err error) (*state.StateDB, *types.Header, error) {
	if err != nil {
		return nil, header, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	state, err := a.b.blockChain.StateAt(header.Root)
	return state, header, err
}

func (a *ArbAPIBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	return a.stateAndHeaderFromHeader(a.HeaderByNumber(ctx, number))
}

func (a *ArbAPIBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	return a.stateAndHeaderFromHeader(a.HeaderByNumberOrHash(ctx, blockNrOrHash))
}

func (a *ArbAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return a.b.blockChain.GetReceiptsByHash(hash), nil
}

func (a *ArbAPIBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmConfig *vm.Config) (*vm.EVM, func() error, error) {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return a.b.blockChain.SubscribeChainEvent(ch)
}

func (a *ArbAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return a.b.blockChain.SubscribeChainHeadEvent(ch)
}

func (a *ArbAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return a.b.blockChain.SubscribeChainSideEvent(ch)
}

// Transaction pool API
func (a *ArbAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return a.b.EnqueueL2Message(signedTx)
}

func (a *ArbAPIBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(a.b.ethDatabase, txHash)
	return tx, blockHash, blockNumber, index, nil
}

func (a *ArbAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) GetPoolTransaction(txHash common.Hash) *types.Transaction {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) Stats() (pending int, queued int) {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) TxPoolContentFrom(addr common.Address) (types.Transactions, types.Transactions) {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) SubscribeNewTxsEvent(_ chan<- core.NewTxsEvent) event.Subscription {
	panic("not implemented") // TODO: Implement
}

// Filter API
func (a *ArbAPIBackend) BloomStatus() (uint64, uint64) {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) GetLogs(ctx context.Context, blockHash common.Hash) ([][]*types.Log, error) {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	panic("not implemented") // TODO: Implement
}

func (a *ArbAPIBackend) ChainConfig() *params.ChainConfig {
	return a.b.blockChain.Config()
}

func (a *ArbAPIBackend) Engine() consensus.Engine {
	return a.b.blockChain.Engine()
}
