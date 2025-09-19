// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package setup

import (
	"context"
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
)

var (
	_ protocol.ChainBackend = &SimulatedBackendWrapper{
		desiredBlockNum: rpc.LatestBlockNumber,
	}
)

type SimulatedBackendWrapper struct {
	lock sync.Mutex
	*simulated.Backend
	desiredBlockNum rpc.BlockNumber
}

func (s *SimulatedBackendWrapper) HeaderU64(ctx context.Context) (uint64, error) {
	header, err := s.Backend.Client().HeaderByNumber(ctx, big.NewInt(int64(s.desiredBlockNum)))
	if err != nil {
		return 0, err
	}
	if !header.Number.IsUint64() {
		return 0, errors.New("block number is not uint64")
	}
	return header.Number.Uint64(), nil
}

func (s *SimulatedBackendWrapper) ChainID(ctx context.Context) (*big.Int, error) {
	return s.Backend.Client().ChainID(ctx)
}

func (s *SimulatedBackendWrapper) Close() {
	s.Backend.Close() // #nosec G104
}

func (s *SimulatedBackendWrapper) Client() rpc.ClientInterface {
	iface, ok := s.Backend.Client().(rpc.ClientInterface)
	if !ok {
		panic("simulated backend client does not implement rpc.ClientInterface")
	}
	return iface
}

func (s *SimulatedBackendWrapper) Commit() common.Hash {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.Backend.Commit()
}

func NewSimulatedBackendWrapper(bk *simulated.Backend) *SimulatedBackendWrapper {
	return &SimulatedBackendWrapper{Backend: bk, desiredBlockNum: rpc.LatestBlockNumber}
}

func (s *SimulatedBackendWrapper) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	return s.Backend.Client().CodeAt(ctx, contract, blockNumber)
}

func (s *SimulatedBackendWrapper) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return s.Backend.Client().CallContract(ctx, call, blockNumber)
}

func (s *SimulatedBackendWrapper) PendingCodeAt(ctx context.Context, contract common.Address) ([]byte, error) {
	return s.Backend.Client().PendingCodeAt(ctx, contract)
}

func (s *SimulatedBackendWrapper) PendingCallContract(ctx context.Context, call ethereum.CallMsg) ([]byte, error) {
	return s.Backend.Client().PendingCallContract(ctx, call)
}

func (s *SimulatedBackendWrapper) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return s.Backend.Client().HeaderByNumber(ctx, number)
}

func (s *SimulatedBackendWrapper) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return s.Backend.Client().PendingNonceAt(ctx, account)
}

func (s *SimulatedBackendWrapper) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return s.Backend.Client().SuggestGasPrice(ctx)
}

func (s *SimulatedBackendWrapper) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return s.Backend.Client().SuggestGasTipCap(ctx)
}

func (s *SimulatedBackendWrapper) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	return s.Backend.Client().EstimateGas(ctx, call)
}

func (s *SimulatedBackendWrapper) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return s.Backend.Client().SendTransaction(ctx, tx)
}

func (s *SimulatedBackendWrapper) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	return s.Backend.Client().FilterLogs(ctx, query)
}

func (s *SimulatedBackendWrapper) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	return s.Backend.Client().SubscribeFilterLogs(ctx, query, ch)
}

func (s *SimulatedBackendWrapper) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	return s.Backend.Client().SubscribeNewHead(ctx, ch)
}

func (s *SimulatedBackendWrapper) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return s.Backend.Client().TransactionReceipt(ctx, txHash)
}

func (s *SimulatedBackendWrapper) TransactionByHash(ctx context.Context, txHash common.Hash) (*types.Transaction, bool, error) {
	return s.Backend.Client().TransactionByHash(ctx, txHash)
}
