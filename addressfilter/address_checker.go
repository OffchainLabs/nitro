// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

// Default parameters for HashedAddressChecker, used in NewDefaultHashedAddressChecker
const (
	restrictedAddrWorkerCount = 4
	restrictedAddrQueueSize   = 1024
)

// HashedAddressChecker is a global, shared address checker that filters
// transactions using a HashStore. Hashing and caching are delegated to
// the HashStore; this checker only manages async execution and per-tx
// aggregation.
type HashedAddressChecker struct {
	store    *HashStore
	workChan chan workItem
}

// HashedAddressCheckerState tracks address filtering for a single transaction.
// It aggregates asynchronous checks initiated by TouchAddress and blocks
// in IsFiltered until all submitted checks complete.
type HashedAddressCheckerState struct {
	checker  *HashedAddressChecker
	filtered atomic.Bool
	pending  sync.WaitGroup
}

type workItem struct {
	addr  common.Address
	state *HashedAddressCheckerState
}

// NewHashedAddressChecker constructs a new checker backed by a HashStore.
func NewHashedAddressChecker(
	store *HashStore,
	workerCount int,
	queueSize int,
) *HashedAddressChecker {
	if store == nil {
		panic("HashStore cannot be nil")
	}

	c := &HashedAddressChecker{
		store:    store,
		workChan: make(chan workItem, queueSize),
	}

	for range workerCount {
		go c.worker()
	}

	return c
}

func NewDefaultHashedAddressChecker(store *HashStore) *HashedAddressChecker {
	return NewHashedAddressChecker(
		store,
		restrictedAddrWorkerCount,
		restrictedAddrQueueSize,
	)
}

func (c *HashedAddressChecker) NewTxState() state.AddressCheckerState {
	return &HashedAddressCheckerState{
		checker: c,
	}
}

// worker runs for the lifetime of the checker; workChan is never closed.
func (c *HashedAddressChecker) worker() {
	for item := range c.workChan {
		restricted := c.store.IsRestricted(item.addr)
		item.state.report(restricted)
	}
}

func (s *HashedAddressCheckerState) TouchAddress(addr common.Address) {
	s.pending.Add(1)
	select {
	case s.checker.workChan <- workItem{addr: addr, state: s}:
		// ok
	default:
		// queue full: drop check (fail-open, false negative possible)
		s.pending.Done()
	}
}

func (s *HashedAddressCheckerState) report(filtered bool) {
	if filtered {
		s.filtered.Store(true)
	}
	s.pending.Done()
}

func (s *HashedAddressCheckerState) IsFiltered() bool {
	s.pending.Wait()
	return s.filtered.Load()
}
