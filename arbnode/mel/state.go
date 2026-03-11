// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package mel

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

// SplitPreimage validates that a preimage is exactly 2*common.HashLength bytes
// and splits it into left (previous accumulator) and right (message hash) halves.
func SplitPreimage(preimage []byte) (left, right common.Hash, err error) {
	if len(preimage) != 2*common.HashLength {
		return common.Hash{}, common.Hash{}, fmt.Errorf("invalid preimage length: %d, wanted %d", len(preimage), 2*common.HashLength)
	}
	return common.BytesToHash(preimage[:common.HashLength]), common.BytesToHash(preimage[common.HashLength:]), nil
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
	BatchCount                         uint64
	MsgCount                           uint64
	LocalMsgAccumulator                common.Hash
	DelayedMessagesRead                uint64
	DelayedMessagesSeen                uint64
	DelayedMessageInboxAcc             common.Hash
	DelayedMessageOutboxAcc            common.Hash

	msgPreimagesDest        daprovider.PreimagesMap
	delayedMsgPreimagesDest daprovider.PreimagesMap
	// delayedMsgPreimages is always populated during delayed message operations
	// (inbox push, pour, outbox pop) regardless of recording mode. It enables
	// the pour and pop operations in native mode without requiring full recording.
	delayedMsgPreimages map[common.Hash][]byte

	// initMsg holds the init delayed message (index 0) in memory. It is both
	// accumulated and read in the same block, so it may not be in the DB yet
	// when ReadDelayedMessage is called in native mode.
	initMsg *DelayedInboxMessage
}

// MessageConsumer is an interface to be implemented by readers of MEL such as transaction streamer of the nitro node
type MessageConsumer interface {
	PushMessages(
		ctx context.Context,
		firstMsgIdx uint64,
		messages []*arbostypes.MessageWithMetadata,
	) error
}

func (s *State) InitMsg() *DelayedInboxMessage { return s.initMsg }

func (s *State) Hash() common.Hash {
	encoded, err := rlp.EncodeToBytes(s)
	if err != nil {
		panic(err)
	}
	return crypto.Keccak256Hash(encoded)
}

// Clone performs a deep clone of the state struct to prevent any unintended
// mutations of pointers at runtime. LocalMsgAccumulator is zeroed because the
// extraction function rebuilds it from scratch per block. Delayed message
// accumulators (DelayedMessageInboxAcc, DelayedMessageOutboxAcc) are preserved
// because they carry state across blocks.
func (s *State) Clone() *State {
	batchPostingTarget := common.Address{}
	delayedMessageTarget := common.Address{}
	parentChainHash := common.Hash{}
	parentChainPrevHash := common.Hash{}
	delayedInboxAcc := common.Hash{}
	delayedOutboxAcc := common.Hash{}
	copy(batchPostingTarget[:], s.BatchPostingTargetAddress[:])
	copy(delayedMessageTarget[:], s.DelayedMessagePostingTargetAddress[:])
	copy(parentChainHash[:], s.ParentChainBlockHash[:])
	copy(parentChainPrevHash[:], s.ParentChainPreviousBlockHash[:])
	copy(delayedInboxAcc[:], s.DelayedMessageInboxAcc[:])
	copy(delayedOutboxAcc[:], s.DelayedMessageOutboxAcc[:])
	return &State{
		Version:                            s.Version,
		ParentChainId:                      s.ParentChainId,
		ParentChainBlockNumber:             s.ParentChainBlockNumber,
		BatchPostingTargetAddress:          batchPostingTarget,
		DelayedMessagePostingTargetAddress: delayedMessageTarget,
		ParentChainBlockHash:               parentChainHash,
		ParentChainPreviousBlockHash:       parentChainPrevHash,
		MsgCount:                           s.MsgCount,
		BatchCount:                         s.BatchCount,
		DelayedMessagesRead:                s.DelayedMessagesRead,
		DelayedMessagesSeen:                s.DelayedMessagesSeen,
		DelayedMessageInboxAcc:             delayedInboxAcc,
		DelayedMessageOutboxAcc:            delayedOutboxAcc,
		// we pass along msgPreimagesDest to continue recording of msg preimages
		msgPreimagesDest:        s.msgPreimagesDest,
		delayedMsgPreimagesDest: s.delayedMsgPreimagesDest,
		delayedMsgPreimages:     s.delayedMsgPreimages,
		initMsg:                 s.initMsg,
	}
}

