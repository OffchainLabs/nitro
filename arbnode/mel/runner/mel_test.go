// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrunner

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
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
	melDB, err := NewDatabase(rawdb.NewMemoryDatabase())
	require.NoError(t, err)
	extractor, err := NewMessageExtractor(
		cfg,
		&mockParentChainReader{},
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil,
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, extractor.SetMessageConsumer(&mockMessageConsumer{}))
	require.True(t, stuckFSMIndicatingGauge.Snapshot().Value() == 0)
	require.NoError(t, extractor.Start(ctx))
	// MEL will be stuck at the 'Start' state as HeadMelState is not yet stored in the db.
	// Poll until the stall detector fires rather than sleeping a fixed duration.
	require.Eventually(t, func() bool {
		return stuckFSMIndicatingGauge.Snapshot().Value() == 1
	}, 5*time.Second, 10*time.Millisecond)
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
	melDB, err := NewDatabase(consensusDB)
	require.NoError(t, err)
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

// recordingConsumer tracks PushMessages calls for test verification.
type recordingConsumer struct {
	calls     []pushCall
	returnErr error
}
type pushCall struct {
	firstMsgIdx uint64
	count       int
}

func (r *recordingConsumer) PushMessages(_ context.Context, firstMsgIdx uint64, messages []*arbostypes.MessageWithMetadata) error {
	r.calls = append(r.calls, pushCall{firstMsgIdx: firstMsgIdx, count: len(messages)})
	return r.returnErr
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
	melDB, err := NewDatabase(consensusDB)
	require.NoError(t, err)
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
		var accErr error
		prevAcc, accErr = delayedMsgs[i].AfterInboxAcc()
		require.NoError(t, accErr)
		require.NoError(t, state.AccumulateDelayedMessage(delayedMsgs[i]))
	}
	require.NoError(t, melDB.SaveState(state))
	require.NoError(t, melDB.saveDelayedMessages(state, delayedMsgs))

	t.Run("position below finalized count returns correct message and accumulator", func(t *testing.T) {
		// finalizedPos at block 10 is 3, requesting position 1 (< 3) should succeed
		msg, acc, parentChainBlock, err := extractor.FinalizedDelayedMessageAtPosition(ctx, 10, common.Hash{}, 1)
		require.NoError(t, err)
		require.NotNil(t, msg)
		expectedRequestID := common.BigToHash(big.NewInt(1))
		require.Equal(t, &expectedRequestID, msg.Header.RequestId, "should return message at requested position")
		expectedAcc1, accErr := delayedMsgs[1].AfterInboxAcc()
		require.NoError(t, accErr)
		require.Equal(t, expectedAcc1, acc, "should return AfterInboxAcc of the message")
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
		expectedAcc2, accErr := delayedMsgs[2].AfterInboxAcc()
		require.NoError(t, accErr)
		require.Equal(t, expectedAcc2, acc, "should return AfterInboxAcc of the message")
		require.Equal(t, uint64(10), parentChainBlock, "should return parent chain block number")
	})

	t.Run("correct lastDelayedAccumulator succeeds", func(t *testing.T) {
		// Pass the AfterInboxAcc of position 0 as lastDelayedAccumulator when requesting position 1.
		// This should match msg[1].BeforeInboxAcc and succeed.
		lastAcc0, accErr := delayedMsgs[0].AfterInboxAcc()
		require.NoError(t, accErr)
		msg, acc, parentChainBlock, err := extractor.FinalizedDelayedMessageAtPosition(ctx, 10, lastAcc0, 1)
		require.NoError(t, err)
		require.NotNil(t, msg)
		expectedAcc1, accErr := delayedMsgs[1].AfterInboxAcc()
		require.NoError(t, accErr)
		require.Equal(t, expectedAcc1, acc)
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

func TestFindInboxBatchContainingMessage(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consensusDB := rawdb.NewMemoryDatabase()
	melDB, err := NewDatabase(consensusDB)
	require.NoError(t, err)

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
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, extractor.SetMessageConsumer(&mockMessageConsumer{}))
	extractor.StopWaiter.Start(ctx, extractor)

	t.Run("zero batches returns not found", func(t *testing.T) {
		state := &mel.State{ParentChainBlockNumber: 1, BatchCount: 0}
		require.NoError(t, melDB.SaveState(state))
		_, found, err := extractor.FindInboxBatchContainingMessage(5)
		require.NoError(t, err)
		require.False(t, found)
	})

	// Set up 4 batches with increasing message counts:
	// batch 0: msgCount=5, batch 1: msgCount=10, batch 2: msgCount=15, batch 3: msgCount=20
	state := &mel.State{ParentChainBlockNumber: 1, BatchCount: 4}
	require.NoError(t, melDB.SaveState(state))
	batchMetas := []*mel.BatchMetadata{
		{MessageCount: 5, ParentChainBlock: 10},
		{MessageCount: 10, ParentChainBlock: 20},
		{MessageCount: 15, ParentChainBlock: 30},
		{MessageCount: 20, ParentChainBlock: 40},
	}
	require.NoError(t, melDB.saveBatchMetas(state, batchMetas))

	t.Run("message in first batch", func(t *testing.T) {
		batch, found, err := extractor.FindInboxBatchContainingMessage(0)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(0), batch)
	})

	t.Run("message at first batch boundary", func(t *testing.T) {
		// pos=4 is the last message in batch 0 (msgCount=5 means positions 0..4)
		batch, found, err := extractor.FindInboxBatchContainingMessage(4)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(0), batch)
	})

	t.Run("message at exact batch boundary", func(t *testing.T) {
		// pos=5 is the first message in batch 1 (batch 0 has msgCount=5)
		batch, found, err := extractor.FindInboxBatchContainingMessage(5)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(1), batch)
	})

	t.Run("message in middle batch", func(t *testing.T) {
		batch, found, err := extractor.FindInboxBatchContainingMessage(12)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(2), batch)
	})

	t.Run("message in last batch", func(t *testing.T) {
		batch, found, err := extractor.FindInboxBatchContainingMessage(19)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(3), batch)
	})

	t.Run("message beyond last batch returns not found", func(t *testing.T) {
		_, found, err := extractor.FindInboxBatchContainingMessage(20)
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("message far beyond last batch returns not found", func(t *testing.T) {
		_, found, err := extractor.FindInboxBatchContainingMessage(100)
		require.NoError(t, err)
		require.False(t, found)
	})

	// Test with a single batch
	t.Run("single batch contains message", func(t *testing.T) {
		singleDB := rawdb.NewMemoryDatabase()
		singleMelDB, err := NewDatabase(singleDB)
		require.NoError(t, err)
		singleState := &mel.State{ParentChainBlockNumber: 1, BatchCount: 1}
		require.NoError(t, singleMelDB.SaveState(singleState))
		require.NoError(t, singleMelDB.saveBatchMetas(singleState, []*mel.BatchMetadata{
			{MessageCount: 10, ParentChainBlock: 5},
		}))
		singleExtractor, err := NewMessageExtractor(
			DefaultMessageExtractionConfig, parentChainReader,
			chaininfo.ArbitrumDevTestChainConfig(), &chaininfo.RollupAddresses{},
			singleMelDB, daprovider.NewDAProviderRegistry(), nil, nil, nil, nil,
		)
		require.NoError(t, err)
		require.NoError(t, singleExtractor.SetMessageConsumer(&mockMessageConsumer{}))
		singleExtractor.StopWaiter.Start(ctx, singleExtractor)

		batch, found, err := singleExtractor.FindInboxBatchContainingMessage(0)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(0), batch)

		batch, found, err = singleExtractor.FindInboxBatchContainingMessage(9)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(0), batch)

		_, found, err = singleExtractor.FindInboxBatchContainingMessage(10)
		require.NoError(t, err)
		require.False(t, found)
	})
}

