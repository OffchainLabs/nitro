package extractionfunction

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/nitro/arbnode"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/stretchr/testify/require"
)

func Test_parseBatchesFromBlock(t *testing.T) {
	ctx := context.Background()
	batchPostingTargetAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	event := &bridgegen.SequencerInboxSequencerBatchDelivered{
		BatchSequenceNumber:      big.NewInt(1),
		BeforeAcc:                common.BytesToHash([]byte{1}),
		AfterAcc:                 common.BytesToHash([]byte{2}),
		DelayedAcc:               common.BytesToHash([]byte{3}),
		AfterDelayedMessagesRead: big.NewInt(4),
		TimeBounds: bridgegen.IBridgeTimeBounds{
			MinTimestamp:   0,
			MaxTimestamp:   100,
			MinBlockNumber: 0,
			MaxBlockNumber: 100,
		},
		DataLocation: 1,
	}
	eventABI := seqInboxABI.Events["SequencerBatchDelivered"]
	packedLog, err := eventABI.Inputs.Pack(
		event.BatchSequenceNumber,
		event.BeforeAcc,
		event.AfterAcc,
		event.DelayedAcc,
		event.AfterDelayedMessagesRead,
		event.TimeBounds,
		event.DataLocation,
	)
	require.NoError(t, err)

	receipt := &types.Receipt{
		Logs: []*types.Log{
			{
				Topics: []common.Hash{batchDeliveredID},
				Data:   packedLog,
			},
		},
	}

	blockHeader := &types.Header{}
	txData := &types.DynamicFeeTx{
		To:        &batchPostingTargetAddr,
		Nonce:     1,
		GasFeeCap: big.NewInt(1),
		GasTipCap: big.NewInt(1),
		Gas:       1,
		Value:     big.NewInt(0),
		Data:      nil,
	}
	tx := types.NewTx(txData)
	blockBody := &types.Body{
		Transactions: []*types.Transaction{tx},
	}
	block := types.NewBlock(
		blockHeader,
		blockBody,
		[]*types.Receipt{receipt},
		trie.NewStackTrie(nil),
	)
	melState := &meltypes.State{
		BatchPostingTargetAddress: batchPostingTargetAddr,
	}
	receiptFetcher := &mockReceiptFetcher{
		receipt: receipt,
		err:     nil,
	}
	eventUnpacker := &mockEventUnpacker{
		returnEvent: event,
	}
	batches, txs, txIndices, err := parseBatchesFromBlock(
		ctx,
		melState,
		block,
		receiptFetcher,
		eventUnpacker,
	)
	require.NoError(t, err)

	wantedBatch := &arbnode.SequencerInboxBatch{
		SequenceNumber:    event.BatchSequenceNumber.Uint64(),
		BeforeInboxAcc:    event.BeforeAcc,
		AfterInboxAcc:     event.AfterAcc,
		AfterDelayedAcc:   event.DelayedAcc,
		AfterDelayedCount: event.AfterDelayedMessagesRead.Uint64(),
	}
	require.Equal(t, 1, len(batches))
	require.Equal(t, wantedBatch.SequenceNumber, batches[0].SequenceNumber)
	require.Equal(t, wantedBatch.BeforeInboxAcc, batches[0].BeforeInboxAcc)
	require.Equal(t, wantedBatch.AfterInboxAcc, batches[0].AfterInboxAcc)
	require.Equal(t, wantedBatch.AfterDelayedAcc, batches[0].AfterDelayedAcc)
	require.Equal(t, wantedBatch.AfterDelayedCount, batches[0].AfterDelayedCount)
	_ = txs
	_ = txIndices
}

type mockEventUnpacker struct {
	returnEvent *bridgegen.SequencerInboxSequencerBatchDelivered
}

func (m *mockEventUnpacker) unpackLogTo(
	event any, abi *abi.ABI, eventName string, log types.Log) error {
	ev, ok := event.(*bridgegen.SequencerInboxSequencerBatchDelivered)
	if !ok {
		return errors.New("wrong event type")
	}
	*ev = *m.returnEvent
	return nil
}

type mockReceiptFetcher struct {
	receipt *types.Receipt
	err     error
}

func (m *mockReceiptFetcher) ReceiptForTransactionIndex(
	_ context.Context,
	_ uint,
) (*types.Receipt, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.receipt, nil
}