func (s *State) AccumulateMessage(msg *arbostypes.MessageWithMetadata) error {
	msgBytes, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	msgHash := crypto.Keccak256Hash(msgBytes)
	preimage := append(s.LocalMsgAccumulator.Bytes(), msgHash.Bytes()...)
	newAcc := crypto.Keccak256Hash(preimage)
	if s.msgPreimagesDest != nil {
		keccakMap := s.msgPreimagesDest[arbutil.Keccak256PreimageType]
		keccakMap[newAcc] = preimage  // acc chain link
		keccakMap[msgHash] = msgBytes // message content
	}
	s.LocalMsgAccumulator = newAcc
	return nil
}

func (s *State) ensureDelayedMsgPreimages() {
	if s.delayedMsgPreimages == nil {
		s.delayedMsgPreimages = make(map[common.Hash][]byte)
	}
}

// AccumulateDelayedMessage pushes a delayed message onto the inbox accumulator hash chain.
func (s *State) AccumulateDelayedMessage(msg *DelayedInboxMessage) error {
	if s.DelayedMessagesSeen == 0 {
		s.initMsg = msg
	}
	s.ensureDelayedMsgPreimages()
	msgBytes, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	msgHash := crypto.Keccak256Hash(msgBytes)
	preimage := append(s.DelayedMessageInboxAcc.Bytes(), msgHash.Bytes()...)
	newAcc := crypto.Keccak256Hash(preimage)
	// Always record delayed message preimages for pour/pop operations
	s.delayedMsgPreimages[newAcc] = preimage
	s.delayedMsgPreimages[msgHash] = msgBytes
	// Also record to the delayed msg validation preimage map if in recording mode
	if s.delayedMsgPreimagesDest != nil {
		keccakMap := s.delayedMsgPreimagesDest[arbutil.Keccak256PreimageType]
		keccakMap[newAcc] = preimage
		keccakMap[msgHash] = msgBytes
	}
	s.DelayedMessageInboxAcc = newAcc
	return nil
}

// resolveDelayedPreimage resolves a preimage from the delayed message preimage map.
func (s *State) resolveDelayedPreimage(hash common.Hash) ([]byte, error) {
	if s.delayedMsgPreimages == nil {
		return nil, fmt.Errorf("delayed message preimage map not initialized")
	}
	preimage, ok := s.delayedMsgPreimages[hash]
	if !ok {
		return nil, fmt.Errorf("delayed message preimage not found for hash %s", hash.Hex())
	}
	return preimage, nil
}

// PourDelayedInboxToOutbox moves all items from the inbox to the outbox,
// reversing their order so that the first-seen message is popped first from the outbox.
// This implements the "pour" operation of the two-stack FIFO queue.
// The number of items to pour is DelayedMessagesSeen - DelayedMessagesRead (called when outbox is empty).
func (s *State) PourDelayedInboxToOutbox() error {
	inboxSize := s.DelayedMessagesSeen - s.DelayedMessagesRead
	if inboxSize == 0 {
		return nil
	}
	s.ensureDelayedMsgPreimages()
	// Pop all items from inbox (LIFO: last-seen comes out first) and Push onto outbox
	// in original order (first-seen first → it ends up on top)
	curr := s.DelayedMessageInboxAcc
	for i := range inboxSize {
		result, err := s.resolveDelayedPreimage(curr)
		if err != nil {
			return fmt.Errorf("error resolving inbox preimage during pour at position %d: %w", i, err)
		}
		prevAcc, msgHash, err := SplitPreimage(result)
		if err != nil {
			return fmt.Errorf("inbox preimage at position %d: %w", i, err)
		}
		preimage := append(s.DelayedMessageOutboxAcc.Bytes(), msgHash.Bytes()...)
		newAcc := crypto.Keccak256Hash(preimage)
		s.DelayedMessageOutboxAcc = newAcc
		s.delayedMsgPreimages[newAcc] = preimage
		if s.delayedMsgPreimagesDest != nil {
			keccakMap := s.delayedMsgPreimagesDest[arbutil.Keccak256PreimageType]
			keccakMap[newAcc] = preimage
			msgBytes, err := s.resolveDelayedPreimage(msgHash)
			if err != nil {
				return fmt.Errorf("error resolving inbox preimage of message hash during pour at position %d: %w", i, err)
			}
			keccakMap[msgHash] = msgBytes
		}
		curr = prevAcc
	}
	// Inbox is now empty
	s.DelayedMessageInboxAcc = common.Hash{}
	return nil
}

