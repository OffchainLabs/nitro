package mel

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
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
	DelayedMessageMerklePartials       []common.Hash `rlp:"optional"`

	delayedMessageBacklog *DelayedMessageBacklog // delayedMessageBacklog is initialized once in the Start fsm step of mel runner and is persisted across all future states
	readCountFromBacklog  uint64                 // delayed messages with index lower than this count have been pre-read and we have hashes for them in-memory for verification

	// seenDelayedMsgsAcc is MerkleAccumulator that accumulates delayed messages seen
	// from parent chain. It resets after the current melstate is finished generating
	// and is reinitialized using appropriate DelayedMessageMerklePartials of the state
	seenDelayedMsgsAcc *merkleAccumulator.MerkleAccumulator
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

func (s *State) Hash() common.Hash {
	return common.Hash{}
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
	var delayedMessageMerklePartials []common.Hash
	for _, partial := range s.DelayedMessageMerklePartials {
		clone := common.Hash{}
		copy(clone[:], partial[:])
		delayedMessageMerklePartials = append(delayedMessageMerklePartials, clone)
	}
	var delayedMessageBacklog *DelayedMessageBacklog
	if s.delayedMessageBacklog != nil {
		delayedMessageBacklog = s.delayedMessageBacklog.clone()
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
		DelayedMessagesSeenRoot:            delayedMsgSeenRoot,
		MsgCount:                           s.MsgCount,
		BatchCount:                         s.BatchCount,
		DelayedMessagesRead:                s.DelayedMessagesRead,
		DelayedMessagesSeen:                s.DelayedMessagesSeen,
		DelayedMessageMerklePartials:       delayedMessageMerklePartials,
		delayedMessageBacklog:              delayedMessageBacklog,
		readCountFromBacklog:               s.readCountFromBacklog,
	}
}

func (s *State) AccumulateMessage(msg *arbostypes.MessageWithMetadata) error {
	// TODO: Unimplemented.
	return nil
}

func (s *State) AccumulateDelayedMessage(msg *DelayedInboxMessage) error {
	if s.seenDelayedMsgsAcc == nil {
		log.Debug("Initializing MelState's seenDelayedMsgsAcc")
		// This is very low cost hence better to reconstruct seenDelayedMsgsAcc from fresh partals instead of risking using a dirty acc
		acc, err := merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(ToPtrSlice(s.DelayedMessageMerklePartials))
		if err != nil {
			return err
		}
		s.seenDelayedMsgsAcc = acc
	}
	msgHash := msg.Hash()
	if _, err := s.seenDelayedMsgsAcc.Append(msgHash); err != nil {
		return err
	}
	if s.delayedMessageBacklog != nil {
		if err := s.delayedMessageBacklog.Add(
			&DelayedMessageBacklogEntry{
				Index:                       s.DelayedMessagesSeen,
				MsgHash:                     msgHash,
				MelStateParentChainBlockNum: s.ParentChainBlockNumber,
			}); err != nil {
			return err
		}
		// Found init message
		if s.DelayedMessagesSeen == 0 {
			s.delayedMessageBacklog.setInitMsg(msg)
		}
	}
	return nil
}

func (s *State) GenerateDelayedMessagesSeenMerklePartialsAndRoot() error {
	partialsPtrs, err := s.seenDelayedMsgsAcc.GetPartials()
	if err != nil {
		return err
	}
	s.DelayedMessageMerklePartials = FromPtrSlice(partialsPtrs)
	root, err := s.seenDelayedMsgsAcc.Root()
	if err != nil {
		return err
	}
	s.DelayedMessagesSeenRoot = root
	return nil
}

func (s *State) GetSeenDelayedMsgsAcc() *merkleAccumulator.MerkleAccumulator {
	return s.seenDelayedMsgsAcc
}

func (s *State) SetSeenDelayedMsgsAcc(acc *merkleAccumulator.MerkleAccumulator) {
	s.seenDelayedMsgsAcc = acc
}

func (s *State) GetDelayedMessageBacklog() *DelayedMessageBacklog {
	return s.delayedMessageBacklog
}

func (s *State) SetDelayedMessageBacklog(delayedMessageBacklog *DelayedMessageBacklog) {
	s.delayedMessageBacklog = delayedMessageBacklog
}

func (s *State) GetReadCountFromBacklog() uint64      { return s.readCountFromBacklog }
func (s *State) SetReadCountFromBacklog(count uint64) { s.readCountFromBacklog = count }

func (s *State) ReorgTo(newState *State) error {
	delayedMessageBacklog := s.delayedMessageBacklog
	if err := delayedMessageBacklog.reorg(newState.DelayedMessagesSeen); err != nil {
		return err
	}
	newState.delayedMessageBacklog = delayedMessageBacklog
	// Reset the pre-read delayed messages count since they havent been verified against latest state's merkle root
	newState.readCountFromBacklog = 0
	return nil
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
