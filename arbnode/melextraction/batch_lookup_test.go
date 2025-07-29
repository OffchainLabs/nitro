package melextraction

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

func Test_parseBatchesFromBlock(t *testing.T) {
	ctx := context.Background()
	batchPostingTargetAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	event, packedLog, wantedBatch := setupParseBatchesTest(t, big.NewInt(1))

	t.Run("block with no transactions", func(t *testing.T) {
		blockHeader := &types.Header{}
		blockBody := &types.Body{}
		block := types.NewBlock(
			blockHeader,
			blockBody,
			nil,
			trie.NewStackTrie(nil),
		)
		batches, txs, txIndices, err := parseBatchesFromBlock(
			ctx,
			nil,
			block.Header(),
			&mockTxsFetcher{},
			nil,
			nil,
		)
		require.NoError(t, err)
		require.Equal(t, 0, len(batches))
		require.Equal(t, 0, len(txs))
		require.Equal(t, 0, len(txIndices))
	})
	t.Run("block with no transactions with a to field", func(t *testing.T) {
		blockHeader := &types.Header{}
		txData := &types.DynamicFeeTx{
			To:        nil,
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
		txsFetcher := &mockTxsFetcher{
			txs: []*types.Transaction{tx},
		}
		block := types.NewBlock(
			blockHeader,
			blockBody,
			nil,
			trie.NewStackTrie(nil),
		)
		batches, txs, txIndices, err := parseBatchesFromBlock(
			ctx,
			&mel.State{MsgCount: 1},
			block.Header(),
			txsFetcher,
			nil,
			nil,
		)
		require.NoError(t, err)
		require.Equal(t, 0, len(batches))
		require.Equal(t, 0, len(txs))
		require.Equal(t, 0, len(txIndices))
	})
	t.Run("block with transactions not targeting the batch posting target address", func(t *testing.T) {
		blockHeader := &types.Header{}
		addr := common.BytesToAddress([]byte("deadbeef"))
		txData := &types.DynamicFeeTx{
			To:        &addr,
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
		txsFetcher := &mockTxsFetcher{
			txs: []*types.Transaction{tx},
		}
		block := types.NewBlock(
			blockHeader,
			blockBody,
			nil,
			trie.NewStackTrie(nil),
		)
		melState := &mel.State{
			BatchPostingTargetAddress: batchPostingTargetAddr,
			MsgCount:                  1,
		}
		batches, txs, txIndices, err := parseBatchesFromBlock(
			ctx,
			melState,
			block.Header(),
			txsFetcher,
			nil,
			nil,
		)
		require.NoError(t, err)
		require.Equal(t, 0, len(batches))
		require.Equal(t, 0, len(txs))
		require.Equal(t, 0, len(txIndices))
	})
	t.Run("bad receipt fetcher request", func(t *testing.T) {
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
		txsFetcher := &mockTxsFetcher{
			txs: []*types.Transaction{tx},
		}
		block := types.NewBlock(
			blockHeader,
			blockBody,
			nil,
			trie.NewStackTrie(nil),
		)
		melState := &mel.State{
			BatchPostingTargetAddress: batchPostingTargetAddr,
		}
		receiptFetcher := &mockReceiptFetcher{
			receipts: nil,
			err:      errors.New("oops"),
		}
		_, _, _, err := parseBatchesFromBlock(
			ctx,
			melState,
			block.Header(),
			txsFetcher,
			receiptFetcher,
			nil,
		)
		require.ErrorContains(t, err, "oops")
	})
	t.Run("transactions with no receipt logs", func(t *testing.T) {
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
		txsFetcher := &mockTxsFetcher{
			txs: []*types.Transaction{tx},
		}
		receipt := &types.Receipt{
			Logs: []*types.Log{},
		}
		receipts := []*types.Receipt{receipt}
		block := types.NewBlock(
			blockHeader,
			blockBody,
			receipts,
			trie.NewStackTrie(nil),
		)
		melState := &mel.State{
			BatchPostingTargetAddress: batchPostingTargetAddr,
		}
		receiptFetcher := &mockReceiptFetcher{
			receipts: receipts,
			err:      nil,
		}
		batches, txs, txIndices, err := parseBatchesFromBlock(
			ctx,
			melState,
			block.Header(),
			txsFetcher,
			receiptFetcher,
			nil,
		)
		require.NoError(t, err)
		require.Equal(t, 0, len(batches))
		require.Equal(t, 0, len(txs))
		require.Equal(t, 0, len(txIndices))
	})
	t.Run("receipt log with wrong topic id", func(t *testing.T) {
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
		receipt := &types.Receipt{
			Logs: []*types.Log{
				{
					Topics: []common.Hash{common.BytesToHash([]byte("wrong topic"))},
				},
			},
		}
		receipts := []*types.Receipt{receipt}
		block := types.NewBlock(
			blockHeader,
			blockBody,
			receipts,
			trie.NewStackTrie(nil),
		)
		melState := &mel.State{
			BatchPostingTargetAddress: batchPostingTargetAddr,
		}
		receiptFetcher := &mockReceiptFetcher{
			receipts: receipts,
			err:      nil,
		}
		txsFetcher := &mockTxsFetcher{
			txs: []*types.Transaction{tx},
		}
		batches, txs, txIndices, err := parseBatchesFromBlock(
			ctx,
			melState,
			block.Header(),
			txsFetcher,
			receiptFetcher,
			nil,
		)
		require.NoError(t, err)
		require.Equal(t, 0, len(batches))
		require.Equal(t, 0, len(txs))
		require.Equal(t, 0, len(txIndices))
	})
	t.Run("Unpack log fails", func(t *testing.T) {
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
		receipt := &types.Receipt{
			Logs: []*types.Log{
				{
					Topics: []common.Hash{batchDeliveredID},
					Data:   packedLog,
				},
			},
		}
		receipts := []*types.Receipt{receipt}
		block := types.NewBlock(
			blockHeader,
			blockBody,
			receipts,
			trie.NewStackTrie(nil),
		)
		melState := &mel.State{
			BatchPostingTargetAddress: batchPostingTargetAddr,
		}
		receiptFetcher := &mockReceiptFetcher{
			receipts: receipts,
			err:      nil,
		}
		eventUnpacker := &mockEventUnpacker{
			events: []*bridgegen.SequencerInboxSequencerBatchDelivered{event},
			idx:    0,
			err:    errors.New("oops event unpacking error"),
		}
		txsFetcher := &mockTxsFetcher{
			txs: []*types.Transaction{tx},
		}
		_, _, _, err := parseBatchesFromBlock(
			ctx,
			melState,
			block.Header(),
			txsFetcher,
			receiptFetcher,
			eventUnpacker,
		)
		require.ErrorContains(t, err, "oops event unpacking error")
	})
	t.Run("OK", func(t *testing.T) {
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
		receipt := &types.Receipt{
			Logs: []*types.Log{
				{
					Topics: []common.Hash{batchDeliveredID},
					Data:   packedLog,
				},
			},
		}
		receipts := []*types.Receipt{receipt}
		block := types.NewBlock(
			blockHeader,
			blockBody,
			receipts,
			trie.NewStackTrie(nil),
		)
		melState := &mel.State{
			BatchPostingTargetAddress: batchPostingTargetAddr,
		}
		receiptFetcher := &mockReceiptFetcher{
			receipts: receipts,
			err:      nil,
		}
		eventUnpacker := &mockEventUnpacker{
			events: []*bridgegen.SequencerInboxSequencerBatchDelivered{event},
			idx:    0,
		}
		txsFetcher := &mockTxsFetcher{
			txs: []*types.Transaction{tx},
		}
		batches, txs, txIndices, err := parseBatchesFromBlock(
			ctx,
			melState,
			block.Header(),
			txsFetcher,
			receiptFetcher,
			eventUnpacker,
		)
		require.NoError(t, err)

		require.Equal(t, 1, len(batches))
		require.Equal(t, 1, len(txs))
		require.Equal(t, 1, len(txIndices))
		require.Equal(t, wantedBatch.SequenceNumber, batches[0].SequenceNumber)
		require.Equal(t, wantedBatch.BeforeInboxAcc, batches[0].BeforeInboxAcc)
		require.Equal(t, wantedBatch.AfterInboxAcc, batches[0].AfterInboxAcc)
		require.Equal(t, wantedBatch.AfterDelayedAcc, batches[0].AfterDelayedAcc)
		require.Equal(t, wantedBatch.AfterDelayedCount, batches[0].AfterDelayedCount)
	})
}

func Test_parseBatchesFromBlock_outOfOrderBatches(t *testing.T) {
	ctx := context.Background()
	batchPostingTargetAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	event1, packedLog1, _ := setupParseBatchesTest(t, big.NewInt(2))
	event2, packedLog2, _ := setupParseBatchesTest(t, big.NewInt(1))

	blockHeader := &types.Header{}
	txData1 := &types.DynamicFeeTx{
		To:        &batchPostingTargetAddr,
		Nonce:     1,
		GasFeeCap: big.NewInt(1),
		GasTipCap: big.NewInt(1),
		Gas:       1,
		Value:     big.NewInt(0),
		Data:      nil,
	}
	txData2 := &types.DynamicFeeTx{
		To:        &batchPostingTargetAddr,
		Nonce:     2,
		GasFeeCap: big.NewInt(1),
		GasTipCap: big.NewInt(1),
		Gas:       1,
		Value:     big.NewInt(0),
		Data:      nil,
	}
	tx1 := types.NewTx(txData1)
	tx2 := types.NewTx(txData2)
	blockBody := &types.Body{
		Transactions: []*types.Transaction{tx1, tx2},
	}
	receipt := &types.Receipt{
		Logs: []*types.Log{
			{
				Topics: []common.Hash{batchDeliveredID},
				Data:   packedLog1,
			},
			{
				Topics: []common.Hash{batchDeliveredID},
				Data:   packedLog2,
			},
		},
	}
	receipts := []*types.Receipt{receipt}
	block := types.NewBlock(
		blockHeader,
		blockBody,
		receipts,
		trie.NewStackTrie(nil),
	)
	melState := &mel.State{
		BatchPostingTargetAddress: batchPostingTargetAddr,
	}
	receiptFetcher := &mockReceiptFetcher{
		receipts: receipts,
		err:      nil,
	}
	eventUnpacker := &mockEventUnpacker{
		events: []*bridgegen.SequencerInboxSequencerBatchDelivered{
			event1,
			event2,
		},
		idx: 0,
	}
	txsFetcher := &mockTxsFetcher{
		txs: []*types.Transaction{tx1, tx2},
	}
	_, _, _, err := parseBatchesFromBlock(
		ctx,
		melState,
		block.Header(),
		txsFetcher,
		receiptFetcher,
		eventUnpacker,
	)
	require.ErrorContains(t, err, "sequencer batches out of order")
}

func setupParseBatchesTest(t *testing.T, seqNumber *big.Int) (
	*bridgegen.SequencerInboxSequencerBatchDelivered,
	[]byte,
	*mel.SequencerInboxBatch,
) {
	event := &bridgegen.SequencerInboxSequencerBatchDelivered{
		BatchSequenceNumber:      seqNumber,
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
	wantedBatch := &mel.SequencerInboxBatch{
		SequenceNumber:    event.BatchSequenceNumber.Uint64(),
		BeforeInboxAcc:    event.BeforeAcc,
		AfterInboxAcc:     event.AfterAcc,
		AfterDelayedAcc:   event.DelayedAcc,
		AfterDelayedCount: event.AfterDelayedMessagesRead.Uint64(),
	}
	return event, packedLog, wantedBatch
}

type mockEventUnpacker struct {
	events []*bridgegen.SequencerInboxSequencerBatchDelivered
	idx    uint
	err    error
}

func (m *mockEventUnpacker) unpackLogTo(
	event any, abi *abi.ABI, eventName string, log types.Log,
) error {
	if m.err != nil {
		return m.err
	}
	ev, ok := event.(*bridgegen.SequencerInboxSequencerBatchDelivered)
	if !ok {
		return errors.New("wrong event type")
	}
	*ev = *m.events[m.idx]
	m.idx += 1
	return nil
}

type mockTxsFetcher struct {
	txs types.Transactions
	err error
}

func (m *mockTxsFetcher) TransactionsByHeader(
	ctx context.Context,
	parentChainHeaderHash common.Hash,
) (types.Transactions, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.txs, nil
}

type mockReceiptFetcher struct {
	receipts []*types.Receipt
	err      error
}

func (m *mockReceiptFetcher) ReceiptForTransactionIndex(
	_ context.Context,
	idx uint,
) (*types.Receipt, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.receipts[idx], nil
}
