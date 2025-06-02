package mel

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
)

var _ ParentChainReader = (*mockParentChainReader)(nil)
var _ meltypes.StateDatabase = (*mockMELDB)(nil)

func TestMessageExtractor(t *testing.T) {
	ctx := context.Background()
	emptyblk := types.NewBlock(&types.Header{Number: common.Big2}, nil, nil, nil)
	parentChainReader := &mockParentChainReader{
		blocks: map[common.Hash]*types.Block{
			{}: {},
		},
		headers: map[common.Hash]*types.Header{
			{}: {},
		},
	}
	parentChainReader.blocks[common.BigToHash(big.NewInt(2))] = emptyblk
	parentChainReader.blocks[common.BigToHash(big.NewInt(3))] = emptyblk
	initialStateFetcher := &mockInitialStateFetcher{}
	mockDB := &mockMELDB{
		states: make(map[uint64]*meltypes.State),
	}
	messageConsumer := &mockMessageConsumer{}
	extractor, err := NewMessageExtractor(
		parentChainReader,
		&chaininfo.RollupAddresses{},
		initialStateFetcher,
		mockDB,
		messageConsumer,
		[]daprovider.Reader{},
		common.Hash{},
		0,
	)
	require.NoError(t, err)
	require.True(t, extractor.CurrentFSMState() == Start)

	t.Run("Start", func(t *testing.T) {
		// Expect that an error in the initial state of the FSM
		// will cause the FSM to return to the start state.
		parentChainReader.returnErr = errors.New("oops")
		_, err := extractor.Act(ctx)
		require.ErrorContains(t, err, "oops")

		require.True(t, extractor.CurrentFSMState() == Start)
		parentChainReader.returnErr = nil

		initialStateFetcher.returnErr = errors.New("failed to get state")
		_, err = extractor.Act(ctx)
		require.ErrorContains(t, err, "failed to get state")

		// Expect that we can now transition to the process
		// next block state.
		melState := &meltypes.State{
			Version:                42,
			ParentChainBlockNumber: 1,
		}
		mockDB.states[1] = melState
		initialStateFetcher.returnErr = nil
		initialStateFetcher.state = melState
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
		require.True(t, mockDB.headMelStateBlockNum == 2)

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

type mockInitialStateFetcher struct {
	state     *meltypes.State
	returnErr error
}

func (m *mockInitialStateFetcher) GetState(
	_ context.Context, _ common.Hash,
) (*meltypes.State, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	return m.state, nil
}

type mockParentChainReader struct {
	blocks    map[common.Hash]*types.Block
	headers   map[common.Hash]*types.Header
	returnErr error
}

func (m *mockParentChainReader) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	block, ok := m.blocks[common.BigToHash(number)]
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

type mockMELDB struct {
	headMelStateBlockNum uint64
	states               map[uint64]*meltypes.State
}

func (m *mockMELDB) State(
	_ context.Context,
	parentChainBlockNumber uint64,
) (*meltypes.State, error) {
	if state, ok := m.states[parentChainBlockNumber]; ok {
		return state, nil
	}
	return nil, errors.New("doesn't exist")
}

func (m *mockMELDB) SaveState(
	_ context.Context,
	state *meltypes.State,
) error {
	m.states[state.ParentChainBlockNumber] = state
	m.headMelStateBlockNum = state.ParentChainBlockNumber
	return nil
}

func (m *mockMELDB) SaveDelayedMessages(
	_ context.Context,
	_ *meltypes.State,
	_ []*arbnode.DelayedInboxMessage,
) error {
	return nil
}
func (m *mockMELDB) ReadDelayedMessage(
	_ context.Context,
	_ *meltypes.State,
	_ uint64,
) (*arbnode.DelayedInboxMessage, error) {
	return nil, errors.New("unimplemented")
}
