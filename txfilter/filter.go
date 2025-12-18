// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package txfilter

import "github.com/ethereum/go-ethereum/common"

// AddressFilter checks if addresses should be filtered from transactions.
type AddressFilter interface {
	// IsFiltered returns true if the address should be filtered
	IsFiltered(addr common.Address) bool
	// Enabled returns true if filtering is active
	Enabled() bool
}

// NoopFilter is a stub that filters nothing.
type NoopFilter struct{}

func (f *NoopFilter) IsFiltered(addr common.Address) bool { return false }
func (f *NoopFilter) Enabled() bool                       { return false }

// StaticFilter filters a fixed set of addresses (for testing).
type StaticFilter struct {
	addresses map[common.Address]struct{}
}

func NewStaticFilter(addrs []common.Address) *StaticFilter {
	m := make(map[common.Address]struct{}, len(addrs))
	for _, addr := range addrs {
		m[addr] = struct{}{}
	}
	return &StaticFilter{addresses: m}
}

func (f *StaticFilter) IsFiltered(addr common.Address) bool {
	_, ok := f.addresses[addr]
	return ok
}

func (f *StaticFilter) Enabled() bool {
	return len(f.addresses) > 0
}
