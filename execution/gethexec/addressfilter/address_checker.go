// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"context"
	"sync"

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
	addr  filter.FilteredAddressWithReason
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

func (c *HashedAddressChecker) FilterSetID() string {
	return c.store.ID().String()
}

func (c *HashedAddressChecker) processItem(item workItem) {
	defer item.state.pending.Done()
	restricted, filterSetID := c.store.IsRestricted(item.addr.Address)
	if restricted {
		record := filter.FilteredAddressRecord{
			FilterSetID:               filterSetID.String(),
			FilteredAddressWithReason: item.addr,
		}
		item.state.report(&record)
	}
}

// worker runs for the lifetime of the checker; workChan is never closed.
func (c *HashedAddressChecker) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-c.workChan:
			c.processItem(item)
		}
	}
}

func (s *HashedAddressCheckerState) TouchAddress(touched filter.FilteredAddressWithReason) {
	s.pending.Add(1)

	// If the checker is stopped, conservatively mark filtered
	if s.checker.Stopped() {
		record := filter.FilteredAddressRecord{
			FilterSetID:               s.checker.FilterSetID(),
			FilteredAddressWithReason: touched,
		}
		s.report(&record)
		s.pending.Done()
		return
	}

	select {
	case s.checker.workChan <- workItem{addr: touched, state: s}:
		// ok
	case <-s.checker.GetContext().Done():
		// shutting down, conservatively mark filtered
		record := filter.FilteredAddressRecord{
			FilterSetID:               s.checker.FilterSetID(),
			FilteredAddressWithReason: touched,
		}
		s.report(&record)
		s.pending.Done()
	}
}

// report records a filtered address. Called when the address is confirmed
// restricted, or conservatively during shutdown when restriction cannot be verified.
func (s *HashedAddressCheckerState) report(record *filter.FilteredAddressRecord) {
	s.mu.Lock()
	s.filtered = true
	s.filteredAddresses = append(s.filteredAddresses, *record)
	s.mu.Unlock()
}

func (s *HashedAddressCheckerState) IsFiltered() (bool, []filter.FilteredAddressRecord) {
	s.pending.Wait()
	return s.filtered, s.filteredAddresses
}
