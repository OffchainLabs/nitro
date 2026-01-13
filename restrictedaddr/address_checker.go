// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package restrictedaddr

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

// RestrictedAddressChecker implements state.AddressChecker interface.
// It checks addresses against the HashStore.
type RestrictedAddressChecker struct {
	store *HashStore
}

func NewRestrictedAddressChecker(store *HashStore) *RestrictedAddressChecker {
	return &RestrictedAddressChecker{store: store}
}

// NewTxState creates fresh state for a new transaction.
// Each transaction gets its own state to track whether any restricted address was touched.
func (c *RestrictedAddressChecker) NewTxState() state.AddressCheckerState {
	return &restrictedAddrState{checker: c}
}

// restrictedAddrState tracks restricted address access for a single transaction.
type restrictedAddrState struct {
	checker       *RestrictedAddressChecker
	filtered      bool
	pendingChecks sync.WaitGroup
}

// TouchAddress records an address access and checks if it should be filtered.
func (s *restrictedAddrState) TouchAddress(addr common.Address) {
	s.pendingChecks.Add(1)
	go func() {
		defer s.pendingChecks.Done()
		if s.checker.store.IsRestricted(addr) {
			s.filtered = true
		}
	}()
}

// IsFiltered returns whether any touched address was filtered.
func (s *restrictedAddrState) IsFiltered() bool {
	s.pendingChecks.Wait()
	return s.filtered
}
