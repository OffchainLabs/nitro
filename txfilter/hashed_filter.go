// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package txfilter

import (
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
)

// HashedAddressChecker is a global, shared address checker that filters
// transactions by comparing hashed addresses against a precomputed hash list.
//
// Hashing is treated as expensive and amortised across all transactions via
// a shared LRU cache. The checker itself is stateless from the StateDB
// perspective; all per-transaction bookkeeping lives in HashedAddressCheckerState.
type HashedAddressChecker struct {
	filteredHashSet map[common.Hash]struct{}
	hashCache       *lru.Cache[common.Address, common.Hash]
	salt            []byte

	workChan chan workItem
}

// HashedAddressCheckerState tracks address filtering for a single transaction.
// It aggregates asynchronous hash checks initiated by TouchAddress and blocks
// in IsFiltered until all submitted checks complete.
type HashedAddressCheckerState struct {
	checker *HashedAddressChecker

	// filtered is set to true if any checked address hash appears in filtered HashSet.
	filtered atomic.Bool

	// pending tracks the number of outstanding hash checks for this transaction.
	pending sync.WaitGroup
}

// workItem is a helper struct representing a single address hashing request associated with
// a specific transaction state.
type workItem struct {
	addr  common.Address
	state *HashedAddressCheckerState
}

// NewHashedAddressChecker constructs a new checker for a given hash list.
// The hash list is copied into an immutable set.
func NewHashedAddressChecker(
	hashes []common.Hash,
	salt []byte,
	hashCacheSize int,
	workerCount int,
	queueSize int,
) *HashedAddressChecker {
	hashSet := make(map[common.Hash]struct{}, len(hashes))
	for _, h := range hashes {
		hashSet[h] = struct{}{}
	}

	cache := lru.NewCache[common.Address, common.Hash](hashCacheSize)

	c := &HashedAddressChecker{
		filteredHashSet: hashSet,
		hashCache:       cache,
		salt:            salt,
		workChan:        make(chan workItem, queueSize),
	}

	for i := 0; i < workerCount; i++ {
		go c.worker()
	}

	return c
}

func (c *HashedAddressChecker) NewTxState() state.AddressCheckerState {
	return &HashedAddressCheckerState{
		checker: c,
	}
}

// worker runs for the lifetime of the checker; workCh is never closed.
func (c *HashedAddressChecker) worker() {
	for item := range c.workChan {
		// First, check the LRU cache for a precomputed hash.
		hash, ok := c.hashCache.Get(item.addr)
		if !ok {
			hash = crypto.Keccak256Hash(item.addr.Bytes(), c.salt)
			c.hashCache.Add(item.addr, hash)
		}

		// Second, check if the computed hash is in the filtered set.
		_, filtered := c.filteredHashSet[hash]
		item.state.report(filtered)
	}
}

func (s *HashedAddressCheckerState) TouchAddress(addr common.Address) {
	select {
	case s.checker.workChan <- workItem{addr: addr, state: s}:
		s.pending.Add(1)
	default:
		// queue full: drop work conservatively
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
