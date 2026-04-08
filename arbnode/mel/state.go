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
	"github.com/offchainlabs/nitro/util/containers"
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
	Version                uint16
	VersionActivationBlock uint64

	ParentChainId                      uint64
	ParentChainBlockNumber             uint64
	BatchPostingTargetAddress          common.Address
	DelayedMessagePostingTargetAddress common.Address
	ParentChainBlockHash               common.Hash
	ParentChainPreviousBlockHash       common.Hash
	BatchCount                         uint64
	MsgCount                           uint64
	LocalMsgAccumulator                common.Hash // starts at zero hash for each clone; updated only by AccumulateMessage; represents messages accumulated during processing of this specific parent chain block
	DelayedMessagesRead                uint64
	DelayedMessagesSeen                uint64
	DelayedMessageInboxAcc             common.Hash
	DelayedMessageOutboxAcc            common.Hash

	// Pending value changes to DelayedMessagePostingTargetAddress and
	// BatchPostingTargetAddress from a MELConfig upgrade event
	PendingInbox          common.Address
	PendingSequencerInbox common.Address

	msgPreimagesDest        daprovider.PreimagesMap
	delayedMsgPreimagesDest daprovider.PreimagesMap
	// delayedMsgPreimages is always populated during delayed message operations
	// (inbox push, pour, outbox pop) regardless of recording mode. It enables
	// the pour and pop operations in native mode without requiring full recording.
	delayedMsgPreimages *containers.LruCache[common.Hash, []byte]

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
	pendingInbox := common.Address{}
	pendingSequencerInbox := common.Address{}
	copy(batchPostingTarget[:], s.BatchPostingTargetAddress[:])
	copy(delayedMessageTarget[:], s.DelayedMessagePostingTargetAddress[:])
	copy(parentChainHash[:], s.ParentChainBlockHash[:])
	copy(parentChainPrevHash[:], s.ParentChainPreviousBlockHash[:])
	copy(delayedInboxAcc[:], s.DelayedMessageInboxAcc[:])
	copy(delayedOutboxAcc[:], s.DelayedMessageOutboxAcc[:])
	copy(pendingInbox[:], s.PendingInbox[:])
	copy(pendingSequencerInbox[:], s.PendingSequencerInbox[:])
	return &State{
		Version:                            s.Version,
		VersionActivationBlock:             s.VersionActivationBlock,
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
		PendingInbox:                       pendingInbox,
		PendingSequencerInbox:              pendingSequencerInbox,
		// LocalMsgAccumulator is intentionally not copied — each cloned state
		// starts a fresh hash chain for its own batch of accumulated messages.
		//
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
	preimage := make([]byte, 0, 2*common.HashLength)
	preimage = append(preimage, s.LocalMsgAccumulator.Bytes()...)
	preimage = append(preimage, msgHash.Bytes()...)
	newAcc := crypto.Keccak256Hash(preimage)
	if s.msgPreimagesDest != nil {
		keccakMap, ok := s.msgPreimagesDest[arbutil.Keccak256PreimageType]
		if !ok {
			return errors.New("keccak256 preimage map not initialized in msgPreimagesDest")
		}
		keccakMap[newAcc] = preimage  // acc chain link
		keccakMap[msgHash] = msgBytes // message content
	}
	s.LocalMsgAccumulator = newAcc
	return nil
}

func (s *State) ensureDelayedMsgPreimages() {
	if s.delayedMsgPreimages == nil {
		s.delayedMsgPreimages = containers.NewLruCache[common.Hash, []byte](0)
	}
}

// AccumulateDelayedMessage pushes a delayed message onto the inbox accumulator hash chain.
func (s *State) AccumulateDelayedMessage(msg *DelayedInboxMessage) error {
	if s.DelayedMessagesSeen == 0 {
		s.initMsg = msg
	}
	s.ensureDelayedMsgPreimages()
	// totalUnread is the count of live unread messages before this accumulation
	// (DelayedMessagesSeen is incremented by the caller after this returns).
	// Resize to totalUnread+1 to ensure capacity for the entry about to be added.
	// Stale entries (e.g. old inbox preimages left after pour) are the oldest in
	// LRU order and will be evicted first if the cache is full, so sizing for
	// live entries only is safe.
	// #nosec G115
	totalUnread := int(s.DelayedMessagesSeen - s.DelayedMessagesRead)
	if s.delayedMsgPreimages.Size() <= totalUnread {
		s.delayedMsgPreimages.Resize(totalUnread + 1)
	}
	msgBytes, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	msgHash := crypto.Keccak256Hash(msgBytes)
	preimage := append(s.DelayedMessageInboxAcc.Bytes(), msgHash.Bytes()...)
	newAcc := crypto.Keccak256Hash(preimage)
	// Always record delayed message preimages for pour/pop operations
	s.delayedMsgPreimages.Add(newAcc, preimage)
	// Also record to the delayed msg validation preimage map if in recording mode
	if s.delayedMsgPreimagesDest != nil {
		keccakMap, ok := s.delayedMsgPreimagesDest[arbutil.Keccak256PreimageType]
		if !ok {
			return errors.New("keccak256 preimage map not initialized in delayedMsgPreimagesDest")
		}
		keccakMap[newAcc] = preimage
		keccakMap[msgHash] = msgBytes
	}
	s.DelayedMessageInboxAcc = newAcc
	return nil
}

