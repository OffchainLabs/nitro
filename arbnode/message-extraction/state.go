package mel

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

type StateDatabase interface {
	SaveState(ctx context.Context, state *State) error
}

type StateFetcher interface {
	GetState(
		ctx context.Context, parentChainBlockHash common.Hash,
	) (*State, error)
}

type State struct {
	Version                      uint16
	ParentChainId                uint64
	ParentChainBlockNumber       uint64
	ParentChainBlockHash         common.Hash
	ParentChainPreviousBlockHash common.Hash
	BatchPostingTargetAddress    common.Address
	MessageAccumulator           common.Hash
}

func (s *State) Clone() *State {
	return s // TODO: Implement smart cloning of the state.
}

func (s *State) AccumulateMessage(message *arbostypes.MessageWithMetadata) {}
