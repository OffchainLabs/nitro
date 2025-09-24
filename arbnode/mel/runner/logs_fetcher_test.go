package melrunner

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode/mel"
	melextraction "github.com/offchainlabs/nitro/arbnode/mel/extraction"
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
	fetcher := newLogsFetcher(parentChainReader, 10)
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
	require.True(t, len(fetcher.logsByTxIndex) == 1)
	require.True(t, fetcher.logsByTxIndex[batchBlockHash] != nil)
	require.True(t, reflect.DeepEqual(fetcher.logsByTxIndex[batchBlockHash][batchTxIndex], batchTxLogs[:2])) // last log shouldn't be returned by the filter query
}
