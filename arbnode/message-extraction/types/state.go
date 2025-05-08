package meltypes

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

// State defines the main struct describing the results of processing a single parent
// chain block at the message extraction layer. It is a versioned consensus type that can
// be deterministically constructed from any start state and parent chain blocks from
// that point onwards.
type State struct {
	Version                            uint16
	ParentChainId                      uint64
	ParentChainBlockNumber             uint64
	BatchPostingTargetAddress          common.Address
	DelayedMessagePostingTargetAddress common.Address
	ParentChainBlockHash               common.Hash
	ParentChainPreviousBlockHash       common.Hash
	MessageAccumulator                 common.Hash
	DelayedMessageAccumulator          common.Hash
	MsgCount                           uint64
	DelayedMessagesRead                uint64
	DelayedMessagedSeen                uint64
}

type StateDatabase interface {
	SaveState(
		ctx context.Context,
		state *State,
		messages []*arbostypes.MessageWithMetadata,
	) error
	SaveDelayedMessages(
		ctx context.Context,
		state *State,
		delayedMessages []*arbnode.DelayedInboxMessage,
	) error
	ReadDelayedMessage(
		ctx context.Context,
		index uint64,
	) (*arbnode.DelayedInboxMessage, error)
}

type StateFetcher interface {
	GetState(
		ctx context.Context, parentChainBlockHash common.Hash,
	) (*State, error)
}

func (s *State) Clone() *State {
	return s // TODO: Implement smart cloning of the state.
}

func (s *State) AccumulateMessage(msgHash common.Hash) *State {
	return s
}

func (s *State) AccumulateDelayedMessage(msg *arbnode.DelayedInboxMessage) *State {
	return s
}
