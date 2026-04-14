// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrunner

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
)

var _ ParentChainReader = (*mockParentChainReader)(nil)

func TestMessageExtractorStallTriggersMetric(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := DefaultMessageExtractionConfig
	cfg.StallTolerance = 2
	cfg.RetryInterval = 100 * time.Millisecond
	extractor, err := NewMessageExtractor(
		cfg,
		&mockParentChainReader{},
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		NewDatabase(rawdb.NewMemoryDatabase()),
		daprovider.NewDAProviderRegistry(),
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, extractor.SetMessageConsumer(&mockMessageConsumer{}))
	require.True(t, stuckFSMIndicatingGauge.Snapshot().Value() == 0)
	require.NoError(t, extractor.Start(ctx))
	// MEL will be stuck at the 'Start' state as HeadMelState is not yet stored in the db
	// so after RetryInterval*StallTolerance amount of time the metric should have been set to 1
	time.Sleep(cfg.RetryInterval*time.Duration(cfg.StallTolerance) + 50*time.Millisecond) // #nosec G115
	require.True(t, stuckFSMIndicatingGauge.Snapshot().Value() == 1)
}

func TestMessageExtractor(t *testing.T) {
	ctx := context.Background()
	emptyblk0 := types.NewBlock(&types.Header{Number: common.Big1}, nil, nil, nil)
	emptyblk1 := types.NewBlock(&types.Header{Number: common.Big2, ParentHash: emptyblk0.Hash()}, nil, nil, nil)
	emptyblk2 := types.NewBlock(&types.Header{Number: common.Big3}, nil, nil, nil)
	parentChainReader := &mockParentChainReader{
		blocks: map[common.Hash]*types.Block{
			{}: {},
		},
		headers: map[common.Hash]*types.Header{
			{}: {},
		},
	}
	parentChainReader.blocks[emptyblk1.Hash()] = emptyblk1
	parentChainReader.blocks[emptyblk2.Hash()] = emptyblk2
	parentChainReader.blocks[common.BigToHash(common.Big1)] = emptyblk0
	parentChainReader.blocks[common.BigToHash(common.Big2)] = emptyblk1
	parentChainReader.blocks[common.BigToHash(common.Big3)] = emptyblk2
	consensusDB := rawdb.NewMemoryDatabase()
	melDB := NewDatabase(consensusDB)
	messageConsumer := &mockMessageConsumer{}
	extractor, err := NewMessageExtractor(
		DefaultMessageExtractionConfig,
		parentChainReader,
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, extractor.SetMessageConsumer(messageConsumer))
	extractor.StopWaiter.Start(ctx, extractor)
	require.NoError(t, err)
	require.True(t, extractor.CurrentFSMState() == Start)

	t.Run("Start", func(t *testing.T) {
		// Expect that an error in the initial state of the FSM
		// will cause the FSM to return to the start state.
		_, err = extractor.Act(ctx)
		require.ErrorContains(t, err, "error getting HeadMelStateBlockNum from database: not found")

		// Expect that we can now transition to the process
		// next block state.
		melState := &mel.State{
			Version:                42,
			ParentChainBlockNumber: 1,
			ParentChainBlockHash:   emptyblk0.Hash(),
		}
		require.NoError(t, melDB.SaveState(melState))

		parentChainReader.returnErr = errors.New("oops")
		_, err := extractor.Act(ctx)
		require.ErrorContains(t, err, "oops")

		require.True(t, extractor.CurrentFSMState() == Start)
		parentChainReader.returnErr = nil
		_, err = extractor.Act(ctx)
		require.NoError(t, err)

		require.True(t, extractor.CurrentFSMState() == ProcessingNextBlock)
		processBlockAction, ok := extractor.fsm.Current().SourceEvent.(processNextBlock)
		require.True(t, ok)
		require.Equal(t, processBlockAction.melState, melState)
	})
	t.Run("ProcessingNextBlock", func(t *testing.T) {
		parentChainReader.returnErr = errors.New("oops")
		_, err := extractor.Act(ctx)
		require.ErrorContains(t, err, "oops")
		require.True(t, extractor.CurrentFSMState() == ProcessingNextBlock)

		// If the parent chain reader returns block not found
		// as an error, the FSM will simply return without an error,
		// but will remain in the ProcessingNextBlock state.
		parentChainReader.returnErr = ethereum.NotFound
		_, err = extractor.Act(ctx)
		require.NoError(t, err)
		require.True(t, extractor.CurrentFSMState() == ProcessingNextBlock)

		// Correctly transitions to the saving messages state.
		parentChainReader.returnErr = nil
		_, err = extractor.Act(ctx)
		require.NoError(t, err)
		require.True(t, extractor.CurrentFSMState() == SavingMessages)
	})
	t.Run("SavingMessages", func(t *testing.T) {
		// Correctly transitions back to the ProcessingNextBlock state.
		_, err = extractor.Act(ctx)
		require.NoError(t, err)
		require.True(t, extractor.CurrentFSMState() == ProcessingNextBlock)
	})
	t.Run("Reorging", func(t *testing.T) {
		parentChainReader.blocks[common.BigToHash(big.NewInt(1))] = types.NewBlock(
			&types.Header{ParentHash: common.MaxHash}, nil, nil, nil,
		)
		headMelStateBlockNum, err := melDB.GetHeadMelStateBlockNum()
		require.NoError(t, err)
		require.True(t, headMelStateBlockNum == 2)

		// Correctly transitions to the Reorging messages state.
		parentChainReader.returnErr = nil
		_, err = extractor.Act(ctx)
		require.NoError(t, err)
		require.True(t, extractor.CurrentFSMState() == Reorging)

		// Reorging step should proceed to ProcessingNextBlock state
		_, err = extractor.Act(ctx)
		require.NoError(t, err)
		require.True(t, extractor.CurrentFSMState() == ProcessingNextBlock)
	})
}

