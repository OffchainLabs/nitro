package setup

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
)

var (
	_ bind.ContractBackend = &SimulatedBackendWrapper{}
)

type SimulatedBackendWrapper struct {
	*simulated.Backend
}

func NewSimulatedBackendWrapper(bk *simulated.Backend) *SimulatedBackendWrapper {
	return &SimulatedBackendWrapper{bk}
}

func (s *SimulatedBackendWrapper) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	return s.Client().CodeAt(ctx, contract, blockNumber)
}

func (s *SimulatedBackendWrapper) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return s.Client().CallContract(ctx, call, blockNumber)
}

func (s *SimulatedBackendWrapper) PendingCodeAt(ctx context.Context, contract common.Address) ([]byte, error) {
	return s.Client().PendingCodeAt(ctx, contract)
}

func (s *SimulatedBackendWrapper) PendingCallContract(ctx context.Context, call ethereum.CallMsg) ([]byte, error) {
	return s.Client().PendingCallContract(ctx, call)
}

func (s *SimulatedBackendWrapper) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return s.Client().HeaderByNumber(ctx, number)
}

func (s *SimulatedBackendWrapper) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return s.Client().PendingNonceAt(ctx, account)
}

func (s *SimulatedBackendWrapper) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return s.Client().SuggestGasPrice(ctx)
}

func (s *SimulatedBackendWrapper) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return s.Client().SuggestGasTipCap(ctx)
}

func (s *SimulatedBackendWrapper) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	return s.Client().EstimateGas(ctx, call)
}

func (s *SimulatedBackendWrapper) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return s.Client().SendTransaction(ctx, tx)
}

func (s *SimulatedBackendWrapper) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	return s.Client().FilterLogs(ctx, query)
}

func (s *SimulatedBackendWrapper) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	return s.Client().SubscribeFilterLogs(ctx, query, ch)
}

func (s *SimulatedBackendWrapper) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	return s.Client().SubscribeNewHead(ctx, ch)
}

func (s *SimulatedBackendWrapper) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return s.Client().TransactionReceipt(ctx, txHash)
}

func (s *SimulatedBackendWrapper) TransactionByHash(ctx context.Context, txHash common.Hash) (*types.Transaction, bool, error) {
	return s.Client().TransactionByHash(ctx, txHash)
}
