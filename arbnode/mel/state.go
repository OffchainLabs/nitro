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
	Version                            uint16
	ParentChainId                      uint64
	ParentChainBlockNumber             uint64
	BatchPostingTargetAddress          common.Address
	DelayedMessagePostingTargetAddress common.Address
	ParentChainBlockHash               common.Hash
	ParentChainPreviousBlockHash       common.Hash
	MessageAccumulator                 common.Hash
	DelayedMessagesSeenRoot            common.Hash
	MsgCount                           uint64
	BatchCount                         uint64
	DelayedMessagesRead                uint64
	DelayedMessagesSeen                uint64
}

// DelayedMessageDatabase can read delayed messages by their global index.
type DelayedMessageDatabase interface {
	ReadDelayedMessage(
		ctx context.Context,
		state *State,
		index uint64,
	) (*DelayedInboxMessage, error)
}

// Defines a basic interface for MEL, including saving states, messages,
// and delayed messages to a database.
type StateDatabase interface {
	DelayedMessageDatabase
	State(
		ctx context.Context,
		parentChainBlockNumber uint64,
	) (*State, error)
	SaveState(
		ctx context.Context,
		state *State,
	) error
	SaveDelayedMessages(
		ctx context.Context,
		state *State,
		delayedMessages []*DelayedInboxMessage,
	) error
}

// MessageConsumer is an interface to be implemented by readers of MEL such as transaction streamer of the nitro node
type MessageConsumer interface {
	PushMessages(
		ctx context.Context,
		firstMsgIdx uint64,
		messages []*arbostypes.MessageWithMetadata,
	) error
}

// Performs a deep clone of the state struct to prevent any unintended
// mutations of pointers at runtime.
func (s *State) Clone() *State {
	batchPostingTarget := common.Address{}
	delayedMessageTarget := common.Address{}
	parentChainHash := common.Hash{}
	parentChainPrevHash := common.Hash{}
	msgAcc := common.Hash{}
	delayedMsgSeenRoot := common.Hash{}
	copy(batchPostingTarget[:], s.BatchPostingTargetAddress[:])
	copy(delayedMessageTarget[:], s.DelayedMessagePostingTargetAddress[:])
	copy(parentChainHash[:], s.ParentChainBlockHash[:])
	copy(parentChainPrevHash[:], s.ParentChainPreviousBlockHash[:])
	copy(msgAcc[:], s.MessageAccumulator[:])
	copy(delayedMsgSeenRoot[:], s.DelayedMessagesSeenRoot[:])
	return &State{
		Version:                            s.Version,
		ParentChainId:                      s.ParentChainId,
		ParentChainBlockNumber:             s.ParentChainBlockNumber,
		BatchPostingTargetAddress:          batchPostingTarget,
		DelayedMessagePostingTargetAddress: delayedMessageTarget,
		ParentChainBlockHash:               parentChainHash,
		ParentChainPreviousBlockHash:       parentChainPrevHash,
		MessageAccumulator:                 msgAcc,
		DelayedMessagesSeenRoot:            delayedMsgSeenRoot,
		MsgCount:                           s.MsgCount,
		DelayedMessagesRead:                s.DelayedMessagesRead,
		DelayedMessagesSeen:                s.DelayedMessagesSeen,
	}
}

func (s *State) AccumulateMessage(msg *arbostypes.MessageWithMetadata) error {
	// TODO: Unimplemented.
	return nil
}

func (s *State) AccumulateDelayedMessage(msg *DelayedInboxMessage) error {
	// TODO: Unimplemented.
	return nil
}
