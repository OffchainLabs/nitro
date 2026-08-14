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
// scheduleParams matrix axes can supply variants at runtime;
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

func expandMatrix(base *builder, sp scheduleParams) []scheduledTest {
	dims := effectiveAxes(base, sp)

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
		clone.mergeParams(sp)
		out = append(out, scheduledTest{
			Spec:       clone.freeze(strings.Join(parts, "/")),
			Scenario:   base.scenario,
			PostHooks:  clone.postHooks,
			SkipReason: clone.shouldSkip(sp),
			overrides:  overrides{Node: clone.nodeOverrides, Exec: clone.execOverrides, Stack: clone.stackOverrides, InitData: clone.initDataOverrides, ChainConfig: clone.chainConfigOverrides},
		})
	}
	return out
}

// effectiveAxes returns the declared axes plus any params-driven matrix axes,
// emitted in deterministic axisOrder. A params matrix overrides a declared axis
// on the same key, but only when the axis isn't pinned by the test.
func effectiveAxes(base *builder, sp scheduleParams) [][]axisVariant {
	merged := maps.Clone(base.dims)
	if merged == nil {
		merged = map[axis][]axisVariant{}
	}
	replaceAxis := func(ax axis, name string, variants []axisVariant) {
		if _, declared := merged[ax]; declared {
			log.Printf("systest: matrix param %s overrides declared matrix for test %q", name, base.name)
		}
		merged[ax] = variants
	}
	if len(sp.MatrixArbOS) > 0 && base.arbOS.IsNone() {
		replaceAxis(axisArbOS, "arbos", mkArbOSAxis(sp.MatrixArbOS))
	}
	if len(sp.MatrixStates) > 0 && base.stateScheme.IsNone() {
		replaceAxis(axisStateScheme, "state-scheme", mkStateSchemeAxis(sp.MatrixStates))
	}
	if len(sp.MatrixDBs) > 0 && base.dbEngine.IsNone() {
		replaceAxis(axisDBEngine, "db-engine", mkDBEngineAxis(sp.MatrixDBs))
	}
	var out [][]axisVariant
	for _, ax := range axisOrder {
		if v, ok := merged[ax]; ok {
			out = append(out, v)
		}
	}
	return out
}