func TestClampToInitialBlock(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consensusDB := rawdb.NewMemoryDatabase()
	melDB, err := NewDatabase(consensusDB)
	require.NoError(t, err)

	// Set up a migration boundary at block 100
	initialState := &mel.State{
		ParentChainBlockNumber: 100,
		BatchCount:             5,
		DelayedMessagesSeen:    3,
	}
	require.NoError(t, melDB.SaveInitialMelState(initialState))

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
		nil, nil, nil, nil,
	)
	require.NoError(t, err)
	require.NoError(t, extractor.SetMessageConsumer(&mockMessageConsumer{}))
	extractor.StopWaiter.Start(ctx, extractor)

	// Block below boundary should be clamped to boundary
	require.Equal(t, uint64(100), extractor.clampToInitialBlock(50))

	// Block at boundary should remain unchanged
	require.Equal(t, uint64(100), extractor.clampToInitialBlock(100))

	// Block above boundary should remain unchanged
	require.Equal(t, uint64(200), extractor.clampToInitialBlock(200))
}

func TestReorgBelowMigrationBoundary(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consensusDB := rawdb.NewMemoryDatabase()
	melDB, err := NewDatabase(consensusDB)
	require.NoError(t, err)

	// Create blocks with proper hash linkage
	block10 := types.NewBlock(&types.Header{Number: big.NewInt(10)}, nil, nil, nil)
	block11 := types.NewBlock(&types.Header{Number: big.NewInt(11), ParentHash: block10.Hash()}, nil, nil, nil)

	// Set up a migration boundary at block 10
	initialState := &mel.State{
		ParentChainBlockNumber: 10,
		ParentChainBlockHash:   block10.Hash(),
		BatchCount:             2,
	}
	require.NoError(t, melDB.SaveInitialMelState(initialState))

	// Save state at block 11 as the head (so initialize can load it)
	state11 := &mel.State{
		ParentChainBlockNumber: 11,
		ParentChainBlockHash:   block11.Hash(),
		BatchCount:             2,
	}
	require.NoError(t, melDB.SaveState(state11))

	parentChainReader := &mockParentChainReader{
		blocks: map[common.Hash]*types.Block{
			common.BigToHash(big.NewInt(10)): block10,
			common.BigToHash(big.NewInt(11)): block11,
		},
		headers: map[common.Hash]*types.Header{},
	}

	extractor, err := NewMessageExtractor(
		DefaultMessageExtractionConfig,
		parentChainReader,
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil, nil, nil, nil,
	)
	require.NoError(t, err)
	require.NoError(t, extractor.SetMessageConsumer(&mockMessageConsumer{}))
	extractor.StopWaiter.Start(ctx, extractor)

	// Run initialize step (Start -> ProcessingNextBlock) to set up logsAndHeadersPreFetcher
	_, err = extractor.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, ProcessingNextBlock, extractor.CurrentFSMState())

	// Now drive to Reorging state with a block at the migration boundary.
	// ParentChainBlockNumber == 11, target = 10 (at boundary, should succeed).
	err = extractor.fsm.Do(reorgToOldBlock{
		melState: state11,
	})
	require.NoError(t, err)
	require.Equal(t, Reorging, extractor.CurrentFSMState())

	_, err = extractor.Act(ctx)
	// Should succeed: target block 10 is at the boundary (not below)
	require.NoError(t, err)

	// Now drive to Reorging with ParentChainBlockNumber == 10.
	// Target = 9, which is BELOW the migration boundary 10. Should fail.
	err = extractor.fsm.Do(reorgToOldBlock{
		melState: initialState,
	})
	require.NoError(t, err)
	require.Equal(t, Reorging, extractor.CurrentFSMState())

	_, err = extractor.Act(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "below the MEL migration boundary")
	require.Contains(t, err.Error(), "manual intervention required")
}

