package mel

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
)

var (
	// ErrInvalidParentChainBlock is returned when the parent chain block
	// hash does not match the expected hash in the state.
	ErrInvalidParentChainBlock = errors.New("invalid parent chain block")
)

func (m *MessageExtractionLayer) extractMessages(
	ctx context.Context,
	state *State,
	parentChainBlock *types.Block,
) (*State, error) {
	// Check parent chain block hash linkage.
	if state.ParentChainPreviousBlockHash != parentChainBlock.ParentHash() {
		return nil, fmt.Errorf(
			"%w: expected %s, got %s",
			ErrInvalidParentChainBlock,
			state.ParentChainPreviousBlockHash.Hex(),
			parentChainBlock.ParentHash().Hex(),
		)
	}
	// First, updates the fields in the state to corresponding to the
	// newly processed parent chain block.
	state.ParentChainBlockHash = parentChainBlock.Hash()
	state.ParentChainBlockNumber = parentChainBlock.NumberU64()
	return state, nil
}
