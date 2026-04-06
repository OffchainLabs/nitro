// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrunner

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbnode/mel/extraction"
)

func TestLogsFetcher(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sequencerBatchDataABI := melextraction.SeqInboxABI.Events["SequencerBatchData"].ID
	batchBlockHash := common.HexToHash("blockContainingBatchTx")
	batchTxHash := common.HexToHash("batchTx")
	batchTxIndex := uint(1)
	batchTxLogs := []*types.Log{
		{
			Index:     0,
			Topics:    []common.Hash{melextraction.BatchDeliveredID},
			TxIndex:   batchTxIndex,
			TxHash:    batchTxHash,
			BlockHash: batchBlockHash,
		},
		{
			Index:     1,
			Topics:    []common.Hash{sequencerBatchDataABI},
			TxIndex:   batchTxIndex,
			TxHash:    batchTxHash,
			BlockHash: batchBlockHash,
		},
		{
			Index:     2,
			Topics:    []common.Hash{common.HexToHash("ignored")},
			TxIndex:   batchTxIndex,
			TxHash:    batchTxHash,
			BlockHash: batchBlockHash,
		},
	}
	messageDeliveredID := melextraction.IBridgeABI.Events["MessageDelivered"].ID
	delayedMessagePostingTargetAddress := common.HexToAddress("delayedMessagePostingTargetAddress")
	delayedBlockHash := common.HexToHash("blockContainingDelayedMsg")
	delayedMsgTxHash := common.HexToHash("delayedTx")
	delayedMsgTxIndex := uint(2)
	delayedMsgTxLogs := []*types.Log{
		{
			Index:     0,
			Topics:    []common.Hash{messageDeliveredID},
			TxIndex:   delayedMsgTxIndex,
			TxHash:    delayedMsgTxHash,
			BlockHash: delayedBlockHash,
			Address:   delayedMessagePostingTargetAddress,
		},
		{
			Index:     1,
			Topics:    []common.Hash{melextraction.InboxMessageFromOriginID},
			TxIndex:   delayedMsgTxIndex,
			TxHash:    delayedMsgTxHash,
			BlockHash: delayedBlockHash,
		},
		{
			Index:     2,
			Topics:    []common.Hash{melextraction.InboxMessageDeliveredID},
			TxIndex:   delayedMsgTxIndex,
			TxHash:    delayedMsgTxHash,
			BlockHash: delayedBlockHash,
		},
		{
			Index:     3,
			Topics:    []common.Hash{common.HexToHash("ignored")},
			TxIndex:   delayedMsgTxIndex,
			TxHash:    delayedMsgTxHash,
			BlockHash: delayedBlockHash,
		},
	}

	parentChainReader := &mockParentChainReader{logs: append(batchTxLogs, delayedMsgTxLogs...)}
	fetcher := newLogsAndHeadersFetcher(parentChainReader, 10)
	fetcher.chainHeight = 100
	melState := &mel.State{
		ParentChainBlockNumber:             1,
		DelayedMessagePostingTargetAddress: delayedMessagePostingTargetAddress,
	}
	require.NoError(t, fetcher.fetch(ctx, melState))

	// Verify that logsByBlockHash is correct
	require.True(t, len(fetcher.logsByBlockHash) == 2)
	require.True(t, fetcher.logsByBlockHash[batchBlockHash] != nil)
	require.True(t, fetcher.logsByBlockHash[delayedBlockHash] != nil)
	require.True(t, reflect.DeepEqual(fetcher.logsByBlockHash[batchBlockHash], batchTxLogs[:2]))        // last log shouldn't be returned by the filter query
	require.True(t, reflect.DeepEqual(fetcher.logsByBlockHash[delayedBlockHash], delayedMsgTxLogs[:3])) // last log shouldn't be returned by the filter query
	// Verify that logsByTxIndex is correct
	require.True(t, len(fetcher.logsByTxIndex) == 2) // for both delayed msg and sequencer batch
	require.True(t, fetcher.logsByTxIndex[batchBlockHash] != nil)
	require.True(t, fetcher.logsByTxIndex[delayedBlockHash] != nil)
	require.True(t, reflect.DeepEqual(fetcher.logsByTxIndex[batchBlockHash][batchTxIndex], batchTxLogs[:2]))             // last log shouldn't be returned by the filter query
	require.True(t, reflect.DeepEqual(fetcher.logsByTxIndex[delayedBlockHash][delayedMsgTxIndex], delayedMsgTxLogs[:3])) // last log shouldn't be returned by the filter query

	_, err := fetcher.getHeaderByNumber(ctx, 0)
	require.Error(t, err)
}

// TestLogsFetcher_NilHeaderFromParentChain verifies that fetch returns an
// error (instead of panicking) when the parent chain returns a nil header
// for the latest block query. This exercises the nil guard added at line 67.
func TestLogsFetcher_NilHeaderFromParentChain(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// nilHeaderParentChainReader returns (nil, nil) for all HeaderByNumber calls.
	parentChainReader := &nilHeaderParentChainReader{mockParentChainReader{
		blocks:  map[common.Hash]*types.Block{},
		headers: map[common.Hash]*types.Header{},
	}}
	fetcher := newLogsAndHeadersFetcher(parentChainReader, 10)
	// chainHeight = 0 forces the fetcher into the branch that queries the
	// parent chain for the latest block height.
	fetcher.chainHeight = 0

	melState := &mel.State{ParentChainBlockNumber: 0}
	err := fetcher.fetch(ctx, melState)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil header")
}

// TestLogsFetcher_NilHeaderNumber verifies that fetch returns an error when
// the parent chain returns a header with a nil Number field.
func TestLogsFetcher_NilHeaderNumber(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	parentChainReader := &nilNumberParentChainReader{mockParentChainReader{
		blocks:  map[common.Hash]*types.Block{},
		headers: map[common.Hash]*types.Header{},
	}}
	fetcher := newLogsAndHeadersFetcher(parentChainReader, 10)
	fetcher.chainHeight = 0

	melState := &mel.State{ParentChainBlockNumber: 0}
	err := fetcher.fetch(ctx, melState)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil")
}

// nilNumberParentChainReader returns a header with Number == nil.
type nilNumberParentChainReader struct {
	mockParentChainReader
}

func (m *nilNumberParentChainReader) HeaderByNumber(_ context.Context, _ *big.Int) (*types.Header, error) {
	return &types.Header{Number: nil}, nil
}
