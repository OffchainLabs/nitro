// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"fmt"
	"log"
	"maps"
	"strings"

	"github.com/offchainlabs/nitro/util/containers"
)

// Matrix expansion model
//
// A test can declare multiple axes (ArbOS version, state scheme, db engine).
// The runner expands one scheduled test per cell in the cartesian product of
// axis variants. Example: MatrixArbOS(30, 40) × MatrixStateScheme("hash") =
// two scheduled tests (arbos30/hash, arbos40/hash).
//
// CLI flags (-v2.matrix.arbos, etc.) can supply axis variants at runtime;
// they override a declared axis on the same key unless the test pinned the
// axis with WithArbOS / WithStateScheme / WithDBEngine.

// axis identifies one matrix dimension over which a test can be expanded.
// Possible values: ArbOS version, state scheme, db engine. Used as the key
// into builder.dims so pin/matrix conflict detection is a single map lookup.
type axis int

const (
	axisArbOS axis = iota
	axisStateScheme
	axisDBEngine
)

// axisOrder is the deterministic iteration order for matrix expansion —
// guarantees stable subtest names across runs.
var axisOrder = []axis{axisArbOS, axisStateScheme, axisDBEngine}

// axisVariant is one value along an axis: a display name plus a mutator that
// applies the value to a cloned builder during expansion.
type axisVariant struct {
	name  string
	apply func(*builder)
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

func mkArbOSAxis(vs []uint64) []axisVariant {
	var out []axisVariant
	for _, v := range vs {
		out = append(out, axisVariant{
			name:  fmt.Sprintf("arbos%d", v),
			apply: func(b *builder) { b.arbOS = containers.Some(v) },
		})
	}
	return out
}

func mkStateSchemeAxis(ss []StateScheme) []axisVariant {
	var out []axisVariant
	for _, s := range ss {
		out = append(out, axisVariant{
			name:  string(s),
			apply: func(b *builder) { b.stateScheme = containers.Some(s) },
		})
	}
	return out
}

func mkDBEngineAxis(es []DBEngine) []axisVariant {
	var out []axisVariant
	for _, e := range es {
		out = append(out, axisVariant{
			name:  string(e),
			apply: func(b *builder) { b.dbEngine = containers.Some(e) },
		})
	}
	return out
}

func expandMatrix(base *builder, cli scheduleParams) []scheduledTest {
	dims := effectiveAxes(base, cli)

	total := 1
	for _, variants := range dims {
		total *= len(variants)
	}
	if total == 0 {
		panic(fmt.Sprintf("systest: matrix axis with zero variants for test %q", base.name))
	}
	idx := make([]int, len(dims))
	var out []scheduledTest
	for n := range total {
		// Decode cell number n as a mixed-radix number: digit i selects the
		// variant on axis i, rightmost axis fastest.
		rem := n
		for i := len(dims) - 1; i >= 0; i-- {
			idx[i] = rem % len(dims[i])
			rem /= len(dims[i])
		}
		clone := base.clone()
		var parts []string
		for i, variants := range dims {
			v := variants[idx[i]]
			v.apply(clone)
			parts = append(parts, v.name)
		}
		clone.mergeParams(cli)
		out = append(out, scheduledTest{
			Spec:       clone.freeze(strings.Join(parts, "/")),
			Scenario:   base.scenario,
			PostHooks:  clone.postHooks,
			SkipReason: clone.shouldSkip(cli),
			overrides:  overrides{Node: clone.nodeOverrides, Exec: clone.execOverrides, Stack: clone.stackOverrides, InitData: clone.initDataOverrides, ChainConfig: clone.chainConfigOverrides},
		})
	}
	return out
}

// effectiveAxes returns the declared axes plus any CLI-driven matrix axes,
// emitted in deterministic axisOrder. CLI matrix overrides a declared axis
// on the same key, but only when the axis isn't pinned by the test.
func effectiveAxes(base *builder, cli scheduleParams) [][]axisVariant {
	merged := maps.Clone(base.dims)
	if merged == nil {
		merged = map[axis][]axisVariant{}
	}
	replaceFromCLI := func(ax axis, name string, variants []axisVariant) {
		if _, declared := merged[ax]; declared {
			log.Printf("systest: -v2.matrix.%s overrides declared matrix for test %q", name, base.name)
		}
		merged[ax] = variants
	}
	if len(cli.MatrixArbOS) > 0 && base.arbOS.IsNone() {
		replaceFromCLI(axisArbOS, "arbos", mkArbOSAxis(cli.MatrixArbOS))
	}
	if len(cli.MatrixStates) > 0 && base.stateScheme.IsNone() {
		replaceFromCLI(axisStateScheme, "state-scheme", mkStateSchemeAxis(cli.MatrixStates))
	}
	if len(cli.MatrixDBs) > 0 && base.dbEngine.IsNone() {
		replaceFromCLI(axisDBEngine, "db-engine", mkDBEngineAxis(cli.MatrixDBs))
	}
	var out [][]axisVariant
	for _, ax := range axisOrder {
		if v, ok := merged[ax]; ok {
			out = append(out, v)
		}
	}
	return out
}