// resolveDelayedPreimage resolves a preimage from the delayed message preimage map.
func (s *State) resolveDelayedPreimage(hash common.Hash) ([]byte, error) {
	if s.delayedMsgPreimages == nil {
		return nil, fmt.Errorf("delayed message preimage cache not initialized")
	}
	preimage, ok := s.delayedMsgPreimages.Peek(hash)
	if !ok {
		return nil, fmt.Errorf("%w: for hash: %s", ErrDelayedMessagePreimageNotFound, hash.Hex())
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
	// During pour, existing inbox entries are read via Peek (no LRU promotion)
	// while new outbox entries are Added. Both coexist in the cache simultaneously,
	// requiring at least 2*inboxSize capacity. We use 3*inboxSize to leave headroom
	// for post-pour accumulations before the next resize.
	// #nosec G115
	if s.delayedMsgPreimages.Size() < 3*int(inboxSize) {
		s.delayedMsgPreimages.Resize(3 * int(inboxSize))
	}
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
		s.delayedMsgPreimages.Add(newAcc, preimage)
		if s.delayedMsgPreimagesDest != nil {
			keccakMap, ok := s.delayedMsgPreimagesDest[arbutil.Keccak256PreimageType]
			if !ok {
				return errors.New("keccak256 preimage map not initialized in delayedMsgPreimagesDest")
			}
			keccakMap[newAcc] = preimage
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
		return common.Hash{}, fmt.Errorf("error resolving outbox preimage, delayedMsgPreimages of size: %d, seenCount: %d, readCount: %d, err: %w", s.delayedMsgPreimages.Size(), s.DelayedMessagesSeen, s.DelayedMessagesRead, err)
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
// When set, AccumulateDelayedMessage will record accumulator chain and message content preimages,
// whereas PourDelayedInboxToOutbox record only accumulator chain preimages
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

// fetchAndHashUnreadMessages fetches all unread delayed messages from the database
// and returns their keccak256 hashes and RLP-encoded bytes. The returned slices
// are indexed relative to DelayedMessagesRead (i.e., index 0 corresponds to
// message at DelayedMessagesRead).
func (s *State) fetchAndHashUnreadMessages(
	totalUnread uint64,
	fetchDelayedMsg func(index uint64) (*DelayedInboxMessage, error),
) ([]common.Hash, [][]byte, error) {
	msgHashes := make([]common.Hash, totalUnread)
	msgBytesArr := make([][]byte, totalUnread)
	for i := range totalUnread {
		msg, err := fetchDelayedMsg(s.DelayedMessagesRead + i)
		if err != nil {
			return nil, nil, fmt.Errorf("error fetching delayed message at index %d: %w", s.DelayedMessagesRead+i, err)
		}
		msgBytes, err := rlp.EncodeToBytes(msg)
		if err != nil {
			return nil, nil, fmt.Errorf("error encoding delayed message at index %d: %w", s.DelayedMessagesRead+i, err)
		}
		msgHashes[i] = crypto.Keccak256Hash(msgBytes)
		msgBytesArr[i] = msgBytes
	}
	return msgHashes, msgBytesArr, nil
}

// findPivot determines how many of the unread messages are in the outbox vs
// the inbox. Returns -1 if no valid pivot can be found (legacy fallback failure).
func (s *State) findPivot(totalUnread uint64, msgHashes []common.Hash) int {
	if s.DelayedMessageOutboxAcc == (common.Hash{}) {
		return 0
	}
	if s.DelayedMessageInboxAcc == (common.Hash{}) {
		// #nosec G115
		return int(totalUnread)
	}
	// Legacy fallback: try each candidate pivot, starting with the smallest
	// inbox (most likely case) and working backward.
	for candidatePivot := totalUnread - 1; candidatePivot >= 1; candidatePivot-- {
		acc := common.Hash{}
		for i := candidatePivot; i < totalUnread; i++ {
			preimage := append(acc.Bytes(), msgHashes[i].Bytes()...)
			acc = crypto.Keccak256Hash(preimage)
		}
		if acc == s.DelayedMessageInboxAcc {
			// #nosec G115
			return int(candidatePivot)
		}
	}
	return -1
}

// buildHashChain reconstructs a hash chain by iterating from start (inclusive)
// toward end (exclusive) with the given step (+1 or -1). Each message hash is
// accumulated onto a running hash (starting from zero). Preimages are stored in
// the LRU cache and optionally the validation preimage map.
func (s *State) buildHashChain(start, end, step int, msgHashes []common.Hash, msgBytesArr [][]byte) (common.Hash, error) {
	acc := common.Hash{}
	for i := start; i != end; i += step {
		preimage := append(acc.Bytes(), msgHashes[i].Bytes()...)
		newAcc := crypto.Keccak256Hash(preimage)
		s.delayedMsgPreimages.Add(newAcc, preimage)
		if s.delayedMsgPreimagesDest != nil {
			keccakMap, ok := s.delayedMsgPreimagesDest[arbutil.Keccak256PreimageType]
			if !ok {
				return common.Hash{}, errors.New("keccak256 preimage map not initialized in delayedMsgPreimagesDest")
			}
			keccakMap[newAcc] = preimage
			keccakMap[msgHashes[i]] = msgBytesArr[i]
		}
		acc = newAcc
	}
	return acc, nil
}

// RebuildDelayedMsgPreimages reconstructs the in-memory preimage cache from
// delayed messages stored in the database. This is needed after loading state
// from DB (where the cache is nil), after reorgs, and periodically for memory
// cleanup.
//
// The delayed message queue is a two-stack FIFO: an inbox accumulator and an
// outbox accumulator. Unread messages (indices [DelayedMessagesRead, DelayedMessagesSeen))
// are split into two groups by a pivot (the outbox size):
//   - Outbox [0..pivot): messages already poured, chain built in reverse order
//   - Inbox  [pivot..N): messages not yet poured, chain built in forward order
//
// The pivot is found via an O(N²) search that tries each candidate partition
// until the recomputed inbox accumulator matches.
func (s *State) RebuildDelayedMsgPreimages(fetchDelayedMsg func(index uint64) (*DelayedInboxMessage, error)) error {
	if s.DelayedMessagesRead == s.DelayedMessagesSeen {
		return nil
	}
	totalUnread := s.DelayedMessagesSeen - s.DelayedMessagesRead
	// #nosec G115
	s.delayedMsgPreimages = containers.NewLruCache[common.Hash, []byte](int(totalUnread))
	msgHashes, msgBytesArr, err := s.fetchAndHashUnreadMessages(totalUnread, fetchDelayedMsg)
	if err != nil {
		return err
	}
	pivot := s.findPivot(totalUnread, msgHashes)
	if pivot < 0 {
		return fmt.Errorf("failed to find pivot: neither outbox acc %s nor inbox acc %s matched any partition",
			s.DelayedMessageOutboxAcc.Hex(), s.DelayedMessageInboxAcc.Hex())
	}
	// Rebuild outbox chain: messages [0..pivot) in reverse order (pivot-1, pivot-2, ..., 0)
	if pivot > 0 {
		acc, err := s.buildHashChain(pivot-1, -1, -1, msgHashes, msgBytesArr)
		if err != nil {
			return fmt.Errorf("error building outbox hash chain: %w", err)
		}
		if acc != s.DelayedMessageOutboxAcc {
			return fmt.Errorf("outbox accumulator mismatch after rebuild: got %s, want %s", acc.Hex(), s.DelayedMessageOutboxAcc.Hex())
		}
	}
	// Rebuild inbox chain: messages [pivot..totalUnread) in forward order
	// #nosec G115
	if pivot < int(totalUnread) {
		// #nosec G115
		acc, err := s.buildHashChain(pivot, int(totalUnread), 1, msgHashes, msgBytesArr)
		if err != nil {
			return fmt.Errorf("error building inbox hash chain: %w", err)
		}
		if acc != s.DelayedMessageInboxAcc {
			return fmt.Errorf("inbox accumulator mismatch after rebuild: got %s, want %s", acc.Hex(), s.DelayedMessageInboxAcc.Hex())
		}
	}
	return nil
}
