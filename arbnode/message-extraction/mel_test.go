package mel

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbnode"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/stretchr/testify/require"
)

var _ ParentChainReader = (*mockParentChainReader)(nil)
var _ meltypes.StateDatabase = (*mockMELDB)(nil)

func TestMessageExtractor(t *testing.T) {
	ctx := context.Background()
	parentChainReader := &mockParentChainReader{
		blocks: map[common.Hash]*types.Block{
			{}:                              {},
			common.BigToHash(big.NewInt(1)): {},
		},
		headers: map[common.Hash]*types.Header{
			{}: {},
		},
	}
	initialStateFetcher := &mockInitialStateFetcher{}
	extractor, err := NewMessageExtractor(
		parentChainReader,
		&chaininfo.RollupAddresses{},
		initialStateFetcher,
		&mockMELDB{},
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
}

func (m *mockMELDB) State(
	_ context.Context,
	_ common.Hash,
) (*meltypes.State, error) {
	return nil, errors.New("unimplemented")
}

func (m *mockMELDB) SaveState(
	_ context.Context,
	_ *meltypes.State,
	_ []*arbostypes.MessageWithMetadata,
) error {
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