// PopDelayedOutbox pops the top item from the outbox accumulator, returning the message hash.
// If the outbox is empty, it pours the inbox first.
func (s *State) PopDelayedOutbox() (common.Hash, error) {
	if s.DelayedMessageOutboxAcc == (common.Hash{}) {
		if s.DelayedMessageInboxAcc == (common.Hash{}) {
			return common.Hash{}, fmt.Errorf("both inbox and outbox are empty, cannot pop")
		}
		if err := s.PourDelayedInboxToOutbox(); err != nil {
			return common.Hash{}, err
		}
	}
	result, err := s.resolveDelayedPreimage(s.DelayedMessageOutboxAcc)
	if err != nil {
		return common.Hash{}, fmt.Errorf("error resolving outbox preimage: %w", err)
	}
	prevOutbox, msgHash, err := SplitPreimage(result)
	if err != nil {
		return common.Hash{}, fmt.Errorf("outbox preimage: %w", err)
	}
	s.DelayedMessageOutboxAcc = prevOutbox
	return msgHash, nil
}

// RecordMsgPreimagesTo initializes the state's msgPreimagesDest to record preimages
// related to the extracted L2 messages needed for MEL validation into the given preimages map.
// When set, AccumulateMessage will record accumulator chain and message content preimages.
func (s *State) RecordMsgPreimagesTo(preimagesMap daprovider.PreimagesMap) error {
	if preimagesMap == nil {
		return errors.New("msg preimages recording destination cannot be nil")
	}
	if _, ok := preimagesMap[arbutil.Keccak256PreimageType]; !ok {
		preimagesMap[arbutil.Keccak256PreimageType] = make(map[common.Hash][]byte)
	}
	s.msgPreimagesDest = preimagesMap
	return nil
}

// RecordDelayedMsgPreimagesTo initializes the state's delayedMsgPreimagesDest to record
// preimages related to delayed messages needed for MEL validation into the given preimages map.
// When set, AccumulateDelayedMessage and PourDelayedInboxToOutbox will record accumulator
// chain and message content preimages.
func (s *State) RecordDelayedMsgPreimagesTo(preimagesMap daprovider.PreimagesMap) error {
	if preimagesMap == nil {
		return errors.New("delayed msg preimages recording destination cannot be nil")
	}
	if _, ok := preimagesMap[arbutil.Keccak256PreimageType]; !ok {
		preimagesMap[arbutil.Keccak256PreimageType] = make(map[common.Hash][]byte)
	}
	s.delayedMsgPreimagesDest = preimagesMap
	return nil
}

