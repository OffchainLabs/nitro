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
	emptyblk := types.NewBlock(&types.Header{}, nil, nil, nil)
	parentChainReader := &mockParentChainReader{
		blocks: map[common.Hash]*types.Block{
			{}:                              {},
			common.BigToHash(big.NewInt(1)): {},
		},
		headers: map[common.Hash]*types.Header{
			{}: {},
		},
	}
	parentChainReader.blocks[common.BigToHash(big.NewInt(1))] = emptyblk
	initialStateFetcher := &mockInitialStateFetcher{}
	mockDB := &mockMELDB{
		states: make(map[common.Hash]*meltypes.State),
	}
	extractor, err := NewMessageExtractor(
		parentChainReader,
		&chaininfo.RollupAddresses{},
		initialStateFetcher,
		mockDB,
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
			ParentChainBlockNumber: 0,
		}
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
		require.True(t, len(mockDB.states) == 1)
		mockDB.states[common.Hash{}] = &meltypes.State{}

		// Correctly transitions to the Reorging messages state.
		parentChainReader.returnErr = nil
		_, err = extractor.Act(ctx)
		require.NoError(t, err)
		require.True(t, extractor.CurrentFSMState() == Reorging)

		// Reorging step should delete the state? and then proceed to ProcessingNextBlock state
		_, err = extractor.Act(ctx)
		require.NoError(t, err)
		require.True(t, len(mockDB.states) == 1)
		require.True(t, extractor.CurrentFSMState() == ProcessingNextBlock)
	})
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
	states map[common.Hash]*meltypes.State
}

func (m *mockMELDB) State(
	_ context.Context,
	parentChainBlockHash common.Hash,
) (*meltypes.State, error) {
	if state, ok := m.states[parentChainBlockHash]; ok {
		return state, nil
	}
	return nil, errors.New("doesn't exist")
}

func (m *mockMELDB) SaveState(
	_ context.Context,
	state *meltypes.State,
	_ []*arbostypes.MessageWithMetadata,
) error {
	m.states[state.ParentChainBlockHash] = state
	return nil
}

func (m *mockMELDB) DeleteState(
	ctx context.Context, parentChainBlockHash common.Hash,
) error {
	delete(m.states, parentChainBlockHash)
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
