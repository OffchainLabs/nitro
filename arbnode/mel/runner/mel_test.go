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
		require.NoError(t, melDB.SaveState(ctx, melState))

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
		melState.SetDelayedMessageBacklog(processBlockAction.melState.GetDelayedMessageBacklog())
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
		return types.NewBlock(&types.Header{Number: big.NewInt(1e10)}, nil, nil, nil), nil // Assume all parent chain blocks are finalized to prevent issues dealing with delayed message backlog, it is tested separately
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
	state.SetDelayedMessageBacklog(&mel.DelayedMessageBacklog{})
	for i := range delayedMsgs {
		requestID := common.BigToHash(big.NewInt(int64(i)))
		delayedMsgs[i] = &mel.DelayedInboxMessage{
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					Kind:      arbostypes.L1MessageType_EndOfBlock,
					RequestId: &requestID,
					L1BaseFee: common.Big0,
				},
			},
		}
		require.NoError(t, state.AccumulateDelayedMessage(delayedMsgs[i]))
		state.DelayedMessagesSeen++
	}
	require.NoError(t, melDB.SaveState(ctx, state))
	require.NoError(t, melDB.SaveDelayedMessages(ctx, state, delayedMsgs))

	t.Run("position below finalized count returns correct message", func(t *testing.T) {
		// finalizedPos at block 10 is 3, requesting position 1 (< 3) should succeed
		msg, _, err := extractor.FinalizedDelayedMessageAtPosition(ctx, 10, common.Hash{}, 1)
		require.NoError(t, err)
		require.NotNil(t, msg)
		expectedRequestID := common.BigToHash(big.NewInt(1))
		require.Equal(t, &expectedRequestID, msg.Header.RequestId, "should return message at requested position")
	})

	t.Run("last valid position returns correct message", func(t *testing.T) {
		// finalizedPos at block 10 is 3, requesting position 2 (== finalizedPos-1) is the
		// last valid position and must succeed. This is the exact boundary for the >= vs > fix.
		msg, _, err := extractor.FinalizedDelayedMessageAtPosition(ctx, 10, common.Hash{}, 2)
		require.NoError(t, err)
		require.NotNil(t, msg)
		expectedRequestID := common.BigToHash(big.NewInt(2))
		require.Equal(t, &expectedRequestID, msg.Header.RequestId, "should return message at last valid position")
	})

	t.Run("position equal to finalized count returns not yet finalized", func(t *testing.T) {
		// finalizedPos at block 10 is 3, requesting position 3 (== 3) should return ErrDelayedMessageNotYetFinalized
		_, _, err := extractor.FinalizedDelayedMessageAtPosition(ctx, 10, common.Hash{}, 3)
		require.ErrorIs(t, err, mel.ErrDelayedMessageNotYetFinalized)
	})

	t.Run("position above finalized count returns not yet finalized", func(t *testing.T) {
		// finalizedPos at block 10 is 3, requesting position 5 (> 3) should return ErrDelayedMessageNotYetFinalized
		_, _, err := extractor.FinalizedDelayedMessageAtPosition(ctx, 10, common.Hash{}, 5)
		require.ErrorIs(t, err, mel.ErrDelayedMessageNotYetFinalized)
	})

	t.Run("db not found returns not yet finalized", func(t *testing.T) {
		// Block 999 has no state in the DB, so GetDelayedCountAtParentChainBlock returns db not-found
		_, _, err := extractor.FinalizedDelayedMessageAtPosition(ctx, 999, common.Hash{}, 0)
		require.ErrorIs(t, err, mel.ErrDelayedMessageNotYetFinalized)
	})
}
