package mel

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
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
}
