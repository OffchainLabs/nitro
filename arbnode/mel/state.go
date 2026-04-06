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

// keccakPreimages returns the Keccak256 preimage sub-map from a PreimagesMap,
// or an error if the sub-map has not been initialized.
func keccakPreimages(dest daprovider.PreimagesMap) (map[common.Hash][]byte, error) {
	m, ok := dest[arbutil.Keccak256PreimageType]
	if !ok {
		return nil, errors.New("keccak256 preimage map not initialized")
	}
	return m, nil
}

// recordDelayedChainLink stores a hash chain link preimage in the LRU cache
// and optionally in the validation preimage map.
func (s *State) recordDelayedChainLink(newAcc common.Hash, preimage []byte) error {
	s.delayedMsgPreimages.Add(newAcc, preimage)
	if s.delayedMsgPreimagesDest == nil {
		return nil
	}
	keccakMap, err := keccakPreimages(s.delayedMsgPreimagesDest)
	if err != nil {
		return err
	}
	keccakMap[newAcc] = preimage
	return nil
}

// recordDelayedContent stores message content in the validation preimage map
// when recording is enabled. No-op when not recording.
func (s *State) recordDelayedContent(hash common.Hash, content []byte) error {
	if s.delayedMsgPreimagesDest == nil {
		return nil
	}
	keccakMap, err := keccakPreimages(s.delayedMsgPreimagesDest)
	if err != nil {
		return err
	}
	keccakMap[hash] = content
	return nil
}

// HashChainLinkHash computes the next accumulator in a Keccak256 hash chain
// without allocating the preimage. Use this when only the hash is needed.
func HashChainLinkHash(prevAcc, itemHash common.Hash) common.Hash {
	var buf [2 * common.HashLength]byte
	copy(buf[:common.HashLength], prevAcc[:])
	copy(buf[common.HashLength:], itemHash[:])
	return crypto.Keccak256Hash(buf[:])
}

// HashChainLink computes the next accumulator in a Keccak256 hash chain.
// Returns the new accumulator hash and the 64-byte preimage (prevAcc || itemHash).
// This is the single canonical implementation of the hash chain step used throughout
// MEL for both message and delayed message accumulators, in native and replay modes.
func HashChainLink(prevAcc, itemHash common.Hash) (newAcc common.Hash, preimage []byte) {
	preimage = make([]byte, 2*common.HashLength)
	copy(preimage[:common.HashLength], prevAcc[:])
	copy(preimage[common.HashLength:], itemHash[:])
	return crypto.Keccak256Hash(preimage), preimage
}

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
	LocalMsgAccumulator                common.Hash // zeroed on Clone(); updated only by AccumulateMessage. In production, each clone processes exactly one parent chain block.
	DelayedMessagesRead                uint64
	DelayedMessagesSeen                uint64
	DelayedMessageInboxAcc             common.Hash
	DelayedMessageOutboxAcc            common.Hash

	msgPreimagesDest        daprovider.PreimagesMap
	delayedMsgPreimagesDest daprovider.PreimagesMap
	// delayedMsgPreimages is populated during inbox push and pour operations
	// regardless of recording mode. Pop reads from this cache via Peek.
	// It enables the pour and pop operations in native mode without requiring full recording.
	delayedMsgPreimages *containers.LruCache[common.Hash, []byte]

	// initMsg holds the init delayed message (index 0), set during the first
	// AccumulateDelayedMessage call (when DelayedMessagesSeen is 0). It persists
	// through Clone() and is accessed by Database.ReadDelayedMessage as a fallback
	// when the message has not yet been written to the DB (accumulated and read in
	// the same block).
	initMsg *DelayedInboxMessage
}

// MessageConsumer is an interface for downstream consumers of messages extracted by MEL.
type MessageConsumer interface {
	PushMessages(
		ctx context.Context,
		firstMsgIdx uint64,
		messages []*arbostypes.MessageWithMetadata,
	) error
}

func (s *State) InitMsg() *DelayedInboxMessage { return s.initMsg }

