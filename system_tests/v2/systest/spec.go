// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"time"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/util/containers"
)

// Scenario is a test function. Fails the test via env assertions.
type Scenario func(*Env)

// Hook runs after the scenario completed without a fatal assertion (runtime.Goexit).
// Returns an error which the runner reports via t.Errorf, so siblings still
// attempt and failures aggregate at the end of the run. Scenario failures
// (Goexit) skip all hooks. Distinct from Scenario: the type system
// prevents passing a Scenario where a Hook is expected (and vice versa).
type Hook func(*Env) error

// weight is how many scheduler slots a scenario consumes, derived from its
// topology and capabilities (see specWeight). Never set directly.
type weight int

const (
	weightLight  weight = iota + 1 // L2-only, single node
	weightMedium                   // L1 + L2
	weightHeavy                    // multi-node
	weightMax                      // full stack, or any validating run
)

// specWeight projects a topology onto a scheduler-slot cost.
func specWeight(topology Topology) weight {
	switch topology {
	case TopologyL1L2:
		return weightMedium
	case TopologyMultiNode:
		return weightHeavy
	case TopologyFullStack:
		return weightMax
	}
	return weightLight
}

// StateScheme is the geth trie storage backend.
type StateScheme string

const (
	StateSchemeHash StateScheme = "hash"
	StateSchemePath StateScheme = "path"
)

// Valid reports whether s is a known state scheme.
func (s StateScheme) Valid() bool {
	switch s {
	case StateSchemeHash, StateSchemePath:
		return true
	}
	return false
}

// validationScheme is the only state scheme block validation supports.
const validationScheme = StateSchemeHash

// DBEngine is the geth chain-data persistence backend.
type DBEngine string

const (
	DBEnginePebble   DBEngine = "pebble"
	DBEngineLevelDB  DBEngine = "leveldb"
	DBEngineInMemory DBEngine = "in-memory"
)

// Valid reports whether e is a known db engine.
func (e DBEngine) Valid() bool {
	switch e {
	case DBEnginePebble, DBEngineLevelDB, DBEngineInMemory:
		return true
	}
	return false
}

// Topology selects which node layout buildNode constructs.
type Topology int

const (
	TopologyL2Only    Topology = iota // L2-only sequencer, no parent chain
	TopologyL1L2                      // L1 + sequencer L2 (batch posting + inbox reading)
	TopologyMultiNode                 // L1 + sequencer L2 + non-sequencer follower L2
	TopologyFullStack                 // L1 + sequencer L2 + follower L2 running block validation + staker
)

// Spec is the resolved per-variant config. Scenarios receive it by value on
// Env.Spec, so any mutation is local to that scenario and affects nothing else.
type Spec struct {
	Name         string
	Weight       weight
	ArbOSVersion containers.Option[uint64]
	StateScheme  containers.Option[StateScheme]
	DBEngine     containers.Option[DBEngine]
	Category     string
	Topology     Topology
	// Timeout is the per-scenario wall-clock backstop. 0 = use the runner default.
	Timeout time.Duration
	// SkipChainOwner skips making the Owner account a chain owner during setup.
	SkipChainOwner bool
	// ExposeRPC exposes HTTP+WS on the L2 stack, populating L2Handle.HTTPClient/WSClient
	// (Off by default — tests use the in-process Client).
	ExposeRPC bool
	// arbOSInit seeds ArbOS init params into the L2 genesis. Nil = defaults.
	arbOSInit *params.ArbOSInit
	// Validate requests block validation (JIT) for this node.
	Validate bool
}
