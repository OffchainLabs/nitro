// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	melrunner "github.com/offchainlabs/nitro/arbnode/mel/runner"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
)

// stubParentChainReader satisfies melrunner.ParentChainReader for construction only.
type stubParentChainReader struct{}

func (s *stubParentChainReader) Client() rpc.ClientInterface { return nil }
func (s *stubParentChainReader) HeaderByNumber(context.Context, *big.Int) (*types.Header, error) {
	return nil, nil
}
func (s *stubParentChainReader) BlockByNumber(context.Context, *big.Int) (*types.Block, error) {
	return nil, nil
}
func (s *stubParentChainReader) BlockByHash(context.Context, common.Hash) (*types.Block, error) {
	return nil, nil
}
func (s *stubParentChainReader) HeaderByHash(context.Context, common.Hash) (*types.Header, error) {
	return nil, nil
}
func (s *stubParentChainReader) TransactionInBlock(context.Context, common.Hash, uint) (*types.Transaction, error) {
	return nil, nil
}
func (s *stubParentChainReader) TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error) {
	return nil, nil
}
func (s *stubParentChainReader) TransactionByHash(context.Context, common.Hash) (*types.Transaction, bool, error) {
	return nil, false, nil
}
func (s *stubParentChainReader) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	return nil, nil
}

func newTestMelBatchMetaFetcher(t *testing.T) *melBatchMetaFetcher {
	t.Helper()
	extractor, err := melrunner.NewMessageExtractor(
		melrunner.DefaultMessageExtractionConfig,
		&stubParentChainReader{},
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melrunner.NewDatabase(rawdb.NewMemoryDatabase()),
		daprovider.NewDAProviderRegistry(),
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	return newMelBatchMetaFetcher(extractor)
}

func TestMelBatchMetaFetcher_GetDelayedAccAlwaysErrors(t *testing.T) {
	t.Parallel()
	// melBatchMetaFetcher.GetDelayedAcc should always return an error
	// since MEL does not track delayed message accumulators.
	fetcher := &melBatchMetaFetcher{} // extractor not needed for GetDelayedAcc
	_, err := fetcher.GetDelayedAcc(0)
	require.ErrorContains(t, err, "MEL does not support delayed message accumulators")
}

func TestNewMelBatchMetaFetcher_PanicsOnNil(t *testing.T) {
	t.Parallel()
	require.Panics(t, func() {
		newMelBatchMetaFetcher(nil)
	})
}

func TestMelBatchMetaFetcher_DelegatesGetBatchCount(t *testing.T) {
	t.Parallel()
	fetcher := newTestMelBatchMetaFetcher(t)
	// Extractor is not started, so GetBatchCount should propagate the "not running" error.
	_, err := fetcher.GetBatchCount()
	require.ErrorContains(t, err, "not running")
}

func TestMelBatchMetaFetcher_DelegatesGetBatchMetadata(t *testing.T) {
	t.Parallel()
	fetcher := newTestMelBatchMetaFetcher(t)
	// Extractor is not started, so GetBatchMetadata should propagate the "not running" error.
	_, err := fetcher.GetBatchMetadata(0)
	require.ErrorContains(t, err, "not running")
}
