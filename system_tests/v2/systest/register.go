// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"fmt"
	"maps"
	"path"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/testhelpers/env"
)

// defaultCategory is the implicit category of tests without WithCategory.
const defaultCategory = "default"

// scheduleParams is schedule's input: filters, axis pins, matrix axes, and
// feature toggles. The zero value schedules everything with test defaults.
type scheduleParams struct {
	Tests        map[string]bool
	Categories   map[string]bool
	ArbOS        containers.Option[uint64]
	StateScheme  containers.Option[StateScheme]
	DBEngine     containers.Option[DBEngine]
	MatrixArbOS  []uint64
	MatrixStates []StateScheme
	MatrixDBs    []DBEngine

	AllCategories bool
	// DefaultStateScheme is the ambient scheme (-test_state_scheme flag) used
	// when neither the test nor -v2.state-scheme pins one; resolved at parse time.
	DefaultStateScheme containers.Option[StateScheme]
}

// resolvedScheme returns the explicit -v2.state-scheme pin, else the env default.
func (p scheduleParams) resolvedScheme() containers.Option[StateScheme] {
	if p.StateScheme.IsSome() {
		return p.StateScheme
	}
	return p.DefaultStateScheme
}

// categoryEnabled reports whether a test's category should run: only categories
// listed in -v2.categories run; an empty list means no filter, like the -v2.tests filter.
func (p scheduleParams) categoryEnabled(cat string) bool {
	if p.AllCategories || len(p.Categories) == 0 {
		return true
	}
	return p.Categories[cat]
}

// envDefaultScheme is the ambient state scheme for runs without -v2.state-scheme.
func envDefaultScheme() containers.Option[StateScheme] {
	if s := env.GetTestStateScheme(); s != "" {
		return containers.Some(StateScheme(s))
	}
	return containers.None[StateScheme]()
}

// Test registers a test scenario. Collect into a package-level slice:
//
//	var transferTests = []systest.Scenario{systest.Test(testRunTransfer)}
//
// Naming convention: name the scenario func "testRunX"; the framework strips
// the prefix to produce the test name ("Transfer"). Override with Named("X")
// when the same scenario is registered under multiple configs or when the
// func name doesn't follow the convention.
//
// Panics on duplicate names or on registration after schedule() — i.e.
// after the TestRunner has started.
func Test(scenario Scenario, opts ...TestOption) Scenario {
	b := newBuilder()
	b.scenario = scenario
	for _, o := range opts {
		o(b)
	}
	if b.name == "" {
		b.name = derivedName(scenario)
	}
	b.validate()

	registry.mu.Lock()
	defer registry.mu.Unlock()

	if registry.frozen.Load() {
		panic("systest: Test called after schedule() — registration must complete during package init")
	}
	name := b.name
	if registry.names[name] {
		panic(fmt.Sprintf("systest: duplicate test registration %q (use systest.Named to disambiguate)", name))
	}
	registry.names[name] = true
	registry.items = append(registry.items, b)
	return scenario
}

// testRegistry is the process-wide set of registered tests.
type testRegistry struct {
	mu     sync.Mutex
	items  []*builder
	names  map[string]bool
	frozen atomic.Bool
}

var registry = testRegistry{names: map[string]bool{}}

// registrySnapshot returns the registry as it stood at call time. Internal;
// callers don't mutate the shared builders — schedule clones before mutating.
func registrySnapshot() []*builder {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	out := make([]*builder, len(registry.items))
	copy(out, registry.items)
	return out
}

// schedule applies CLI filters and matrix expansion. Freezes the registry —
// subsequent Test calls panic.
func schedule(cli scheduleParams) []scheduledTest {
	registry.frozen.Store(true)
	unmatchedTests := maps.Clone(cli.Tests)
	unmatchedCategories := maps.Clone(cli.Categories)
	delete(unmatchedCategories, defaultCategory)
	var out []scheduledTest
	for _, orig := range registrySnapshot() {
		delete(unmatchedCategories, orig.category)
		match := len(cli.Tests) == 0
		for p := range cli.Tests {
			if ok, _ := path.Match(p, orig.name); ok {
				delete(unmatchedTests, p)
				match = true
			}
		}
		if !match {
			continue
		}
		// Mutate a copy, not the shared registry builder; clone() drops dims, restore for expansion.
		b := orig.clone()
		b.dims = orig.dims
		out = append(out, expandMatrix(b, cli)...)
	}
	for p := range unmatchedTests {
		panic(fmt.Sprintf("systest: -v2.tests pattern %q matched no registered test", p))
	}
	for c := range unmatchedCategories {
		panic(fmt.Sprintf("systest: -v2.categories value %q has no registered tests", c))
	}
	// Sort scenarios heaviest-first
	sort.SliceStable(out, func(i, j int) bool { return out[i].Spec.Weight > out[j].Spec.Weight })
	return out
}

// scheduledTest is one resolved run produced by schedule. The runner iterates
// over a slice of these and dispatches each to a worker.
type scheduledTest struct {
	Spec       Spec
	Scenario   Scenario
	PostHooks  []Hook
	SkipReason string
	overrides  overrides
}

// maxWeight returns the heaviest weight among runnable scheduled tests, or 0
// for empty input. Skipped items never acquire, so they don't size the pool.
func maxWeight(items []scheduledTest) weight {
	var m weight
	for _, it := range items {
		if it.SkipReason == "" && it.Spec.Weight > m {
			m = it.Spec.Weight
		}
	}
	return m
}
