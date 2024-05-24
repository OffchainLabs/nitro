package endtoend

import (
	"context"
	"errors"
	"math/big"
	"math/rand"
	"testing"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	challengemanager "github.com/OffchainLabs/bold/challenge-manager"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func setupChallengeManager(
	t *testing.T,
	ctx context.Context,
	backend protocol.ChainBackend,
	rollup common.Address,
	sm l2stateprovider.Provider,
	txOpts *bind.TransactOpts,
	name string,
	opts ...challengemanager.Opt,
) *challengemanager.Manager {
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
	)
	require.NoError(t, err)

	manager, err := challengemanager.New(
		ctx,
		chain,
		sm,
		rollup,
		opts...,
	)
	require.NoError(t, err)
	return manager
}

func totalWasmOpcodes(heights *protocol.LayerZeroHeights, numBigSteps uint8) uint64 {
	totalWasmOpcodes := uint64(1)
	for i := uint8(0); i < numBigSteps; i++ {
		totalWasmOpcodes *= heights.BigStepChallengeHeight
	}
	totalWasmOpcodes *= heights.SmallStepChallengeHeight
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

type FlakyEthClient struct {
	*ethclient.Client
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
	return f.Client.TransactionReceipt(ctx, txHash)
}

func (f *FlakyEthClient) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.CodeAt(ctx, contract, blockNumber)
}

func (f *FlakyEthClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.CallContract(ctx, call, blockNumber)
}

func (f *FlakyEthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.HeaderByNumber(ctx, number)
}

func (f *FlakyEthClient) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.PendingCodeAt(ctx, account)
}

func (f *FlakyEthClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	if err := f.flaky(); err != nil {
		return 0, err
	}
	return f.Client.PendingNonceAt(ctx, account)
}

func (f *FlakyEthClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.SuggestGasPrice(ctx)
}

func (f *FlakyEthClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.SuggestGasTipCap(ctx)
}

func (f *FlakyEthClient) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	if err := f.flaky(); err != nil {
		return 0, err
	}
	return f.Client.EstimateGas(ctx, call)
}

func (f *FlakyEthClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	if err := f.flaky(); err != nil {
		return err
	}
	return f.Client.SendTransaction(ctx, tx)
}

func (f *FlakyEthClient) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.FilterLogs(ctx, query)
}
func (f *FlakyEthClient) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.SubscribeFilterLogs(ctx, query, ch)
}
