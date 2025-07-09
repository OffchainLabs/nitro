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
	DelayedMessagesSeenRoot            common.Hash
	MessageAccumulator                 common.Hash
	MsgCount                           uint64
	BatchCount                         uint64
	DelayedMessagesRead                uint64
	DelayedMessagedSeen                uint64
	DelayedMessageMerklePartials       []common.Hash `rlp:"optional"`

	// delayedMessageBacklog is initialized once in the Start fsm step of mel runner and is persisted across all future states
	delayedMessageBacklog *DelayedMessageBacklog

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
		delayedMessages []*DelayedInboxMessage,
	) error
	DelayedMessageDatabase
}

type DelayedMessageDatabase interface {
	ReadDelayedMessage(
		ctx context.Context,
		state *State,
		index uint64,
	) (*DelayedInboxMessage, error)
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
type InitialStateFetcher interface {
	FetchInitialState(
		ctx context.Context, parentChainBlockHash common.Hash,
	) (*State, error)
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
	delayedMsgAcc := common.Hash{}
	copy(batchPostingTarget[:], s.BatchPostingTargetAddress[:])
	copy(delayedMessageTarget[:], s.DelayedMessagePostingTargetAddress[:])
	copy(parentChainHash[:], s.ParentChainBlockHash[:])
	copy(parentChainPrevHash[:], s.ParentChainPreviousBlockHash[:])
	copy(delayedMsgAcc[:], s.DelayedMessagesSeenRoot[:])
	copy(msgAcc[:], s.MessageAccumulator[:])
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
		DelayedMessagesSeenRoot:            delayedMsgAcc,
		MessageAccumulator:                 msgAcc,
		MsgCount:                           s.MsgCount,
		BatchCount:                         s.BatchCount,
		DelayedMessagesRead:                s.DelayedMessagesRead,
		DelayedMessagedSeen:                s.DelayedMessagedSeen,
		DelayedMessageMerklePartials:       delayedMessageMerklePartials,
		delayedMessageBacklog:              delayedMessageBacklog,
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
	if _, err := s.seenDelayedMsgsAcc.Append(msg.Hash()); err != nil {
		return err
	}
	merkleRoot, err := s.seenDelayedMsgsAcc.Root()
	if err != nil {
		return err
	}
	if s.delayedMessageBacklog != nil {
		if err := s.delayedMessageBacklog.Add(
			&DelayedMessageBacklogEntry{
				Index:                       s.DelayedMessagedSeen,
				MerkleRoot:                  merkleRoot,
				MelStateParentChainBlockNum: s.ParentChainBlockNumber,
			}); err != nil {
			return err
		}
		// Found init message
		if s.DelayedMessagedSeen == 0 {
			s.delayedMessageBacklog.setInitMsg(msg)
		}
	}
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

func (s *State) GetDelayedMessageBacklog() *DelayedMessageBacklog {
	return s.delayedMessageBacklog
}

func (s *State) SetDelayedMessageBacklog(delayedMessageBacklog *DelayedMessageBacklog) {
	s.delayedMessageBacklog = delayedMessageBacklog
}

func (s *State) ReorgTo(newState *State) error {
	delayedMessageBacklog := s.delayedMessageBacklog
	if err := delayedMessageBacklog.reorg(newState.DelayedMessagedSeen); err != nil {
		return err
	}
	newState.delayedMessageBacklog = delayedMessageBacklog
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