func TestFatalErrChanEscalation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := DefaultMessageExtractionConfig
	cfg.StallTolerance = 1
	cfg.RetryInterval = 10 * time.Millisecond

	melDB, err := NewDatabase(rawdb.NewMemoryDatabase())
	require.NoError(t, err)

	fatalErrChan := make(chan error, 1)
	extractor, err := NewMessageExtractor(
		cfg,
		&mockParentChainReader{},
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil,
		nil,
		nil,
		fatalErrChan,
	)
	require.NoError(t, err)
	require.NoError(t, extractor.SetMessageConsumer(&mockMessageConsumer{}))
	require.NoError(t, extractor.Start(ctx))

	// MEL will be stuck at Start (no head state in DB). After 2*StallTolerance+1
	// errors, a fatal error should be sent on the channel.
	select {
	case fatalErr := <-fatalErrChan:
		require.Error(t, fatalErr)
		require.Contains(t, fatalErr.Error(), "message extractor stuck")
	case <-time.After(cfg.RetryInterval*time.Duration(2*cfg.StallTolerance+2) + 200*time.Millisecond): // #nosec G115
		t.Fatal("expected fatal error on fatalErrChan, but timed out")
	}
}

type nilHeaderParentChainReader struct {
	mockParentChainReader
}

// HeaderByNumber always returns (nil, nil) to simulate a parent chain that
// has no finalized/safe block available.
func (m *nilHeaderParentChainReader) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return nil, nil
}