type mockMessageConsumer struct{ returnErr error }

func (m *mockMessageConsumer) PushMessages(ctx context.Context, firstMsgIdx uint64, messages []*arbostypes.MessageWithMetadata) error {
	return m.returnErr
}

type mockParentChainReader struct {
	blocks    map[common.Hash]*types.Block
	headers   map[common.Hash]*types.Header
	logs      []*types.Log
	returnErr error
}

func (m *mockParentChainReader) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	blk, err := m.BlockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	return blk.Header(), err
}

func (m *mockParentChainReader) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	if number == nil || number.Int64() == rpc.FinalizedBlockNumber.Int64() {
		return types.NewBlock(&types.Header{Number: big.NewInt(1e10)}, nil, nil, nil), nil // Assume all parent chain blocks are finalized
	}
	block, ok := m.blocks[common.BigToHash(number)]
	if !ok {
		return nil, fmt.Errorf("block not found")
	}
	return block, nil
}

func (m *mockParentChainReader) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	block, ok := m.blocks[hash]
	if !ok {
		return nil, fmt.Errorf("block not found")
	}
	return block, nil
}

func (m *mockParentChainReader) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	header, ok := m.headers[hash]
	if !ok {
		return nil, fmt.Errorf("header not found")
	}
	return header, nil
}

func (m *mockParentChainReader) TransactionInBlock(ctx context.Context, blockHash common.Hash, index uint) (*types.Transaction, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	block, ok := m.blocks[blockHash]
	if !ok {
		return nil, fmt.Errorf("block not found")
	}
	if index >= uint(len(block.Transactions())) {
		return nil, fmt.Errorf("transaction index out of range")
	}
	txs := block.Transactions()
	return txs[index], nil

}
func (m *mockParentChainReader) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	// Mock implementation, return a dummy receipt
	return &types.Receipt{}, nil
}

func (m *mockParentChainReader) TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	return nil, false, nil
}

