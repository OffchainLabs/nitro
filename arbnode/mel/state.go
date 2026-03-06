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

	msgPreimagesDest daprovider.PreimagesMap
	// delayedMsgPreimages is always populated during delayed message operations
	// (inbox push, pour, outbox pop) regardless of recording mode. It enables
	// the pour and pop operations in native mode without requiring full recording.
	delayedMsgPreimages map[common.Hash][]byte
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
	encoded, err := rlp.EncodeToBytes(s)
	if err != nil {
		panic(err)
	}
	return crypto.Keccak256Hash(encoded)
}

// Clone performs a deep clone of the state struct to prevent any unintended
// mutations of pointers at runtime. Accumulator fields (LocalMsgAccumulator,
// DelayedMessageInboxAcc, DelayedMessageOutboxAcc) are zeroed because the
// extraction function rebuilds them from scratch.
func (s *State) Clone() *State {
	batchPostingTarget := common.Address{}
	delayedMessageTarget := common.Address{}
	parentChainHash := common.Hash{}
	parentChainPrevHash := common.Hash{}
	copy(batchPostingTarget[:], s.BatchPostingTargetAddress[:])
	copy(delayedMessageTarget[:], s.DelayedMessagePostingTargetAddress[:])
	copy(parentChainHash[:], s.ParentChainBlockHash[:])
	copy(parentChainPrevHash[:], s.ParentChainPreviousBlockHash[:])
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
		// we pass along msgPreimagesDest to continue recording of msg preimages
		msgPreimagesDest: s.msgPreimagesDest,
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
	// Also record to the validation preimage map if in recording mode
	if s.msgPreimagesDest != nil {
		keccakMap := s.msgPreimagesDest[arbutil.Keccak256PreimageType]
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
	// Pop all items from inbox (LIFO: last-seen comes out first)
	msgHashes := make([]common.Hash, inboxSize)
	curr := s.DelayedMessageInboxAcc
	for i := inboxSize; i > 0; i-- {
		result, err := s.resolveDelayedPreimage(curr)
		if err != nil {
			return fmt.Errorf("error resolving inbox preimage during pour at position %d: %w", i, err)
		}
		if len(result) != 2*common.HashLength {
			return fmt.Errorf("invalid inbox preimage length: %d, wanted %d", len(result), 2*common.HashLength)
		}
		prevAcc := common.BytesToHash(result[:common.HashLength])
		msgHash := common.BytesToHash(result[common.HashLength:])
		msgHashes[i-1] = msgHash
		curr = prevAcc
	}
	// Push items onto outbox in original order (first-seen first → it ends up on top)
	for _, msgHash := range msgHashes {
		preimage := append(s.DelayedMessageOutboxAcc.Bytes(), msgHash.Bytes()...)
		newAcc := crypto.Keccak256Hash(preimage)
		s.delayedMsgPreimages[newAcc] = preimage
		if s.msgPreimagesDest != nil {
			keccakMap := s.msgPreimagesDest[arbutil.Keccak256PreimageType]
			keccakMap[newAcc] = preimage
		}
		s.DelayedMessageOutboxAcc = newAcc
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
	if len(result) != 2*common.HashLength {
		return common.Hash{}, fmt.Errorf("invalid outbox preimage length: %d, wanted %d", len(result), 2*common.HashLength)
	}
	prevOutbox := common.BytesToHash(result[:common.HashLength])
	msgHash := common.BytesToHash(result[common.HashLength:])
	s.DelayedMessageOutboxAcc = prevOutbox
	return msgHash, nil
}

// RecordMsgPreimagesTo initializes the state's msgPreimagesDest to record preimages
// related to the extracted messages needed for MEL validation into the given preimages map.
// When set, AccumulateMessage and AccumulateDelayedMessage will record accumulator chain
// and message content preimages.
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