func TestUpdateLastBlockToRead_NilHeaderEscalatesToFatal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := DefaultMessageExtractionConfig
	cfg.StallTolerance = 1
	cfg.RetryInterval = 10 * time.Millisecond
	cfg.ReadMode = ReadModeFinalized

	melDB, err := NewDatabase(rawdb.NewMemoryDatabase())
	require.NoError(t, err)

	fatalErrChan := make(chan error, 1)
	extractor, err := NewMessageExtractor(
		cfg,
		&nilHeaderParentChainReader{mockParentChainReader{
			blocks:  map[common.Hash]*types.Block{},
			headers: map[common.Hash]*types.Header{},
		}},
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil,
		nil,
		nil,
		fatalErrChan,
	)
	require.NoError(t, err)
	require.NoError(t, extractor.SetMessageConsumer(&mockMessageConsumer{}))

	// Drive updateLastBlockToRead manually past the fatal threshold.
	for i := uint64(0); i <= 2*cfg.StallTolerance; i++ {
		extractor.updateLastBlockToRead(ctx)
	}

	select {
	case fatalErr := <-fatalErrChan:
		require.Error(t, fatalErr)
		require.Contains(t, fatalErr.Error(), "nil header")
	default:
		t.Fatal("expected fatal error on fatalErrChan after repeated nil headers")
	}
}

func TestGetDelayedCountAtParentChainBlock_RejectsAboveHead(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := rawdb.NewMemoryDatabase()
	melDB, err := NewDatabase(db)
	require.NoError(t, err)

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
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, extractor.SetMessageConsumer(&mockMessageConsumer{}))

	// Save states at blocks 5 and 10, head at 10
	require.NoError(t, melDB.SaveState(&mel.State{
		ParentChainBlockNumber: 5,
		DelayedMessagesSeen:    2,
	}))
	require.NoError(t, melDB.SaveState(&mel.State{
		ParentChainBlockNumber: 10,
		DelayedMessagesSeen:    5,
	}))

	// At head (10) should work
	count, err := extractor.GetDelayedCountAtParentChainBlock(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, uint64(5), count)

	// Below head (5) should work
	count, err = extractor.GetDelayedCountAtParentChainBlock(ctx, 5)
	require.NoError(t, err)
	require.Equal(t, uint64(2), count)

	// Above head (15) should fail — StateAtOrBelowHead rejects it
	_, err = extractor.GetDelayedCountAtParentChainBlock(ctx, 15)
	require.Error(t, err)
	require.Contains(t, err.Error(), "above current head")
}

func TestHandlePreimageCacheMissLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	melDB, err := NewDatabase(rawdb.NewMemoryDatabase())
	require.NoError(t, err)

	extractor, err := NewMessageExtractor(
		DefaultMessageExtractionConfig,
		&mockParentChainReader{},
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil, nil, nil, nil,
	)
	require.NoError(t, err)
	require.NoError(t, extractor.SetMessageConsumer(&mockMessageConsumer{}))
	extractor.StopWaiter.Start(ctx, extractor)

	// State with no unread delayed messages — RebuildDelayedMsgPreimages is a no-op.
	state := &mel.State{}

	// First two calls should succeed (return 0, nil for immediate retry).
	dur, err := extractor.handlePreimageCacheMiss(state)
	require.NoError(t, err)
	require.Zero(t, dur)
	require.Equal(t, 1, extractor.consecutivePreimageRebuilds)

	dur, err = extractor.handlePreimageCacheMiss(state)
	require.NoError(t, err)
	require.Zero(t, dur)
	require.Equal(t, 2, extractor.consecutivePreimageRebuilds)

	// Third call should return an error.
	dur, err = extractor.handlePreimageCacheMiss(state)
	require.Error(t, err)
	require.Contains(t, err.Error(), "repeated preimage rebuild")
	require.Equal(t, 3, extractor.consecutivePreimageRebuilds)
	require.Equal(t, DefaultMessageExtractionConfig.RetryInterval, dur)
}