func (s *State) Hash() (common.Hash, error) {
	encoded, err := rlp.EncodeToBytes(s)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to RLP-encode MEL state at block %d: %w", s.ParentChainBlockNumber, err)
	}
	return crypto.Keccak256Hash(encoded), nil
}

// Validate checks structural invariants of the state.
func (s *State) Validate() error {
	if s.DelayedMessagesSeen < s.DelayedMessagesRead {
		return fmt.Errorf("invalid MEL state at block %d: DelayedMessagesSeen (%d) < DelayedMessagesRead (%d)", s.ParentChainBlockNumber, s.DelayedMessagesSeen, s.DelayedMessagesRead)
	}
	if s.DelayedMessageOutboxAcc != (common.Hash{}) && s.DelayedMessagesSeen <= s.DelayedMessagesRead {
		return fmt.Errorf("invalid MEL state at block %d: non-zero DelayedMessageOutboxAcc but no unread messages (seen=%d, read=%d)", s.ParentChainBlockNumber, s.DelayedMessagesSeen, s.DelayedMessagesRead)
	}
	return nil
}

// Clone copies all state-tracking fields by value and zeroes LocalMsgAccumulator
// because the extraction function rebuilds it from scratch per block. Delayed
// message accumulators (DelayedMessageInboxAcc, DelayedMessageOutboxAcc) are
// preserved because they carry state across blocks.
func (s *State) Clone() *State {
	// common.Hash and common.Address are fixed-size arrays, copied by value on assignment.
	return &State{
		Version:                            s.Version,
		ParentChainId:                      s.ParentChainId,
		ParentChainBlockNumber:             s.ParentChainBlockNumber,
		BatchPostingTargetAddress:          s.BatchPostingTargetAddress,
		DelayedMessagePostingTargetAddress: s.DelayedMessagePostingTargetAddress,
		ParentChainBlockHash:               s.ParentChainBlockHash,
		ParentChainPreviousBlockHash:       s.ParentChainPreviousBlockHash,
		MsgCount:                           s.MsgCount,
		BatchCount:                         s.BatchCount,
		DelayedMessagesRead:                s.DelayedMessagesRead,
		DelayedMessagesSeen:                s.DelayedMessagesSeen,
		DelayedMessageInboxAcc:             s.DelayedMessageInboxAcc,
		DelayedMessageOutboxAcc:            s.DelayedMessageOutboxAcc,
		// LocalMsgAccumulator is intentionally not copied — each cloned state
		// starts a fresh hash chain for its own batch of accumulated messages.
		//
		// we pass along msgPreimagesDest to continue recording of msg preimages
		msgPreimagesDest:        s.msgPreimagesDest,
		delayedMsgPreimagesDest: s.delayedMsgPreimagesDest,
		// delayedMsgPreimages is intentionally shared (not deep-copied) between
		// the original and cloned state. This is safe because the FSM processes
		// blocks sequentially: only the post-state is used going forward, and
		// the pre-state is never read concurrently. Do NOT use the pre-state
		// after cloning if the post-state may be mutated concurrently.
		delayedMsgPreimages: s.delayedMsgPreimages,
		initMsg:             s.initMsg,
	}
}

// AccumulateMessage appends a message to the local accumulator hash chain and
// increments MsgCount.
func (s *State) AccumulateMessage(msg *arbostypes.MessageWithMetadata) error {
	msgBytes, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	msgHash := crypto.Keccak256Hash(msgBytes)
	newAcc, preimage := HashChainLink(s.LocalMsgAccumulator, msgHash)
	if s.msgPreimagesDest != nil {
		keccakMap, err := keccakPreimages(s.msgPreimagesDest)
		if err != nil {
			return err
		}
		keccakMap[newAcc] = preimage  // acc chain link
		keccakMap[msgHash] = msgBytes // message content
	}
	s.LocalMsgAccumulator = newAcc
	s.MsgCount++
	return nil
}

