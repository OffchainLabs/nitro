package meltypes

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
)

// DelayedMsgInfoQueueItem contains metadata relating to delayed messages
type DelayedMsgInfoQueueItem struct {
	Index                       uint64
	Read                        bool
	MerkleRoot                  common.Hash
	MelStateParentChainBlockNum uint64
}

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
	MsgCount                           uint64
	DelayedMessagesRead                uint64
	DelayedMessagedSeen                uint64
	DelayedMessageMerklePartials       []common.Hash `rlp:"optional"`

	// seenDelayedMsgInfoQueue represents the queue containing DelayedMsgInfoQueueItems that hold metadata relating to delayed messages that have been seen but not yet read
	// queue is trimmed from left by pruner function defined on the state, after corresponding delayed message is read and its melStateParentChainBlockNum is finalized
	// trimmed from right in case of a reorg by the Reorging fsm step one melstate at a time
	seenDelayedMsgInfoQueue []*DelayedMsgInfoQueueItem

	// seen and read DelayedMsgsAcc are MerkleAccumulators that reset after the current melstate is finished generating, to prevent stale validations
	seenDelayedMsgsAcc *merkleAccumulator.MerkleAccumulator
	readDelayedMsgsAcc *merkleAccumulator.MerkleAccumulator
}

// Defines a basic interface for MEL, including saving states, messages,
// and delayed messages to a database.
type StateDatabase interface {
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
		delayedMessages []*arbnode.DelayedInboxMessage,
	) error
	ReadDelayedMessage(
		ctx context.Context,
		state *State,
		index uint64,
	) (*arbnode.DelayedInboxMessage, error)
}

// MessageConsumer is an interface to be implemented by readers of MEL such as transaction streamer of the nitro node
type MessageConsumer interface {
	PushMessages(
		ctx context.Context,
		firstMsgIdx uint64,
		messages []*arbostypes.MessageWithMetadata,
	) error
}

// Defines an interface for fetching a MEL state by parent chain block hash.
//
// If the initial implementation is melDB then the melState's seenDelayedMsgInfoQueue will be
// initialized automatically but for non-melDB implementations:
//   - either DelayedMessagesSeen must equal DelayedMessagesRead
//     (OR)
//   - seenDelayedMsgInfoQueue must be manually initialized using SetSeenDelayedMsgInfoQueue
type StateFetcher interface {
	// GetState should initialize seenDelayedMsgInfoQueue in case the initial state's DelayedMessagedSeen is ahead of DelayedMessagedRead
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
	msgAcc := common.Hash{}
	copy(batchPostingTarget[:], s.BatchPostingTargetAddress[:])
	copy(delayedMessageTarget[:], s.DelayedMessagePostingTargetAddress[:])
	copy(parentChainHash[:], s.ParentChainBlockHash[:])
	copy(parentChainPrevHash[:], s.ParentChainPreviousBlockHash[:])
	copy(msgAcc[:], s.MessageAccumulator[:])
	var delayedMessageMerklePartials []common.Hash
	for _, partial := range s.DelayedMessageMerklePartials {
		clone := common.Hash{}
		copy(clone[:], partial[:])
		delayedMessageMerklePartials = append(delayedMessageMerklePartials, clone)
	}
	var seenDelayedMsgInfoQueue []*DelayedMsgInfoQueueItem
	for _, item := range s.seenDelayedMsgInfoQueue {
		merkleRoot := common.Hash{}
		copy(merkleRoot[:], item.MerkleRoot[:])
		seenDelayedMsgInfoQueue = append(seenDelayedMsgInfoQueue, &DelayedMsgInfoQueueItem{
			Index:                       item.Index,
			Read:                        item.Read,
			MerkleRoot:                  merkleRoot,
			MelStateParentChainBlockNum: item.MelStateParentChainBlockNum,
		})
	}
	return &State{
		Version:                            s.Version,
		ParentChainId:                      s.ParentChainId,
		ParentChainBlockNumber:             s.ParentChainBlockNumber,
		BatchPostingTargetAddress:          batchPostingTarget,
		DelayedMessagePostingTargetAddress: delayedMessageTarget,
		ParentChainBlockHash:               parentChainHash,
		ParentChainPreviousBlockHash:       parentChainPrevHash,
		MessageAccumulator:                 msgAcc,
		MsgCount:                           s.MsgCount,
		DelayedMessagesRead:                s.DelayedMessagesRead,
		DelayedMessagedSeen:                s.DelayedMessagedSeen,
		DelayedMessageMerklePartials:       delayedMessageMerklePartials,
		seenDelayedMsgInfoQueue:            seenDelayedMsgInfoQueue,
	}
}