func TestStallToleranceZeroDoesNotErrorOnFirstNotFound(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := DefaultMessageExtractionConfig
	cfg.StallTolerance = 0

	melDB, err := NewDatabase(rawdb.NewMemoryDatabase())
	require.NoError(t, err)

	headBlock := types.NewBlock(&types.Header{Number: common.Big1}, nil, nil, nil)
	parentChainReader := &mockParentChainReader{
		blocks: map[common.Hash]*types.Block{
			common.BigToHash(common.Big1): headBlock,
		},
		headers: map[common.Hash]*types.Header{},
	}
	extractor, err := NewMessageExtractor(
		cfg,
		parentChainReader,
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil, nil, nil, nil,
	)
	require.NoError(t, err)
	require.NoError(t, extractor.SetMessageConsumer(&mockMessageConsumer{}))
	extractor.StopWaiter.Start(ctx, extractor)

	// Set up state at block 1 so initialize succeeds.
	melState := &mel.State{
		ParentChainBlockNumber: 1,
		ParentChainBlockHash:   headBlock.Hash(),
	}
	require.NoError(t, melDB.SaveState(melState))

	// Initialize: Start -> ProcessingNextBlock
	_, err = extractor.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, ProcessingNextBlock, extractor.CurrentFSMState())

	// Block 2 not found — with StallTolerance=0, this should NOT return an error.
	parentChainReader.returnErr = ethereum.NotFound
	_, err = extractor.Act(ctx)
	require.NoError(t, err, "StallTolerance=0 should not error on first NotFound")
	require.Equal(t, ProcessingNextBlock, extractor.CurrentFSMState())
}

// toggleFailKVS wraps a KeyValueStore with a toggleable batch Write failure.
type toggleFailKVS struct {
	ethdb.KeyValueStore
	fail    atomic.Bool
	failErr error
}

func (t *toggleFailKVS) NewBatch() ethdb.Batch {
	return &toggleFailBatch{Batch: t.KeyValueStore.NewBatch(), parent: t}
}

func (t *toggleFailKVS) NewBatchWithSize(size int) ethdb.Batch {
	return &toggleFailBatch{Batch: t.KeyValueStore.NewBatchWithSize(size), parent: t}
}

type toggleFailBatch struct {
	ethdb.Batch
	parent *toggleFailKVS
}

func (b *toggleFailBatch) Write() error {
	if b.parent.fail.Load() {
		return b.parent.failErr
	}
	return b.Batch.Write()
}

func TestSaveMessages_RetryAfterDBFailure(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumer := &recordingConsumer{}
	wrapper := &toggleFailKVS{
		KeyValueStore: rawdb.NewMemoryDatabase(),
		failErr:       errors.New("disk full"),
	}
	melDB, err := NewDatabase(wrapper)
	require.NoError(t, err)

	initialState := &mel.State{
		ParentChainBlockNumber: 100,
		ParentChainBlockHash:   common.HexToHash("0xaaa"),
	}
	require.NoError(t, melDB.SaveInitialMelState(initialState))

	cfg := TestMessageExtractionConfig
	fsmInst, err := newFSM(Start)
	require.NoError(t, err)

	extractor := &MessageExtractor{
		config:       cfg,
		melDB:        melDB,
		msgConsumer:  consumer,
		fsm:          fsmInst,
		caughtUpChan: make(chan struct{}),
	}

	// Transition FSM: Start -> ProcessingNextBlock -> SavingMessages
	postState := initialState.Clone()
	postState.ParentChainBlockNumber = 101
	postState.ParentChainBlockHash = common.HexToHash("0xbbb")
	postState.ParentChainPreviousBlockHash = common.HexToHash("0xaaa")
	postState.MsgCount = 3

	require.NoError(t, fsmInst.Do(processNextBlock{melState: initialState}))
	require.NoError(t, fsmInst.Do(saveMessages{
		preStateMsgCount: 0,
		postState:        postState,
		messages:         []*arbostypes.MessageWithMetadata{{Message: &arbostypes.L1IncomingMessage{Header: &arbostypes.L1IncomingMessageHeader{}}}},
	}))
	require.Equal(t, SavingMessages, extractor.CurrentFSMState())

	// First Act: PushMessages succeeds but SaveProcessedBlock fails
	wrapper.fail.Store(true)
	_, err = extractor.Act(ctx)
	require.Error(t, err)
	require.ErrorContains(t, err, "disk full")
	require.Equal(t, SavingMessages, extractor.CurrentFSMState(), "FSM must stay in SavingMessages on DB failure")
	require.Len(t, consumer.calls, 1, "PushMessages should have been called once")
	require.Equal(t, uint64(0), consumer.calls[0].firstMsgIdx)
	require.Equal(t, 1, consumer.calls[0].count)

	// Second Act: DB succeeds, PushMessages is called again (re-push), FSM advances
	wrapper.fail.Store(false)
	_, err = extractor.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, ProcessingNextBlock, extractor.CurrentFSMState(), "FSM should advance to ProcessingNextBlock after retry")
	require.Len(t, consumer.calls, 2, "PushMessages should have been called twice (idempotent re-push)")
	require.Equal(t, consumer.calls[0], consumer.calls[1], "retry must push identical arguments")
}