func (s *State) ensureDelayedMsgPreimages() {
	if s.delayedMsgPreimages == nil {
		s.delayedMsgPreimages = containers.NewLruCache[common.Hash, []byte](16)
	}
}

// IncrementBatchCount increments the batch count and returns the new value.
func (s *State) IncrementBatchCount() uint64 {
	s.BatchCount++
	return s.BatchCount
}

// IncrementDelayedMessagesRead increments the delayed messages read counter
// and returns the new value. Returns an error if the counter would exceed
// DelayedMessagesSeen.
func (s *State) IncrementDelayedMessagesRead() (uint64, error) {
	if s.DelayedMessagesRead >= s.DelayedMessagesSeen {
		return 0, fmt.Errorf("cannot increment DelayedMessagesRead (%d) beyond DelayedMessagesSeen (%d)", s.DelayedMessagesRead, s.DelayedMessagesSeen)
	}
	s.DelayedMessagesRead++
	return s.DelayedMessagesRead, nil
}

// AccumulateDelayedMessage pushes a delayed message onto the inbox accumulator
// hash chain and increments DelayedMessagesSeen.
func (s *State) AccumulateDelayedMessage(msg *DelayedInboxMessage) error {
	if s.DelayedMessagesSeen == 0 {
		s.initMsg = msg
	}
	s.ensureDelayedMsgPreimages()
	// totalUnread is the count of live unread messages before this accumulation.
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
	newAcc, preimage := HashChainLink(s.DelayedMessageInboxAcc, msgHash)
	if err := s.recordDelayedChainLink(newAcc, preimage); err != nil {
		return err
	}
	if err := s.recordDelayedContent(msgHash, msgBytes); err != nil {
		return err
	}
	s.DelayedMessageInboxAcc = newAcc
	s.DelayedMessagesSeen++
	return nil
}

// resolveDelayedPreimage resolves a preimage from the delayed message preimage map.
func (s *State) resolveDelayedPreimage(hash common.Hash) ([]byte, error) {
	if s.delayedMsgPreimages == nil {
		return nil, fmt.Errorf("delayed message preimage cache not initialized")
	}
	preimage, ok := s.delayedMsgPreimages.Peek(hash)
	if !ok {
		return nil, fmt.Errorf("%w: for hash: %s (cache size: %d, capacity: %d)", ErrDelayedMessagePreimageNotFound, hash.Hex(), s.delayedMsgPreimages.Len(), s.delayedMsgPreimages.Size())
	}
	return preimage, nil
}

