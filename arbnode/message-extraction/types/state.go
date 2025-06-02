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
	DelayedMessagesSeenRoot            common.Hash
	MessageAccumulator                 common.Hash
	MsgCount                           uint64
	DelayedMessagesRead                uint64
	DelayedMessagedSeen                uint64
}

// Defines a basic interface for MEL, including saving states, messages,
// and delayed messages to a database.
type StateDatabase interface {
	State(
		ctx context.Context,
		parentChainBlockHash common.Hash,
	) (*State, error)
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
	DelayedMessageDatabase
}

type DelayedMessageDatabase interface {
	ReadDelayedMessage(
		ctx context.Context,
		state *State,
		index uint64,
	) (*arbnode.DelayedInboxMessage, error)
}

// Defines an interface for fetching a MEL state by parent chain block hash.
type StateFetcher interface {
	GetState(
		ctx context.Context, parentChainBlockHash common.Hash,
	) (*State, error)
}

// Performs a deep clone of the state struct to prevent any unintended
// mutations of pointers at runtime.
func (s *State) Clone() *State {
	batchPostingTarget := common.Address{}
	delayedMessageTarget := common.Address{}
	parentChainHash := common.Hash{}
	parentChainPrevHash := common.Hash{}
	delayedMsgAcc := common.Hash{}
	msgAcc := common.Hash{}
	copy(batchPostingTarget[:], s.BatchPostingTargetAddress[:])
	copy(delayedMessageTarget[:], s.DelayedMessagePostingTargetAddress[:])
	copy(parentChainHash[:], s.ParentChainBlockHash[:])
	copy(parentChainPrevHash[:], s.ParentChainPreviousBlockHash[:])
	copy(delayedMsgAcc[:], s.DelayedMessagesSeenRoot[:])
	copy(msgAcc[:], s.MessageAccumulator[:])
	return &State{
		Version:                            s.Version,
		ParentChainId:                      s.ParentChainId,
		ParentChainBlockNumber:             s.ParentChainBlockNumber,
		BatchPostingTargetAddress:          batchPostingTarget,
		DelayedMessagePostingTargetAddress: delayedMessageTarget,
		ParentChainBlockHash:               parentChainHash,
		ParentChainPreviousBlockHash:       parentChainPrevHash,
		DelayedMessagesSeenRoot:            delayedMsgAcc,
		MessageAccumulator:                 msgAcc,
		MsgCount:                           s.MsgCount,
		DelayedMessagesRead:                s.DelayedMessagesRead,
		DelayedMessagedSeen:                s.DelayedMessagedSeen,
	}
}

func (s *State) AccumulateDelayedMessage(msg *arbnode.DelayedInboxMessage) *State {
	// TODO: Unimplemented.
	return s
}

func (s *State) AccumulateMessage(msg *arbnode.DelayedInboxMessage) *State {
	// TODO: Unimplemented.
	return s
}