func (m *mockParentChainReader) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	filteredLogs := types.FilterLogs(m.logs, nil, nil, q.Addresses, q.Topics)
	var result []types.Log
	for _, log := range filteredLogs {
		result = append(result, *log)
	}
	return result, nil
}

func (m *mockParentChainReader) Client() rpc.ClientInterface { return nil }

func TestFinalizedDelayedMessageAtPosition(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consensusDB := rawdb.NewMemoryDatabase()
	melDB := NewDatabase(consensusDB)
	parentChainReader := &mockParentChainReader{
		blocks:  map[common.Hash]*types.Block{},
		headers: map[common.Hash]*types.Header{},
	}
	extractor, err := NewMessageExtractor(
		DefaultMessageExtractionConfig,
		parentChainReader,
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, extractor.SetMessageConsumer(&mockMessageConsumer{}))
	extractor.StopWaiter.Start(ctx, extractor)

	// Store delayed messages at positions 0..2 (count=3) at parent chain block 10.
	delayedMsgs := make([]*mel.DelayedInboxMessage, 3)
	state := &mel.State{
		ParentChainBlockNumber: 10,
		ParentChainBlockHash:   common.HexToHash("0xaa"),
	}
	var prevAcc common.Hash
	for i := range delayedMsgs {
		requestID := common.BigToHash(big.NewInt(int64(i)))
		delayedMsgs[i] = &mel.DelayedInboxMessage{
			BeforeInboxAcc:         prevAcc,
			ParentChainBlockNumber: 10,
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					Kind:      arbostypes.L1MessageType_EndOfBlock,
					RequestId: &requestID,
					L1BaseFee: common.Big0,
				},
			},
		}
		prevAcc = delayedMsgs[i].AfterInboxAcc()
		require.NoError(t, state.AccumulateDelayedMessage(delayedMsgs[i]))
		state.DelayedMessagesSeen++
	}
	require.NoError(t, melDB.SaveState(state))
	require.NoError(t, melDB.SaveDelayedMessages(state, delayedMsgs))

	t.Run("position below finalized count returns correct message and accumulator", func(t *testing.T) {
		// finalizedPos at block 10 is 3, requesting position 1 (< 3) should succeed
		msg, acc, parentChainBlock, err := extractor.FinalizedDelayedMessageAtPosition(ctx, 10, common.Hash{}, 1)
		require.NoError(t, err)
		require.NotNil(t, msg)
		expectedRequestID := common.BigToHash(big.NewInt(1))
		require.Equal(t, &expectedRequestID, msg.Header.RequestId, "should return message at requested position")
		require.Equal(t, delayedMsgs[1].AfterInboxAcc(), acc, "should return AfterInboxAcc of the message")
		require.Equal(t, uint64(10), parentChainBlock, "should return parent chain block number")
	})

	t.Run("last valid position returns correct message", func(t *testing.T) {
		// finalizedPos at block 10 is 3, requesting position 2 (== finalizedPos-1) is the
		// last valid position and must succeed. This is the exact boundary for the >= vs > fix.
		msg, acc, parentChainBlock, err := extractor.FinalizedDelayedMessageAtPosition(ctx, 10, common.Hash{}, 2)
		require.NoError(t, err)
		require.NotNil(t, msg)
		expectedRequestID := common.BigToHash(big.NewInt(2))
		require.Equal(t, &expectedRequestID, msg.Header.RequestId, "should return message at last valid position")
		require.Equal(t, delayedMsgs[2].AfterInboxAcc(), acc, "should return AfterInboxAcc of the message")
		require.Equal(t, uint64(10), parentChainBlock, "should return parent chain block number")
	})

	t.Run("correct lastDelayedAccumulator succeeds", func(t *testing.T) {
		// Pass the AfterInboxAcc of position 0 as lastDelayedAccumulator when requesting position 1.
		// This should match msg[1].BeforeInboxAcc and succeed.
		msg, acc, parentChainBlock, err := extractor.FinalizedDelayedMessageAtPosition(ctx, 10, delayedMsgs[0].AfterInboxAcc(), 1)
		require.NoError(t, err)
		require.NotNil(t, msg)
		require.Equal(t, delayedMsgs[1].AfterInboxAcc(), acc)
		require.Equal(t, uint64(10), parentChainBlock, "should return parent chain block number")
	})

	t.Run("wrong lastDelayedAccumulator returns error", func(t *testing.T) {
		bogusAcc := common.HexToHash("0xdeadbeef")
		_, _, _, err := extractor.FinalizedDelayedMessageAtPosition(ctx, 10, bogusAcc, 1)
		require.ErrorIs(t, err, mel.ErrDelayedAccumulatorMismatch)
	})

	t.Run("position equal to finalized count returns error", func(t *testing.T) {
		// finalizedPos at block 10 is 3, requesting position 3 (== 3) fails because
		// GetDelayedMessage is called first and the message doesn't exist in the DB.
		_, _, _, err := extractor.FinalizedDelayedMessageAtPosition(ctx, 10, common.Hash{}, 3)
		require.Error(t, err)
	})

	t.Run("position above finalized count returns error", func(t *testing.T) {
		// finalizedPos at block 10 is 3, requesting position 5 (> 3) fails because
		// GetDelayedMessage is called first and the message doesn't exist in the DB.
		_, _, _, err := extractor.FinalizedDelayedMessageAtPosition(ctx, 10, common.Hash{}, 5)
		require.Error(t, err)
	})

	t.Run("db not found returns not yet finalized", func(t *testing.T) {
		// Block 999 has no state in the DB, so GetDelayedCountAtParentChainBlock returns db not-found
		_, _, _, err := extractor.FinalizedDelayedMessageAtPosition(ctx, 999, common.Hash{}, 0)
		require.ErrorIs(t, err, mel.ErrDelayedMessageNotYetFinalized)
	})
}

