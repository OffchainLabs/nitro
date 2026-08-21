// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/containers"
)

// TestOption mutates a fresh internal builder during Test. Functions
// (not values) so they compose via Compose() and form preset vars.
type TestOption func(*builder)

// Named overrides the reflection-derived test name. Use when the same scenario
// is registered with multiple configurations. Panics if called twice on the
// same builder.
func Named(name string) TestOption {
	return func(b *builder) {
		if name == "" || strings.Contains(name, "/") {
			panic(fmt.Sprintf("systest: Named(%q): name must be non-empty and must not contain '/'", name))
		}
		if b.name != "" {
			panic(fmt.Sprintf("systest: Named applied twice (existing=%q, new=%q)", b.name, name))
		}
		b.name = name
	}
}

// WithArbOS pins ArbOS version. Conflicts with a scheduleParams ArbOS pin cause skip. Panics
// if applied twice or if combined with MatrixArbOS on the same axis.
func WithArbOS(v uint64) TestOption {
	return func(b *builder) {
		if v == 0 {
			panic("systest: WithArbOS version must be positive")
		}
		if b.arbOS.IsSome() {
			panic(fmt.Sprintf("systest: WithArbOS applied twice (existing=%d, new=%d)", b.arbOS.Unwrap(), v))
		}
		if _, ok := b.dims[axisArbOS]; ok {
			panic(fmt.Sprintf("systest: WithArbOS(%d) conflicts with MatrixArbOS on same axis", v))
		}
		b.arbOS = containers.Some(v)
	}
}

// WithStateScheme pins this test to a specific state scheme.
func WithStateScheme(s StateScheme) TestOption {
	return func(b *builder) {
		if !s.Valid() {
			panic(fmt.Sprintf("systest: WithStateScheme invalid scheme %q", s))
		}
		if b.stateScheme.IsSome() {
			panic(fmt.Sprintf("systest: WithStateScheme applied twice (existing=%q, new=%q)", b.stateScheme.Unwrap(), s))
		}
		if _, ok := b.dims[axisStateScheme]; ok {
			panic(fmt.Sprintf("systest: WithStateScheme(%q) conflicts with MatrixStateScheme on same axis", s))
		}
		b.stateScheme = containers.Some(s)
	}
}

// WithDBEngine pins this test to a specific db engine.
func WithDBEngine(e DBEngine) TestOption {
	return func(b *builder) {
		if !e.Valid() {
			panic(fmt.Sprintf("systest: WithDBEngine invalid engine %q", e))
		}
		if b.dbEngine.IsSome() {
			panic(fmt.Sprintf("systest: WithDBEngine applied twice (existing=%q, new=%q)", b.dbEngine.Unwrap(), e))
		}
		if _, ok := b.dims[axisDBEngine]; ok {
			panic(fmt.Sprintf("systest: WithDBEngine(%q) conflicts with MatrixDBEngine on same axis", e))
		}
		b.dbEngine = containers.Some(e)
	}
}

// MatrixArbOS declares a matrix axis over ArbOS versions. The runner expands
// one scheduled test per supplied value. Panics if no values are given, if
// combined with WithArbOS pin, or if applied twice.
func MatrixArbOS(vs ...uint64) TestOption {
	if len(vs) == 0 {
		panic("systest: MatrixArbOS requires at least one value")
	}
	seen := map[uint64]bool{}
	for _, v := range vs {
		if v == 0 {
			panic("systest: MatrixArbOS versions must be positive")
		}
		if seen[v] {
			panic(fmt.Sprintf("systest: MatrixArbOS has duplicate value %d", v))
		}
		seen[v] = true
	}
	variants := mkArbOSAxis(vs)
	return func(b *builder) {
		if b.arbOS.IsSome() {
			panic(fmt.Sprintf("systest: MatrixArbOS(...) conflicts with WithArbOS(%d) pin on same axis", b.arbOS.Unwrap()))
		}
		if _, ok := b.dims[axisArbOS]; ok {
			panic("systest: MatrixArbOS(...) applied twice")
		}
		b.dims[axisArbOS] = variants
	}
}

// MatrixStateScheme declares a matrix axis over state schemes.
// Panics if no values are given, if combined with WithStateScheme pin, or if
// applied twice.
func MatrixStateScheme(ss ...StateScheme) TestOption {
	if len(ss) == 0 {
		panic("systest: MatrixStateScheme requires at least one value")
	}
	seen := map[StateScheme]bool{}
	for _, s := range ss {
		if !s.Valid() {
			panic(fmt.Sprintf("systest: MatrixStateScheme invalid scheme %q", s))
		}
		if seen[s] {
			panic(fmt.Sprintf("systest: MatrixStateScheme has duplicate value %q", s))
		}
		seen[s] = true
	}
	variants := mkStateSchemeAxis(ss)
	return func(b *builder) {
		if b.stateScheme.IsSome() {
			panic(fmt.Sprintf("systest: MatrixStateScheme(...) conflicts with WithStateScheme(%q) pin on same axis", b.stateScheme.Unwrap()))
		}
		if _, ok := b.dims[axisStateScheme]; ok {
			panic("systest: MatrixStateScheme(...) applied twice")
		}
		b.dims[axisStateScheme] = variants
	}
}

