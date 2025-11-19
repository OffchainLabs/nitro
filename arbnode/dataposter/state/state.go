package state

import (
	"context"
	"math/big"
	"sync"

	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
)

// lint:require-exhaustive-initialization
type InternalState struct {
	mutex       sync.Mutex
	lockedState LockedInternalState
}

func NewInternalState(queue QueueStorage) *InternalState {
	return &InternalState{
		mutex: sync.Mutex{},
		lockedState: LockedInternalState{
			LastBlock:  big.NewInt(0),
			Balance:    big.NewInt(0),
			Nonce:      0,
			Queue:      queue,
			ErrorCount: make(map[uint64]int),
		},
	}
}

func (s *InternalState) Lock() *LockedInternalState {
	s.mutex.Lock()
	return &s.lockedState
}

func (s *InternalState) Unlock() {
	s.mutex.Unlock()
}

// lint:require-exhaustive-initialization
type LockedInternalState struct {
	LastBlock  *big.Int
	Balance    *big.Int
	Nonce      uint64
	Queue      QueueStorage
	ErrorCount map[uint64]int // number of consecutive intermittent errors rbf-ing or sending, per nonce
}

// QueueStorage implements queue-alike storage that can
// - Insert item at specified index
// - Update item with the condition that existing value equals assumed value
// - Delete all the items up to specified index (prune)
// - Calculate length
// Note: one of the implementation of this interface (Redis storage) does not
// support duplicate values.
type QueueStorage interface {
	// FetchContents returns at most maxResults items starting from specified index.
	FetchContents(ctx context.Context, startingIndex uint64, maxResults uint64) ([]*storage.QueuedTransaction, error)
	// Get returns the item at index, or nil if not found.
	Get(ctx context.Context, index uint64) (*storage.QueuedTransaction, error)
	// FetchLast returns item with the biggest index, or nil if the queue is empty.
	FetchLast(ctx context.Context) (*storage.QueuedTransaction, error)
	// Prune prunes items up to (excluding) specified index.
	Prune(ctx context.Context, until uint64) error
	// Put inserts new item at specified index if previous value matches specified value.
	Put(ctx context.Context, index uint64, prevItem, newItem *storage.QueuedTransaction) error
	// Length returns the size of a queue.
	Length(ctx context.Context) (int, error)
	// IsPersistent indicates whether queue stored at disk.
	IsPersistent() bool
}
