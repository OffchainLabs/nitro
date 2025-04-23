package mel

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
)

var (
	// ErrInvalidParentChainBlock is returned when the parent chain block
	// hash does not match the expected hash in the state.
	ErrInvalidParentChainBlock = errors.New("invalid parent chain block")
)

func (m *MessageExtractionLayer) extractMessages(
	ctx context.Context,
	inputState *State,
	parentChainBlock *types.Block,
) (*State, []*arbostypes.MessageWithMetadata, error) {
	state := inputState.Clone()
	// Copies the state to avoid mutating the input in case of errors.
	// Check parent chain block hash linkage.
	if state.ParentChainPreviousBlockHash != parentChainBlock.ParentHash() {
		return nil, nil, fmt.Errorf(
			"%w: expected %s, got %s",
			ErrInvalidParentChainBlock,
			state.ParentChainPreviousBlockHash.Hex(),
			parentChainBlock.ParentHash().Hex(),
		)
	}
	// Now, check for any logs emitted by the sequencer inbox by txs
	// included in the parent chain block.
	prevBlockNum := parentChainBlock.NumberU64() - 1
	batches, err := m.sequencerInbox.LookupBatchesInRange(
		ctx,
		new(big.Int).SetUint64(prevBlockNum),
		parentChainBlock.Number(),
	)
	if err != nil {
		return nil, nil, err
	}
	var messages []*arbostypes.MessageWithMetadata
	for _, batch := range batches {
		serialized, err := batch.Serialize(ctx, m.l1Reader.Client())
		if err != nil {
			return nil, nil, err
		}
		rawSequencerMsg, err := arbstate.ParseSequencerMessage(
			ctx,
			batch.SequenceNumber,
			batch.BlockHash,
			serialized,
			m.dataProviders,
			daprovider.KeysetValidate,
		)
		if err != nil {
			return nil, nil, err
		}
		_ = rawSequencerMsg
		// TODO: Implement get next message.
		msg := &arbostypes.MessageWithMetadata{}
		messages = append(messages, msg)
		state.AccumulateMessage(msg)
	}

	// Updates the fields in the state to corresponding to the
	// incoming parent chain block.
	state.ParentChainBlockHash = parentChainBlock.Hash()
	state.ParentChainBlockNumber = parentChainBlock.NumberU64()

	return state, messages, nil
}
