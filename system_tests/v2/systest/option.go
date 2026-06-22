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

// WithArbOS pins ArbOS version. Conflicts with -v2.arbos cause skip. Panics
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

// WithCategory tags this test with a named category; untagged tests are in the
// default one. Tests run only when their category is enabled via -v2.categories.
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

// WithTimeout overrides the per-scenario wall-clock backstop (-v2.test-timeout)
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

// Compose collapses multiple TestOptions into one — used to define presets:
//
//	var Challenge = Compose(WithCategory("challenge"), WithMultiNode())
func Compose(opts ...TestOption) TestOption {
	return func(b *builder) {
		for _, o := range opts {
			o(b)
		}
	}
}