// MatrixDBEngine declares a matrix axis over db engines.
// Panics if no values are given, if combined with WithDBEngine pin, or if
// applied twice.
func MatrixDBEngine(es ...DBEngine) TestOption {
	if len(es) == 0 {
		panic("systest: MatrixDBEngine requires at least one value")
	}
	seen := map[DBEngine]bool{}
	for _, e := range es {
		if !e.Valid() {
			panic(fmt.Sprintf("systest: MatrixDBEngine invalid engine %q", e))
		}
		if seen[e] {
			panic(fmt.Sprintf("systest: MatrixDBEngine has duplicate value %q", e))
		}
		seen[e] = true
	}
	variants := mkDBEngineAxis(es)
	return func(b *builder) {
		if b.dbEngine.IsSome() {
			panic(fmt.Sprintf("systest: MatrixDBEngine(...) conflicts with WithDBEngine(%q) pin on same axis", b.dbEngine.Unwrap()))
		}
		if _, ok := b.dims[axisDBEngine]; ok {
			panic("systest: MatrixDBEngine(...) applied twice")
		}
		b.dims[axisDBEngine] = variants
	}
}

// WithCategory tags this test with a named category; untagged tests are in the
// default one. Tests run only when their category is enabled.
// Panics if applied twice.
func WithCategory(c string) TestOption {
	return func(b *builder) {
		if c == "" || c == defaultCategory {
			panic(fmt.Sprintf("systest: WithCategory(%q): the default category is implicit", c))
		}
		if b.category != defaultCategory {
			panic(fmt.Sprintf("systest: WithCategory applied twice (existing=%q, new=%q)", b.category, c))
		}
		b.category = c
	}
}

// MinArbOS sets the minimum ArbOS version this test supports. Panics if zero
// or applied twice.
func MinArbOS(v uint64) TestOption {
	return func(b *builder) {
		if v == 0 {
			panic("systest: MinArbOS version must be positive")
		}
		if b.minArbOS != 0 {
			panic(fmt.Sprintf("systest: MinArbOS applied twice (existing=%d, new=%d)", b.minArbOS, v))
		}
		b.minArbOS = v
	}
}

// MaxArbOS sets the maximum ArbOS version this test supports. Panics if zero
// or applied twice.
func MaxArbOS(v uint64) TestOption {
	return func(b *builder) {
		if v == 0 {
			panic("systest: MaxArbOS version must be positive")
		}
		if b.maxArbOS != 0 {
			panic(fmt.Sprintf("systest: MaxArbOS applied twice (existing=%d, new=%d)", b.maxArbOS, v))
		}
		b.maxArbOS = v
	}
}

// WithTimeout overrides the per-scenario wall-clock backstop
// for this test. Panics if non-positive or applied twice.
func WithTimeout(d time.Duration) TestOption {
	return func(b *builder) {
		if d <= 0 {
			panic(fmt.Sprintf("systest: WithTimeout(%v) must be positive", d))
		}
		if b.timeout != 0 {
			panic(fmt.Sprintf("systest: WithTimeout applied twice (existing=%v, new=%v)", b.timeout, d))
		}
		b.timeout = d
	}
}

// SkipOnStateSchemes skips the test when its resolved state scheme matches any
// listed scheme. Panics on an invalid scheme.
func SkipOnStateSchemes(ss ...StateScheme) TestOption {
	return func(b *builder) {
		for _, s := range ss {
			if !s.Valid() {
				panic(fmt.Sprintf("systest: SkipOnStateSchemes invalid scheme %q", s))
			}
			b.skipStateSchemes = append(b.skipStateSchemes, s)
		}
	}
}

// SkipOnRace skips the test when the binary is built with the race detector,
// for tests too timing-sensitive or slow to pass reliably under -race.
func SkipOnRace() TestOption {
	return func(b *builder) { b.skipOnRace = true }
}

// WithArbOSInit seeds ArbOS init params into the L2 genesis.
func WithArbOSInit(init params.ArbOSInit) TestOption {
	return func(b *builder) {
		if b.arbOSInit != nil {
			panic("systest: WithArbOSInit applied twice")
		}
		b.arbOSInit = &init
	}
}

