package melrunner

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

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
	arbDb := rawdb.NewMemoryDatabase()
	melDb := NewDatabase(arbDb)
	messageConsumer := &mockMessageConsumer{}
	extractor, err := NewMessageExtractor(
		parentChainReader,
		&chaininfo.RollupAddresses{},
		melDb,
		messageConsumer,
		[]daprovider.Reader{},
		0,
	)
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
		require.NoError(t, melDb.SaveState(ctx, melState))

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
		headMelStateBlockNum, err := melDb.GetHeadMelStateBlockNum()
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
	return nil, nil
}