// RebuildDelayedMsgPreimages rebuilds the delayedMsgPreimages map from stored delayed
// messages. This is needed after loading state from DB (where private fields are nil)
// so that PourDelayedInboxToOutbox and PopDelayedOutbox can resolve preimages.
// It uses an O(N) algorithm with pivot finding to reconstruct both the outbox and inbox chains.
func (s *State) RebuildDelayedMsgPreimages(fetchDelayedMsg func(index uint64) (*DelayedInboxMessage, error)) error {
	if s.DelayedMessagesRead == s.DelayedMessagesSeen {
		return nil
	}
	s.delayedMsgPreimages = make(map[common.Hash][]byte)
	totalUnread := s.DelayedMessagesSeen - s.DelayedMessagesRead
	msgHashes := make([]common.Hash, totalUnread)
	msgBytesArr := make([][]byte, totalUnread)
	for i := range totalUnread {
		msg, err := fetchDelayedMsg(s.DelayedMessagesRead + i)
		if err != nil {
			return fmt.Errorf("error fetching delayed message at index %d: %w", s.DelayedMessagesRead+i, err)
		}
		msgBytes, err := rlp.EncodeToBytes(msg)
		if err != nil {
			return fmt.Errorf("error encoding delayed message at index %d: %w", s.DelayedMessagesRead+i, err)
		}
		msgHashes[i] = crypto.Keccak256Hash(msgBytes)
		msgBytesArr[i] = msgBytes
	}
	// Find pivot: messages [0..pivot-1] are in outbox, [pivot..totalUnread-1] are in inbox.
	// The outbox is built by pouring (push in reverse order), so its chain direction is reversed.
	var pivot uint64
	if s.DelayedMessageOutboxAcc == (common.Hash{}) {
		pivot = 0
	} else if s.DelayedMessageInboxAcc == (common.Hash{}) {
		pivot = totalUnread
	} else {
		// Mixed case: find pivot by trying inbox chains from the end (small inbox first)
		found := false
		for candidatePivot := totalUnread - 1; candidatePivot >= 1; candidatePivot-- {
			acc := common.Hash{}
			for i := candidatePivot; i < totalUnread; i++ {
				preimage := append(acc.Bytes(), msgHashes[i].Bytes()...)
				acc = crypto.Keccak256Hash(preimage)
			}
			if acc == s.DelayedMessageInboxAcc {
				pivot = candidatePivot
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("failed to find pivot: neither outbox acc %s nor inbox acc %s matched any partition", s.DelayedMessageOutboxAcc.Hex(), s.DelayedMessageInboxAcc.Hex())
		}
	}
	// recordPreimage hashes msgHashes[i] onto acc, stores preimages to both maps, and returns the new accumulator.
	recordPreimage := func(acc common.Hash, i uint64) common.Hash {
		preimage := append(acc.Bytes(), msgHashes[i].Bytes()...)
		newAcc := crypto.Keccak256Hash(preimage)
		s.delayedMsgPreimages[newAcc] = preimage
		s.delayedMsgPreimages[msgHashes[i]] = msgBytesArr[i]
		if s.delayedMsgPreimagesDest != nil {
			keccakMap := s.delayedMsgPreimagesDest[arbutil.Keccak256PreimageType]
			keccakMap[newAcc] = preimage
			keccakMap[msgHashes[i]] = msgBytesArr[i]
		}
		return newAcc
	}
	// Build outbox chain: messages [0..pivot-1] pushed in reverse order [pivot-1, pivot-2, ..., 0]
	if pivot > 0 {
		acc := common.Hash{}
		// #nosec G115
		for i := int64(pivot - 1); i >= 0; i-- {
			acc = recordPreimage(acc, uint64(i))
		}
		if acc != s.DelayedMessageOutboxAcc {
			return fmt.Errorf("outbox accumulator mismatch after rebuild: got %s, want %s", acc.Hex(), s.DelayedMessageOutboxAcc.Hex())
		}
	}
	// Build inbox chain: messages [pivot..totalUnread-1] in forward order
	if pivot < totalUnread {
		acc := common.Hash{}
		for i := pivot; i < totalUnread; i++ {
			acc = recordPreimage(acc, i)
		}
		if acc != s.DelayedMessageInboxAcc {
			return fmt.Errorf("inbox accumulator mismatch after rebuild: got %s, want %s", acc.Hex(), s.DelayedMessageInboxAcc.Hex())
		}
	}
	return nil
}
