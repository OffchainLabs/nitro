// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"

	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// Default parameters for HashedAddressChecker, used in NewDefaultHashedAddressChecker
const (
	defaultRestrictedAddrWorkerCount = 4
	defaultRestrictedAddrQueueSize   = 8192
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

func NewDefaultHashedAddressChecker(store *HashStore) *HashedAddressChecker {
	return NewHashedAddressChecker(
		store,
		defaultRestrictedAddrWorkerCount,
		defaultRestrictedAddrQueueSize,
	)
}

func (c *HashedAddressChecker) NewTxState() state.AddressCheckerState {
	return &HashedAddressCheckerState{
		checker: c,
	}
}

func (c *HashedAddressChecker) processAddress(addr common.Address, state *HashedAddressCheckerState) {
	restricted := c.store.IsRestricted(addr)
	state.report(restricted)
}

// worker runs for the lifetime of the checker; workChan is never closed.
func (c *HashedAddressChecker) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-c.workChan:
			c.processAddress(item.addr, item.state)
		}
	}
}

func (s *HashedAddressCheckerState) TouchAddress(addr common.Address) {
	s.pending.Add(1)

	// If the checker is stopped, process synchronously
	if s.checker.Stopped() {
		s.checker.processAddress(addr, s)
		return
	}

	select {
	case s.checker.workChan <- workItem{addr: addr, state: s}:
		// ok
	case <-s.checker.GetContext().Done():
		// shutting down, canceling worker
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
