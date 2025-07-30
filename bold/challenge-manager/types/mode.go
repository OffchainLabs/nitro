// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package types

type Mode uint8

const (
	// Watchtower: don't do anything on L1, but log if there's a bad assertion
	WatchTowerMode Mode = iota
	// Defensive: stake if there's a bad assertion
	DefensiveMode
	// Resolve nodes: stay staked on the latest node and resolve any unconfirmed
	// nodes, challenging bad assertions
	ResolveMode
	// Make nodes: continually create new nodes, challenging bad assertions
	MakeMode
)

// SupportsStaking returns true if the mode supports staking
func (m Mode) SupportsStaking() bool {
	return m >= MakeMode
}

// SupportsPostingRivals returns true if the mode supports posting rival
// assertions.
func (m Mode) SupportsPostingRivals() bool {
	return m >= DefensiveMode
}

// SupportsPostingChallenges returns true if the mode supports posting
// challenging edges.
func (m Mode) SupportsPostingChallenges() bool {
	return m > DefensiveMode
}
