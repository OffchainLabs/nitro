package mel

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

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
	BatchCount                         uint64
	MsgCount                           uint64
	MsgRoot                            common.Hash
	DelayedMessagesRead                uint64
	DelayedMessagesSeen                uint64
	DelayedMessagesSeenRoot            common.Hash
	MessageMerklePartials              []common.Hash `rlp:"optional"`
	DelayedMessageMerklePartials       []common.Hash `rlp:"optional"`

	delayedMessageBacklog *DelayedMessageBacklog // delayedMessageBacklog is initialized once in the Start fsm step of mel runner and is persisted across all future states
	readCountFromBacklog  uint64                 // delayed messages with index lower than this count have been pre-read and we have hashes for them in-memory for verification

	// seenDelayedMsgsAcc is MerkleAccumulator that accumulates delayed messages seen
	// from parent chain. It resets after the current melstate is finished generating
	// and is reinitialized using appropriate DelayedMessageMerklePartials of the state
	seenDelayedMsgsAcc *merkleAccumulator.MerkleAccumulator
	// msgsAcc is MerkleAccumulator that accumulates all the L2 messages extracted. It
	// resets after the current melstate is finished generating and is reinitialized using
	// appropriate MessageMerklePartials of the state
	msgsAcc *merkleAccumulator.MerkleAccumulator
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
	msgAccRoot := common.Hash{}
	delayedMsgSeenRoot := common.Hash{}
	copy(batchPostingTarget[:], s.BatchPostingTargetAddress[:])
	copy(delayedMessageTarget[:], s.DelayedMessagePostingTargetAddress[:])
	copy(parentChainHash[:], s.ParentChainBlockHash[:])
	copy(parentChainPrevHash[:], s.ParentChainPreviousBlockHash[:])
	copy(msgAccRoot[:], s.MsgRoot[:])
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
		MsgRoot:                            msgAccRoot,
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
	if s.msgsAcc == nil {
		log.Debug("Initializing MelState's msgsAcc")
		// This is very low cost hence better to reconstruct msgsAcc from fresh partals instead of risking using a dirty acc
		acc, err := merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(ToPtrSlice(s.MessageMerklePartials))
		if err != nil {
			return err
		}
		s.msgsAcc = acc
	}
	msgBytes, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	// In recording mode this would also record the message preimages needed for MEL validation
	if _, err := s.msgsAcc.Append(msg.Hash(), msgBytes...); err != nil {
		return err
	}
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

func (s *State) GenerateMessageMerklePartialsAndRoot() error {
	var err error
	s.MessageMerklePartials, s.MsgRoot, err = getPartialsAndRoot(s.msgsAcc)
	return err
}

func (s *State) GenerateDelayedMessagesSeenMerklePartialsAndRoot() error {
	var err error
	s.DelayedMessageMerklePartials, s.DelayedMessagesSeenRoot, err = getPartialsAndRoot(s.seenDelayedMsgsAcc)
	return err
}

func getPartialsAndRoot(acc *merkleAccumulator.MerkleAccumulator) ([]common.Hash, common.Hash, error) {
	partialsPtrs, err := acc.GetPartials()
	if err != nil {
		return nil, common.Hash{}, err
	}
	partials := FromPtrSlice(partialsPtrs)
	root, err := acc.Root()
	if err != nil {
		return nil, common.Hash{}, err
	}
	return partials, root, err
}

func (s *State) SetMsgsAcc(acc *merkleAccumulator.MerkleAccumulator) {
	s.msgsAcc = acc
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