func (s *State) AccumulateMessage(msg *arbostypes.MessageWithMetadata) *State {
	// TODO: Unimplemented.
	return s
}

func (s *State) AccumulateDelayedMessage(msg *arbnode.DelayedInboxMessage) error {
	if s.seenDelayedMsgsAcc == nil {
		log.Debug("Initializing MelState's seenDelayedMsgsAcc")
		// This is very low cost hence better to reconstruct seenDelayedMsgsAcc from fresh partals instead of risking using a dirty acc
		acc, err := merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(ToPtrSlice(s.DelayedMessageMerklePartials))
		if err != nil {
			return err
		}
		s.seenDelayedMsgsAcc = acc
	}
	if _, err := s.seenDelayedMsgsAcc.Append(msg.Hash()); err != nil {
		return err
	}
	merkleRoot, err := s.seenDelayedMsgsAcc.Root()
	if err != nil {
		return err
	}
	s.seenDelayedMsgInfoQueue = append(s.seenDelayedMsgInfoQueue, &DelayedMsgInfoQueueItem{
		Index:                       s.DelayedMessagedSeen,
		MerkleRoot:                  merkleRoot,
		MelStateParentChainBlockNum: s.ParentChainBlockNumber,
	})
	return nil
}

func (s *State) GenerateDelayedMessageMerklePartials() error {
	partialsPtrs, err := s.seenDelayedMsgsAcc.GetPartials()
	if err != nil {
		return err
	}
	s.DelayedMessageMerklePartials = FromPtrSlice(partialsPtrs)
	return nil
}

func (s *State) GetReadDelayedMsgsAcc() *merkleAccumulator.MerkleAccumulator {
	return s.readDelayedMsgsAcc
}

func (s *State) SetReadDelayedMsgsAcc(acc *merkleAccumulator.MerkleAccumulator) {
	s.readDelayedMsgsAcc = acc
}

func (s *State) GetSeenDelayedMsgInfoQueue() []*DelayedMsgInfoQueueItem {
	return s.seenDelayedMsgInfoQueue
}

func (s *State) SetSeenDelayedMsgInfoQueue(seenDelayedMsgInfoQueue []*DelayedMsgInfoQueueItem) {
	s.seenDelayedMsgInfoQueue = seenDelayedMsgInfoQueue
}

// TrimSeenDelayedMsgInfoQueue trims the seenDelayedMsgInfoQueue from left, such that the item is only removed if the corresponding delayed message is
// read and the MelStateParentChainBlockNum is finalized- this is to make seenDelayedMsgInfoQueue as reorg resistant as possible
func (s *State) TrimSeenDelayedMsgInfoQueue(finalizedBlock uint64) {
	i := 0
	for i < len(s.seenDelayedMsgInfoQueue) {
		if !s.seenDelayedMsgInfoQueue[i].Read || s.seenDelayedMsgInfoQueue[i].MelStateParentChainBlockNum > finalizedBlock {
			break
		}
		i++
	}
	s.seenDelayedMsgInfoQueue = s.seenDelayedMsgInfoQueue[i:]
}

func ToPtrSlice[T any](list []T) []*T {
	var ptrs []*T
	for _, item := range list {
		ptrs = append(ptrs, &item)
	}
	return ptrs
}

func FromPtrSlice[T any](ptrs []*T) []T {
	list := make([]T, len(ptrs))
	for i, ptr := range ptrs {
		if ptr != nil {
			list[i] = *ptr
		}
	}
	return list
}