func TestSetMessageConsumer_Guards(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	melDB, err := NewDatabase(rawdb.NewMemoryDatabase())
	require.NoError(t, err)
	extractor, err := NewMessageExtractor(
		DefaultMessageExtractionConfig,
		&mockParentChainReader{},
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil, nil, nil, nil,
	)
	require.NoError(t, err)

	// First set succeeds.
	require.NoError(t, extractor.SetMessageConsumer(&mockMessageConsumer{}))

	// Double-set returns error.
	err = extractor.SetMessageConsumer(&mockMessageConsumer{})
	require.ErrorContains(t, err, "already set")

	// After start, setting returns error.
	require.NoError(t, extractor.Start(ctx))
	extractor2, err := NewMessageExtractor(
		DefaultMessageExtractionConfig,
		&mockParentChainReader{},
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil, nil, nil, nil,
	)
	require.NoError(t, err)
	require.NoError(t, extractor2.SetMessageConsumer(&mockMessageConsumer{}))
	require.NoError(t, extractor2.Start(ctx))
	err = extractor2.SetMessageConsumer(&mockMessageConsumer{})
	require.ErrorContains(t, err, "after start")
}

func TestSetBlockValidator_Guards(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	melDB, err := NewDatabase(rawdb.NewMemoryDatabase())
	require.NoError(t, err)
	extractor, err := NewMessageExtractor(
		DefaultMessageExtractionConfig,
		&mockParentChainReader{},
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil, nil, nil, nil,
	)
	require.NoError(t, err)
	require.NoError(t, extractor.SetMessageConsumer(&mockMessageConsumer{}))

	// First set succeeds (passing nil is fine for this guard test).
	require.NoError(t, extractor.SetBlockValidator(nil))

	// Double-set returns error (even with nil, the field is checked for non-nil pointer).
	// Note: SetBlockValidator checks `m.blockValidator != nil` which won't trigger for nil.
	// So we need to set a non-nil value first to test the double-set guard.
	extractor2, err := NewMessageExtractor(
		DefaultMessageExtractionConfig,
		&mockParentChainReader{},
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil, nil, nil, nil,
	)
	require.NoError(t, err)
	require.NoError(t, extractor2.SetMessageConsumer(&mockMessageConsumer{}))
	// We can't easily create a real BlockValidator, but we can verify the after-start guard.
	require.NoError(t, extractor2.Start(ctx))
	err = extractor2.SetBlockValidator(nil)
	require.ErrorContains(t, err, "after start")
}