// PourDelayedInboxToOutbox moves all items from the inbox to the outbox,
// reversing their order so that the first-seen message is popped first from the outbox.
// This implements the "pour" operation of the two-stack FIFO queue.
// The caller must ensure the outbox is empty before calling. The number of items to
// pour is DelayedMessagesSeen - DelayedMessagesRead.
//
// Replay-mode counterpart: mel-replay/delayed_message_db.go pourInboxToOutbox.
// Both must produce identical accumulator state transitions for fraud proof correctness.
func (s *State) PourDelayedInboxToOutbox() error {
	if s.DelayedMessageOutboxAcc != (common.Hash{}) {
		return errors.New("PourDelayedInboxToOutbox: outbox must be empty before pouring")
	}
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
	// Pop from inbox (LIFO: last-seen out first), push each onto outbox. First-seen is
	// pushed last, landing on top of the outbox (LIFO), restoring FIFO order.
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
		newAcc, preimage := HashChainLink(s.DelayedMessageOutboxAcc, msgHash)
		s.DelayedMessageOutboxAcc = newAcc
		if err := s.recordDelayedChainLink(newAcc, preimage); err != nil {
			return err
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

// ensureKeccakPreimagesMap validates a PreimagesMap is non-nil and ensures
// the Keccak256 sub-map exists.
func ensureKeccakPreimagesMap(preimagesMap daprovider.PreimagesMap) error {
	if preimagesMap == nil {
		return errors.New("preimages recording destination cannot be nil")
	}
	if _, ok := preimagesMap[arbutil.Keccak256PreimageType]; !ok {
		preimagesMap[arbutil.Keccak256PreimageType] = make(map[common.Hash][]byte)
	}
	return nil
}

// RecordMsgPreimagesTo initializes the state's msgPreimagesDest to record preimages
// related to the extracted L2 messages needed for MEL validation into the given preimages map.
// When set, AccumulateMessage will record accumulator chain and message content preimages.
func (s *State) RecordMsgPreimagesTo(preimagesMap daprovider.PreimagesMap) error {
	if err := ensureKeccakPreimagesMap(preimagesMap); err != nil {
		return err
	}
	s.msgPreimagesDest = preimagesMap
	return nil
}

// RecordDelayedMsgPreimagesTo initializes the state's delayedMsgPreimagesDest to record
// preimages related to delayed messages needed for MEL validation into the given preimages map.
// When set, AccumulateDelayedMessage records accumulator chain and message content preimages,
// whereas PourDelayedInboxToOutbox records only accumulator chain preimages.
func (s *State) RecordDelayedMsgPreimagesTo(preimagesMap daprovider.PreimagesMap) error {
	if err := ensureKeccakPreimagesMap(preimagesMap); err != nil {
		return err
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

// findPivot returns the index boundary between outbox and inbox messages within
// the unread range. Messages at indices [0..pivot) are in the outbox;
// [pivot..totalUnread) are in the inbox. Returns -1 if no valid partition is found.
func (s *State) findPivot(totalUnread uint64, msgHashes []common.Hash) int {
	if s.DelayedMessageOutboxAcc == (common.Hash{}) {
		return 0
	}
	if s.DelayedMessageInboxAcc == (common.Hash{}) {
		// #nosec G115
		return int(totalUnread)
	}
	// Brute-force O(N^2) search: iterate candidate pivots from totalUnread-1 down to 0.
	// Each pivot cp splits the range into inbox [cp..totalUnread) and outbox [0..cp).
	// Starting with the largest cp (smallest inbox) is an optimization for the common
	// case where few messages remain in the inbox after the last pour.
	// Use signed int to avoid unsigned underflow when totalUnread == 1.
	for cp := int(totalUnread) - 1; cp >= 0; cp-- { //nolint:gosec
		acc := common.Hash{}
		for i := uint64(cp); i < totalUnread; i++ {
			acc = HashChainLinkHash(acc, msgHashes[i])
		}
		if acc == s.DelayedMessageInboxAcc {
			return cp
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
		newAcc, preimage := HashChainLink(acc, msgHashes[i])
		if err := s.recordDelayedChainLink(newAcc, preimage); err != nil {
			return common.Hash{}, err
		}
		if err := s.recordDelayedContent(msgHashes[i], msgBytesArr[i]); err != nil {
			return common.Hash{}, err
		}
		acc = newAcc
	}
	return acc, nil
}

// RebuildDelayedMsgPreimages reconstructs the in-memory preimage cache from
// delayed messages stored in the database. This is needed after loading state
// from DB (where the cache is nil), after reorgs, and for recovery from
// preimage cache misses.
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
	// Use 2x capacity to leave headroom for post-rebuild accumulations before
	// the next pour, preventing eviction of needed preimages.
	// #nosec G115
	s.delayedMsgPreimages = containers.NewLruCache[common.Hash, []byte](max(2*int(totalUnread), 64))
	msgHashes, msgBytesArr, err := s.fetchAndHashUnreadMessages(totalUnread, fetchDelayedMsg)
	if err != nil {
		return err
	}
	pivot := s.findPivot(totalUnread, msgHashes)
	if pivot < 0 {
		return fmt.Errorf("failed to find pivot between inbox and outbox: totalUnread=%d, delayedRead=%d, delayedSeen=%d, inboxAcc=%s, outboxAcc=%s",
			totalUnread, s.DelayedMessagesRead, s.DelayedMessagesSeen, s.DelayedMessageInboxAcc.Hex(), s.DelayedMessageOutboxAcc.Hex())
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
