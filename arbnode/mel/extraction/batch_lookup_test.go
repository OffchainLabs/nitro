package melextraction

import (
	"context"
	"errors"
	"fmt"
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

	t.Run("block with no logs", func(t *testing.T) {
		blockHeader := &types.Header{}
		blockBody := &types.Body{}
		block := types.NewBlock(
			blockHeader,
			blockBody,
			nil,
			trie.NewStackTrie(nil),
		)
		batches, txs, err := parseBatchesFromBlock(
			ctx,
			nil,
			block.Header(),
			nil,
			&mockBlockLogsFetcher{},
			nil,
		)
		require.NoError(t, err)
		require.Equal(t, 0, len(batches))
		require.Equal(t, 0, len(txs))
	})
	t.Run("bad block logs fetcher request", func(t *testing.T) {
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
		txFetcher := &mockTxFetcher{
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
		blockLogsFetcher := &mockBlockLogsFetcher{
			err: errors.New("oops"),
		}
		_, _, err := parseBatchesFromBlock(
			ctx,
			melState,
			block.Header(),
			txFetcher,
			blockLogsFetcher,
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
		txFetcher := &mockTxFetcher{
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
		blockLogsFetcher := &mockBlockLogsFetcher{
			err: nil,
		}
		batches, txs, err := parseBatchesFromBlock(
			ctx,
			melState,
			block.Header(),
			txFetcher,
			blockLogsFetcher,
			nil,
		)
		require.NoError(t, err)
		require.Equal(t, 0, len(batches))
		require.Equal(t, 0, len(txs))
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
		blockLogsFetcher := newMockBlockLogsFetcher(receipts)
		txFetcher := &mockTxFetcher{
			txs: []*types.Transaction{tx},
		}
		batches, txs, err := parseBatchesFromBlock(
			ctx,
			melState,
			block.Header(),
			txFetcher,
			blockLogsFetcher,
			nil,
		)
		require.NoError(t, err)
		require.Equal(t, 0, len(batches))
		require.Equal(t, 0, len(txs))
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
					Topics: []common.Hash{BatchDeliveredID},
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
		blockLogsFetcher := newMockBlockLogsFetcher(receipts)
		eventUnpacker := &mockEventUnpacker{
			events: []*bridgegen.SequencerInboxSequencerBatchDelivered{event},
			idx:    0,
			err:    errors.New("oops event unpacking error"),
		}
		txFetcher := &mockTxFetcher{
			txs: []*types.Transaction{tx},
		}
		_, _, err := parseBatchesFromBlock(
			ctx,
			melState,
			block.Header(),
			txFetcher,
			blockLogsFetcher,
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
					Topics: []common.Hash{BatchDeliveredID},
					Data:   packedLog,
					TxHash: tx.Hash(),
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
		blockLogsFetcher := newMockBlockLogsFetcher(receipts)
		eventUnpacker := &mockEventUnpacker{
			events: []*bridgegen.SequencerInboxSequencerBatchDelivered{event},
			idx:    0,
		}
		txFetcher := &mockTxFetcher{
			txs: []*types.Transaction{tx},
		}
		batches, txs, err := parseBatchesFromBlock(
			ctx,
			melState,
			block.Header(),
			txFetcher,
			blockLogsFetcher,
			eventUnpacker,
		)
		require.NoError(t, err)

		require.Equal(t, 1, len(batches))
		require.Equal(t, 1, len(txs))
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
				Topics: []common.Hash{BatchDeliveredID},
				Data:   packedLog1,
				TxHash: tx1.Hash(),
			},
			{
				Topics: []common.Hash{BatchDeliveredID},
				Data:   packedLog2,
				TxHash: tx2.Hash(),
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
	blockLogsFetcher := newMockBlockLogsFetcher(receipts)
	eventUnpacker := &mockEventUnpacker{
		events: []*bridgegen.SequencerInboxSequencerBatchDelivered{
			event1,
			event2,
		},
		idx: 0,
	}
	txFetcher := &mockTxFetcher{
		txs: []*types.Transaction{tx1, tx2},
	}
	_, _, err := parseBatchesFromBlock(
		ctx,
		melState,
		block.Header(),
		txFetcher,
		blockLogsFetcher,
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
	eventABI := SeqInboxABI.Events["SequencerBatchDelivered"]
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

type mockTxFetcher struct {
	txs types.Transactions
	err error
}

func (m *mockTxFetcher) TransactionByLog(
	ctx context.Context,
	log *types.Log,
) (*types.Transaction, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, tx := range m.txs {
		if tx.Hash() == log.TxHash {
			return tx, nil
		}
	}
	return nil, fmt.Errorf("tx: %v not found", log.TxHash)
}

type mockBlockLogsFetcher struct {
	blockLogs     []*types.Log
	logsByTxIndex map[common.Hash]map[uint][]*types.Log
	err           error
}

func newMockBlockLogsFetcher(blockReceipts []*types.Receipt) *mockBlockLogsFetcher {
	fetcher := &mockBlockLogsFetcher{logsByTxIndex: make(map[common.Hash]map[uint][]*types.Log)}
	for _, receipt := range blockReceipts {
		fetcher.blockLogs = append(fetcher.blockLogs, receipt.Logs...)
		if _, ok := fetcher.logsByTxIndex[receipt.BlockHash]; !ok {
			fetcher.logsByTxIndex[receipt.BlockHash] = make(map[uint][]*types.Log)
		}
		fetcher.logsByTxIndex[receipt.BlockHash][receipt.TransactionIndex] = receipt.Logs
	}
	return fetcher
}

func (m *mockBlockLogsFetcher) LogsForBlockHash(ctx context.Context, parentChainBlockHash common.Hash) ([]*types.Log, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.blockLogs, nil
}

func (m *mockBlockLogsFetcher) LogsForTxIndex(ctx context.Context, parentChainBlockHash common.Hash, txIndex uint) ([]*types.Log, error) {
	if m.err != nil {
		return nil, m.err
	}
	if indexMap, ok := m.logsByTxIndex[parentChainBlockHash]; ok {
		if _, ok := indexMap[txIndex]; ok {
			return indexMap[txIndex], nil
		}
	}
	return nil, fmt.Errorf("logs for blockHash: %v and txIndex: %d not found", parentChainBlockHash, txIndex)
}