func TestEscalateIfPersistent(t *testing.T) {
	t.Parallel()

	t.Run("nil fatalErrChan is a no-op", func(t *testing.T) {
		t.Parallel()
		extractor := &MessageExtractor{
			config:       MessageExtractionConfig{StallTolerance: 5},
			fatalErrChan: nil,
		}
		// Should not panic or block.
		ctx := context.Background()
		extractor.escalateIfPersistent(ctx, 100, errors.New("test"))
	})

	t.Run("StallTolerance zero disables escalation", func(t *testing.T) {
		t.Parallel()
		fatalChan := make(chan error, 1)
		extractor := &MessageExtractor{
			config:       MessageExtractionConfig{StallTolerance: 0},
			fatalErrChan: fatalChan,
		}
		ctx := context.Background()
		extractor.escalateIfPersistent(ctx, 100, errors.New("test"))
		select {
		case <-fatalChan:
			t.Fatal("should not have sent to fatalErrChan when StallTolerance is 0")
		default:
		}
	})

	t.Run("below threshold does not escalate", func(t *testing.T) {
		t.Parallel()
		fatalChan := make(chan error, 1)
		extractor := &MessageExtractor{
			config:       MessageExtractionConfig{StallTolerance: 5},
			fatalErrChan: fatalChan,
		}
		ctx := context.Background()
		// 2*5 = 10; failures=10 is NOT > 10, so no escalation.
		extractor.escalateIfPersistent(ctx, 10, errors.New("test"))
		select {
		case <-fatalChan:
			t.Fatal("should not escalate at exactly 2x threshold")
		default:
		}
	})

	t.Run("above threshold escalates", func(t *testing.T) {
		t.Parallel()
		fatalChan := make(chan error, 1)
		extractor := &MessageExtractor{
			config:       MessageExtractionConfig{StallTolerance: 5},
			fatalErrChan: fatalChan,
		}
		ctx := context.Background()
		testErr := errors.New("persistent failure")
		extractor.escalateIfPersistent(ctx, 11, testErr)
		select {
		case err := <-fatalChan:
			require.Equal(t, testErr, err)
		default:
			t.Fatal("should have sent to fatalErrChan when failures > 2*StallTolerance")
		}
	})

	t.Run("context cancellation prevents blocking", func(t *testing.T) {
		t.Parallel()
		// Unbuffered channel that nobody reads — would block forever without ctx.
		fatalChan := make(chan error)
		extractor := &MessageExtractor{
			config:       MessageExtractionConfig{StallTolerance: 1},
			fatalErrChan: fatalChan,
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately
		// Should return without blocking.
		extractor.escalateIfPersistent(ctx, 100, errors.New("test"))
	})
}

func TestGetDelayedMessage_OutOfBounds(t *testing.T) {
	t.Parallel()

	melDB, err := NewDatabase(rawdb.NewMemoryDatabase())
	require.NoError(t, err)
	extractor, err := NewMessageExtractor(
		DefaultMessageExtractionConfig,
		&mockParentChainReader{},
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil, nil, nil, nil,
	)
	require.NoError(t, err)

	// State has 3 delayed messages seen.
	state := &mel.State{
		ParentChainBlockNumber: 1,
		DelayedMessagesSeen:    3,
	}
	require.NoError(t, melDB.SaveState(state))

	// Requesting at the boundary (index == seen) should fail.
	_, err = extractor.GetDelayedMessage(3)
	require.ErrorIs(t, err, mel.ErrAccumulatorNotFound)

	// Requesting above the boundary should fail.
	_, err = extractor.GetDelayedMessage(100)
	require.ErrorIs(t, err, mel.ErrAccumulatorNotFound)
}

func TestGetBatchMetadata_OutOfBounds(t *testing.T) {
	t.Parallel()

	melDB, err := NewDatabase(rawdb.NewMemoryDatabase())
	require.NoError(t, err)
	extractor, err := NewMessageExtractor(
		DefaultMessageExtractionConfig,
		&mockParentChainReader{},
		chaininfo.ArbitrumDevTestChainConfig(),
		&chaininfo.RollupAddresses{},
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil, nil, nil, nil,
	)
	require.NoError(t, err)

	// State has 2 batches.
	state := &mel.State{
		ParentChainBlockNumber: 1,
		BatchCount:             2,
	}
	require.NoError(t, melDB.SaveState(state))

	// Requesting at the boundary (seqNum == count) should fail.
	_, err = extractor.GetBatchMetadata(2)
	require.ErrorContains(t, err, "batchMetadata not available")

	// Requesting above the boundary should fail.
	_, err = extractor.GetBatchMetadata(100)
	require.ErrorContains(t, err, "batchMetadata not available")
}
