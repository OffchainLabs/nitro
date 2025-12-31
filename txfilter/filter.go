// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package txfilter

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

// NoopChecker is a stub that filters nothing.
type NoopChecker struct{}

func (c *NoopChecker) NewTxState() state.AddressCheckerState {
	return &noopState{}
}

type noopState struct{}

func (s *noopState) TouchAddress(addr common.Address) {}
func (s *noopState) IsFiltered() bool                 { return false }

// StaticAsyncChecker filters a fixed set of addresses (for testing).
// Checks addresses asynchronously using goroutines to demonstrate the async pattern.
type StaticAsyncChecker struct {
	addresses map[common.Address]struct{}
}

func NewStaticAsyncChecker(addrs []common.Address) *StaticAsyncChecker {
	m := make(map[common.Address]struct{}, len(addrs))
	for _, addr := range addrs {
		m[addr] = struct{}{}
	}
	return &StaticAsyncChecker{addresses: m}
}

func (c *StaticAsyncChecker) NewTxState() state.AddressCheckerState {
	return &staticAsyncState{checker: c}
}

type staticAsyncState struct {
	checker       *StaticAsyncChecker
	filtered      bool
	pendingChecks sync.WaitGroup
}

func (s *staticAsyncState) TouchAddress(addr common.Address) {
	s.pendingChecks.Add(1)
	go func() {
		defer s.pendingChecks.Done()
		if _, found := s.checker.addresses[addr]; found {
			s.filtered = true
		}
	}()
}

func (s *staticAsyncState) IsFiltered() bool {
	s.pendingChecks.Wait()
	return s.filtered
}
