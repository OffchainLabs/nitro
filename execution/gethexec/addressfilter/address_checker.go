// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/arbitrum/filter"
	"github.com/ethereum/go-ethereum/core/state"

	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// HashedAddressChecker is a global, shared address checker that filters
// transactions using a HashStore. Hashing and caching are delegated to
// the HashStore; this checker only manages async execution and per-tx
// aggregation.
type HashedAddressChecker struct {
	stopwaiter.StopWaiter
	store       *HashStore
	workChan    chan workItem
	workerCount int
}

// HashedAddressCheckerState tracks address filtering for a single transaction.
// It aggregates asynchronous checks initiated by TouchAddress and blocks
// in IsFiltered until all submitted checks complete.
type HashedAddressCheckerState struct {
	checker           *HashedAddressChecker
	mu                sync.Mutex
	filtered          bool
	filteredAddresses []filter.FilteredAddressRecord
	pending           sync.WaitGroup
}

type workItem struct {
	record *filter.FilteredAddressRecord
	state  *HashedAddressCheckerState
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
		store:       store,
		workChan:    make(chan workItem, queueSize),
		workerCount: workerCount,
	}

	return c
}

func (c *HashedAddressChecker) Start(ctx context.Context) {
	c.StopWaiter.Start(ctx, c)

	for i := 0; i < c.workerCount; i++ {
		c.LaunchThread(func(ctx context.Context) {
			c.worker(ctx)
		})
	}
}

func (c *HashedAddressChecker) NewTxState() state.AddressCheckerState {
	return &HashedAddressCheckerState{
		checker: c,
	}
}

func (c *HashedAddressChecker) FilterSetID() uuid.UUID {
	return c.store.ID()
}

func (c *HashedAddressChecker) processRecord(record *filter.FilteredAddressRecord, state *HashedAddressCheckerState) {
	restricted, filterSetID := c.store.IsRestricted(record.Address)
	// Override with the ID from the same snapshot as the restriction check.
	record.FilterSetID = filterSetID
	state.report(record, restricted)
}

// worker runs for the lifetime of the checker; workChan is never closed.
func (c *HashedAddressChecker) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-c.workChan:
			c.processRecord(item.record, item.state)
		}
	}
}

func (s *HashedAddressCheckerState) TouchAddress(record *filter.FilteredAddressRecord) {
	s.pending.Add(1)

	// If the checker is stopped, conservatively mark filtered
	if s.checker.Stopped() {
		s.report(nil, true)
		return
	}

	select {
	case s.checker.workChan <- workItem{record: record, state: s}:
		// ok
	case <-s.checker.GetContext().Done():
		// shutting down, conservatively mark filtered
		s.report(nil, true)
	}
}

func (s *HashedAddressCheckerState) report(record *filter.FilteredAddressRecord, filtered bool) {
	if filtered {
		s.mu.Lock()
		s.filtered = true
		if record != nil {
			s.filteredAddresses = append(s.filteredAddresses, *record)
		}
		s.mu.Unlock()
	}
	s.pending.Done()
}

func (s *HashedAddressCheckerState) IsFiltered() (bool, []filter.FilteredAddressRecord) {
	s.pending.Wait()
	return s.filtered, s.filteredAddresses
}
