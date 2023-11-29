package types

type Mode uint8

const (
	// Watchtower: don't do anything on L1, but log if there's a bad assertion
	WatchTowerMode Mode = iota
	// Defensive: stake if there's a bad assertion
	DefensiveMode
	// Resolve nodes: stay staked on the latest node and resolve any unconfirmed nodes, challenging bad assertions
	ResolveMode
	// Make nodes: continually create new nodes, challenging bad assertions
	MakeMode
)