// newTestExtractor creates a MessageExtractor backed by an in-memory DB for unit tests.
func newTestExtractor(t *testing.T) (*MessageExtractor, *Database) {
	t.Helper()
	consensusDB := rawdb.NewMemoryDatabase()
	melDB := NewDatabase(consensusDB)
	extractor, err := NewMessageExtractor(
		DefaultMessageExtractionConfig,
		&mockParentChainReader{
			blocks:  map[common.Hash]*types.Block{},
			headers: map[common.Hash]*types.Header{},
		},
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	return extractor, melDB
}

func TestFindInboxBatchContainingMessage(t *testing.T) {
	t.Parallel()

	// Setup: create an extractor with 5 batches.
	// Batch 0: messages [0, 3)  → MessageCount = 3
	// Batch 1: messages [3, 7)  → MessageCount = 7
	// Batch 2: messages [7, 10) → MessageCount = 10
	// Batch 3: messages [10, 10) (empty) → MessageCount = 10
	// Batch 4: messages [10, 15) → MessageCount = 15
	extractor, melDB := newTestExtractor(t)
	batchMessageCounts := []uint64{3, 7, 10, 10, 15}
	state := &mel.State{
		ParentChainBlockNumber: 100,
		ParentChainBlockHash:   common.HexToHash("0xaa"),
		BatchCount:             uint64(len(batchMessageCounts)),
	}
	require.NoError(t, melDB.SaveState(state))
	batchMetas := make([]*mel.BatchMetadata, len(batchMessageCounts))
	for i, mc := range batchMessageCounts {
		batchMetas[i] = &mel.BatchMetadata{MessageCount: arbutil.MessageIndex(mc)}
	}
	require.NoError(t, melDB.SaveBatchMetas(state, batchMetas))

	t.Run("zero batch count", func(t *testing.T) {
		emptyExtractor, emptyDB := newTestExtractor(t)
		emptyState := &mel.State{
			ParentChainBlockNumber: 1,
			ParentChainBlockHash:   common.HexToHash("0xbb"),
			BatchCount:             0,
		}
		require.NoError(t, emptyDB.SaveState(emptyState))
		batch, found, err := emptyExtractor.FindInboxBatchContainingMessage(0)
		require.NoError(t, err)
		require.False(t, found)
		require.Equal(t, uint64(0), batch)
	})

	t.Run("pos beyond last batch", func(t *testing.T) {
		batch, found, err := extractor.FindInboxBatchContainingMessage(15)
		require.NoError(t, err)
		require.False(t, found)
		require.Equal(t, uint64(0), batch)
	})

	t.Run("pos well beyond last batch", func(t *testing.T) {
		batch, found, err := extractor.FindInboxBatchContainingMessage(100)
		require.NoError(t, err)
		require.False(t, found)
		require.Equal(t, uint64(0), batch)
	})

	t.Run("pos in first batch", func(t *testing.T) {
		batch, found, err := extractor.FindInboxBatchContainingMessage(0)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(0), batch)
	})

	t.Run("pos at first batch boundary", func(t *testing.T) {
		// pos=2 is the last message in batch 0 (msgCount=3, messages 0-2)
		batch, found, err := extractor.FindInboxBatchContainingMessage(2)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(0), batch)
	})

	t.Run("pos exactly at batch boundary (count==pos)", func(t *testing.T) {
		// pos=3 → batch 0 has count=3, so count==pos → return batch 1
		batch, found, err := extractor.FindInboxBatchContainingMessage(3)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(1), batch)
	})

	t.Run("pos in middle batch", func(t *testing.T) {
		batch, found, err := extractor.FindInboxBatchContainingMessage(5)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(1), batch)
	})

	t.Run("pos at last message of last batch", func(t *testing.T) {
		batch, found, err := extractor.FindInboxBatchContainingMessage(14)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(4), batch)
	})

	t.Run("pos in batch after empty batch", func(t *testing.T) {
		// Batch 3 is empty (count=10, same as batch 2). The binary search finds
		// count(2)==pos and returns mid+1=3. This is the first batch after the
		// "full" batch, which is correct for non-empty batch sequences and an
		// acceptable edge case for empty batches (which don't occur in practice).
		batch, found, err := extractor.FindInboxBatchContainingMessage(10)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(3), batch)
	})

	t.Run("single batch containing pos", func(t *testing.T) {
		singleExtractor, singleDB := newTestExtractor(t)
		singleState := &mel.State{
			ParentChainBlockNumber: 1,
			ParentChainBlockHash:   common.HexToHash("0xcc"),
			BatchCount:             1,
		}
		require.NoError(t, singleDB.SaveState(singleState))
		require.NoError(t, singleDB.SaveBatchMetas(singleState, []*mel.BatchMetadata{
			{MessageCount: 5},
		}))
		batch, found, err := singleExtractor.FindInboxBatchContainingMessage(3)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(0), batch)
	})

	t.Run("single batch pos beyond", func(t *testing.T) {
		singleExtractor, singleDB := newTestExtractor(t)
		singleState := &mel.State{
			ParentChainBlockNumber: 1,
			ParentChainBlockHash:   common.HexToHash("0xdd"),
			BatchCount:             1,
		}
		require.NoError(t, singleDB.SaveState(singleState))
		require.NoError(t, singleDB.SaveBatchMetas(singleState, []*mel.BatchMetadata{
			{MessageCount: 5},
		}))
		batch, found, err := singleExtractor.FindInboxBatchContainingMessage(5)
		require.NoError(t, err)
		require.False(t, found)
		require.Equal(t, uint64(0), batch)
	})
}