// WithoutChainOwner skips the default setup step that makes the Owner account a
// chain owner. For tests that need a pristine chain. Panics if applied twice.
func WithoutChainOwner() TestOption {
	return func(b *builder) {
		if b.skipChainOwner {
			panic("systest: WithoutChainOwner applied twice")
		}
		b.skipChainOwner = true
	}
}

// WithRPCEndpoints exposes HTTP+WS on the L2 stack so tests can dial
// L2Handle.HTTPClient / WSClient. Off by default — tests use the in-process Client.
func WithRPCEndpoints() TestOption {
	return func(b *builder) { b.exposeRPC = true }
}

// WithPostHook appends a hook run after the scenario completes without a fatal
// assertion. Hook errors fail the test via t.Errorf so all hooks attempt before exit.
func WithPostHook(h Hook) TestOption {
	return func(b *builder) { b.postHooks = append(b.postHooks, h) }
}

// WithInitDataOverride registers a function that mutates the L2 genesis init data
// (accounts, contracts, balances) before the chain is built. Multiple accumulate;
// each runs once per L2 node per matrix cell, concurrently across cells — keep it stateless.
func WithInitDataOverride(f func(*statetransfer.ArbosInitializationInfo)) TestOption {
	return func(b *builder) { b.initDataOverrides = append(b.initDataOverrides, f) }
}

// WithExecConfigOverride registers a function that mutates the execution config
// of every L2 node built, after defaults are applied and before the config is
// frozen. Escape hatch for v1-style raw poking. Multiple accumulate; each runs
// once per L2 node per matrix cell, concurrently across cells — keep it stateless.
func WithExecConfigOverride(f func(*gethexec.Config)) TestOption {
	return func(b *builder) { b.execOverrides = append(b.execOverrides, f) }
}

// WithNodeConfigOverride mutates the consensus (arbnode) config of every L2
// node built. Same contract as WithExecConfigOverride.
func WithNodeConfigOverride(f func(*arbnode.Config)) TestOption {
	return func(b *builder) { b.nodeOverrides = append(b.nodeOverrides, f) }
}

// WithStackConfigOverride mutates the geth stack config (HTTP, WS, P2P,
// DB engine, …) of every L2 node built. Same contract as WithExecConfigOverride.
func WithStackConfigOverride(f func(*node.Config)) TestOption {
	return func(b *builder) { b.stackOverrides = append(b.stackOverrides, f) }
}

// WithChainConfigOverride mutates the L2 chain config (ArbOS chain params,
// code-size limits, …) before the chain is built. Same contract as WithExecConfigOverride.
func WithChainConfigOverride(f func(*params.ChainConfig)) TestOption {
	return func(b *builder) { b.chainConfigOverrides = append(b.chainConfigOverrides, f) }
}

// WithL1 builds the test on a real parent chain (L1 + sequencer L2 with batch
// posting and inbox reading) instead of the default L2-only node.
func WithL1() TestOption {
	return func(b *builder) {
		setTopology(b, TopologyL1L2, "WithL1")
	}
}

// WithMultiNode builds an L1 + sequencer L2 plus a non-sequencer L2 syncing
// via L1 (env.Followers()).
func WithMultiNode() TestOption {
	return func(b *builder) {
		setTopology(b, TopologyMultiNode, "WithMultiNode")
	}
}

// WithValidation enables block validation (JIT) for this test. Requires a
// topology with a parent chain.
func WithValidation() TestOption {
	return func(b *builder) {
		if b.validation {
			panic("systest: WithValidation applied twice")
		}
		b.validation = true
		b.postHooks = append(b.postHooks, validateToHead)
	}
}

// WithStakingValidation builds an L1 + sequencer L2 plus a follower L2 (env.L2Followers) that
// runs block validation and a staker. Requires wasm machines.
func WithStakingValidation() TestOption {
	return func(b *builder) {
		setTopology(b, TopologyStakingValidation, "WithStakingValidation")
		b.postHooks = append(b.postHooks, validateToHead, verifyStaked)
	}
}

// setTopology pins the node layout, rejecting a second topology option.
func setTopology(b *builder, topo Topology, name string) {
	if b.topology == topo {
		panic(fmt.Sprintf("systest: %s applied twice", name))
	}
	if b.topology != TopologyL2Only {
		panic(fmt.Sprintf("systest: %s conflicts with another topology option", name))
	}
	b.topology = topo
}

// Compose collapses multiple TestOptions into one — used to define presets:
//
//	var Challenge = Compose(WithCategory("challenge"), WithL1())
func Compose(opts ...TestOption) TestOption {
	return func(b *builder) {
		for _, o := range opts {
			o(b)
		}
	}
}
