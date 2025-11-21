// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package endtoend

import (
	"context"
	"errors"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/chain-abstraction/sol-implementation"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

func setupAssertionChain(
	t *testing.T,
	ctx context.Context,
	backend protocol.ChainBackend,
	rollup common.Address,
	txOpts *bind.TransactOpts,
	opts ...solimpl.Opt,
) *solimpl.AssertionChain {
	t.Helper()
	assertionChainBinding, err := rollupgen.NewRollupUserLogic(
		rollup, backend,
	)
	require.NoError(t, err)
	challengeManagerAddr, err := assertionChainBinding.RollupUserLogicCaller.ChallengeManager(
		&bind.CallOpts{Context: ctx},
	)
	require.NoError(t, err)
	chain, err := solimpl.NewAssertionChain(
		ctx,
		rollup,
		challengeManagerAddr,
		txOpts,
		backend,
		solimpl.NewChainBackendTransactor(backend),
		opts...,
	)
	require.NoError(t, err)
	return chain
}

func totalWasmOpcodes(heights *protocol.LayerZeroHeights, numBigSteps uint8) uint64 {
	totalWasmOpcodes := uint64(1)
	for i := uint8(0); i < numBigSteps; i++ {
		totalWasmOpcodes *= heights.BigStepChallengeHeight.Uint64()
	}
	totalWasmOpcodes *= heights.SmallStepChallengeHeight.Uint64()
	return totalWasmOpcodes
}

// rand.Uint64() returns a random uint64 value.
// To get a value in the range [0, n), take the modulo n.
func randUint64(n uint64) uint64 {
	if n == 0 {
		return 0
	}
	return rand.Uint64() % n
}

func TestTotalWasmOpcodes(t *testing.T) {
	t.Run("2^43 production value", func(t *testing.T) {
		layerZeroHeights := &protocol.LayerZeroHeights{
			BlockChallengeHeight:     1 << 10,
			BigStepChallengeHeight:   1 << 10,
			SmallStepChallengeHeight: 1 << 13,
		}
		numBigSteps := uint8(3)
		require.Equal(t, uint64(1<<43), totalWasmOpcodes(layerZeroHeights, numBigSteps))
	})
	t.Run("minimal configuration", func(t *testing.T) {
		layerZeroHeights := &protocol.LayerZeroHeights{
			BlockChallengeHeight:     1 << 5,
			BigStepChallengeHeight:   1 << 5,
			SmallStepChallengeHeight: 1 << 5,
		}
		numBigSteps := uint8(1)
		require.Equal(t, uint64(1<<10), totalWasmOpcodes(layerZeroHeights, numBigSteps))
	})
}

var (
	_ protocol.ChainBackend = &FlakyEthClient{}
)

type FlakyEthClient struct {
	protocol.ChainBackend
}

func (f *FlakyEthClient) flaky() error {
	// 10% chance of error
	if rand.Intn(10) > 8 {
		return errors.New("flaky error")
	}
	return nil
}

func (f *FlakyEthClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.ChainBackend.TransactionReceipt(ctx, txHash)
}

func (f *FlakyEthClient) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.ChainBackend.CodeAt(ctx, contract, blockNumber)
}

func (f *FlakyEthClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.ChainBackend.CallContract(ctx, call, blockNumber)
}

func (f *FlakyEthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.ChainBackend.HeaderByNumber(ctx, number)
}

func (f *FlakyEthClient) HeaderU64(ctx context.Context) (uint64, error) {
	if err := f.flaky(); err != nil {
		return 0, err
	}
	return f.ChainBackend.HeaderU64(ctx)
}

func (f *FlakyEthClient) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.ChainBackend.PendingCodeAt(ctx, account)
}

func (f *FlakyEthClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	if err := f.flaky(); err != nil {
		return 0, err
	}
	return f.ChainBackend.PendingNonceAt(ctx, account)
}

func (f *FlakyEthClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.ChainBackend.SuggestGasPrice(ctx)
}

func (f *FlakyEthClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.ChainBackend.SuggestGasTipCap(ctx)
}

func (f *FlakyEthClient) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	if err := f.flaky(); err != nil {
		return 0, err
	}
	return f.ChainBackend.EstimateGas(ctx, call)
}

func (f *FlakyEthClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	if err := f.flaky(); err != nil {
		return err
	}
	return f.ChainBackend.SendTransaction(ctx, tx)
}

func (f *FlakyEthClient) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.ChainBackend.FilterLogs(ctx, query)
}
func (f *FlakyEthClient) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.ChainBackend.SubscribeFilterLogs(ctx, query, ch)
}