func TestSaveMessagesErrorPaths(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	emptyblk0 := types.NewBlock(&types.Header{Number: common.Big1}, nil, nil, nil)
	emptyblk1 := types.NewBlock(&types.Header{Number: common.Big2, ParentHash: emptyblk0.Hash()}, nil, nil, nil)
	parentChainReader := &mockParentChainReader{
		blocks: map[common.Hash]*types.Block{
			{}:                            {},
			common.BigToHash(common.Big1): emptyblk0,
			common.BigToHash(common.Big2): emptyblk1,
			emptyblk1.Hash():              emptyblk1,
		},
		headers: map[common.Hash]*types.Header{
			{}: {},
		},
	}
	consensusDB := rawdb.NewMemoryDatabase()
	melDB := NewDatabase(consensusDB)
	consumer := &mockMessageConsumer{}
	extractor, err := NewMessageExtractor(
		DefaultMessageExtractionConfig,
		parentChainReader,
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, extractor.SetMessageConsumer(consumer))
	extractor.StopWaiter.Start(ctx, extractor)

	// Initialize state and advance FSM to SavingMessages
	melState := &mel.State{
		ParentChainBlockNumber: 1,
		ParentChainBlockHash:   emptyblk0.Hash(),
	}
	require.NoError(t, melDB.SaveState(melState))
	// Start → ProcessingNextBlock
	_, err = extractor.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, ProcessingNextBlock, extractor.CurrentFSMState())
	// ProcessingNextBlock → SavingMessages
	_, err = extractor.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, SavingMessages, extractor.CurrentFSMState())

	t.Run("PushMessages error keeps FSM in SavingMessages", func(t *testing.T) {
		consumer.returnErr = errors.New("push failed")
		_, err := extractor.Act(ctx)
		require.ErrorContains(t, err, "push failed")
		require.Equal(t, SavingMessages, extractor.CurrentFSMState())
	})

	t.Run("PushMessages retry succeeds after error clears", func(t *testing.T) {
		consumer.returnErr = nil
		_, err := extractor.Act(ctx)
		require.NoError(t, err)
		require.Equal(t, ProcessingNextBlock, extractor.CurrentFSMState())
	})
}

