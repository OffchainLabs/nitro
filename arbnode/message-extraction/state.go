package mel

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

// State defines the main struct describing the results of processing a single parent
// chain block at the message extraction layer. It is a versioned consensus type that can
// be deterministically constructed from any start state and parent chain blocks from
// that point onwards.
type State struct {
	Version                      uint16
	ParentChainId                uint64
	ParentChainBlockNumber       uint64
	ParentChainBlockHash         common.Hash
	ParentChainPreviousBlockHash common.Hash
	BatchPostingTargetAddress    common.Address
	MessageAccumulator           common.Hash
}

type StateDatabase interface {
	SaveState(
		ctx context.Context,
		state *State,
		messages []*arbostypes.MessageWithMetadata,
	) error
}

type StateFetcher interface {
	GetState(
		ctx context.Context, parentChainBlockHash common.Hash,
	) (*State, error)
}

func (s *State) Clone() *State {
	return s // TODO: Implement smart cloning of the state.
}

func (s *State) AccumulateMessage(message *arbostypes.MessageWithMetadata) {}