func TestSetMessageConsumerGuards(t *testing.T) {
	t.Parallel()
	extractor, _ := newTestExtractor(t)

	t.Run("set consumer succeeds initially", func(t *testing.T) {
		err := extractor.SetMessageConsumer(&mockMessageConsumer{})
		require.NoError(t, err)
	})

	t.Run("set consumer twice returns error", func(t *testing.T) {
		err := extractor.SetMessageConsumer(&mockMessageConsumer{})
		require.ErrorContains(t, err, "already set")
	})

	t.Run("set consumer after start returns error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		fresh, _ := newTestExtractor(t)
		require.NoError(t, fresh.SetMessageConsumer(&mockMessageConsumer{}))
		require.NoError(t, fresh.Start(ctx))
		err := fresh.SetMessageConsumer(&mockMessageConsumer{})
		require.ErrorContains(t, err, "cannot set message consumer after start")
	})
}

func TestMessageExtractionConfigValidate(t *testing.T) {
	t.Parallel()

	t.Run("valid latest", func(t *testing.T) {
		cfg := DefaultMessageExtractionConfig
		cfg.ReadMode = "latest"
		require.NoError(t, cfg.Validate())
	})

	t.Run("valid safe", func(t *testing.T) {
		cfg := DefaultMessageExtractionConfig
		cfg.ReadMode = "safe"
		require.NoError(t, cfg.Validate())
	})

	t.Run("valid finalized", func(t *testing.T) {
		cfg := DefaultMessageExtractionConfig
		cfg.ReadMode = "finalized"
		require.NoError(t, cfg.Validate())
	})

	t.Run("case normalization", func(t *testing.T) {
		cfg := DefaultMessageExtractionConfig
		cfg.ReadMode = "LATEST"
		require.NoError(t, cfg.Validate())
		require.Equal(t, "latest", cfg.ReadMode)
	})

	t.Run("invalid read mode", func(t *testing.T) {
		cfg := DefaultMessageExtractionConfig
		cfg.ReadMode = "unsafe"
		require.ErrorContains(t, cfg.Validate(), "invalid")
	})

	t.Run("zero log frequency", func(t *testing.T) {
		cfg := DefaultMessageExtractionConfig
		cfg.LogExtractionStatusFrequencyBlocks = 0
		require.ErrorContains(t, cfg.Validate(), "must be greater than 0")
	})
}
