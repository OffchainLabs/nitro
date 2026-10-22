// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/containers"
)

func TestPinMatrixConflictPanics(t *testing.T) {
	cases := []struct {
		name string
		opts []TestOption
		want string
	}{
		{"WithArbOS then MatrixArbOS", []TestOption{WithArbOS(params.ArbosVersion_30), MatrixArbOS(params.ArbosVersion_40, params.ArbosVersion_50)}, "MatrixArbOS(...) conflicts with WithArbOS"},
		{"MatrixArbOS then WithArbOS", []TestOption{MatrixArbOS(params.ArbosVersion_40, params.ArbosVersion_50), WithArbOS(params.ArbosVersion_30)}, "WithArbOS(30) conflicts with MatrixArbOS"},
		{"WithStateScheme then MatrixStateScheme", []TestOption{WithStateScheme(StateSchemeHash), MatrixStateScheme(StateSchemePath)}, "MatrixStateScheme(...) conflicts with WithStateScheme"},
		{"MatrixStateScheme then WithStateScheme", []TestOption{MatrixStateScheme(StateSchemePath), WithStateScheme(StateSchemeHash)}, `WithStateScheme("hash") conflicts with MatrixStateScheme`},
		{"WithDBEngine then MatrixDBEngine", []TestOption{WithDBEngine(DBEnginePebble), MatrixDBEngine(DBEngineLevelDB)}, "MatrixDBEngine(...) conflicts with WithDBEngine"},
		{"MatrixDBEngine then WithDBEngine", []TestOption{MatrixDBEngine(DBEngineLevelDB), WithDBEngine(DBEnginePebble)}, `WithDBEngine("pebble") conflicts with MatrixDBEngine`},
		{"MatrixArbOS twice", []TestOption{MatrixArbOS(params.ArbosVersion_30), MatrixArbOS(params.ArbosVersion_40)}, "MatrixArbOS(...) applied twice"},
		{"MatrixStateScheme twice", []TestOption{MatrixStateScheme(StateSchemeHash), MatrixStateScheme(StateSchemePath)}, "MatrixStateScheme(...) applied twice"},
		{"MatrixDBEngine twice", []TestOption{MatrixDBEngine(DBEnginePebble), MatrixDBEngine(DBEngineLevelDB)}, "MatrixDBEngine(...) applied twice"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mustPanic(t, c.want, func() {
				b := newBuilder()
				for _, o := range c.opts {
					o(b)
				}
			})
		})
	}
}

func TestMatrixOptionsRejectInvalidValues(t *testing.T) {
	cases := []struct {
		name string
		fn   func()
		want string
	}{
		{"MatrixStateScheme invalid", func() { _ = MatrixStateScheme(StateSchemeHash, "bogus") }, "MatrixStateScheme invalid"},
		{"MatrixDBEngine invalid", func() { _ = MatrixDBEngine("rocksdb") }, "MatrixDBEngine invalid"},
		{"MatrixArbOS duplicate", func() { _ = MatrixArbOS(params.ArbosVersion_30, params.ArbosVersion_30) }, "MatrixArbOS has duplicate"},
		{"MatrixStateScheme duplicate", func() { _ = MatrixStateScheme(StateSchemeHash, StateSchemeHash) }, "MatrixStateScheme has duplicate"},
		{"MatrixDBEngine duplicate", func() { _ = MatrixDBEngine(DBEnginePebble, DBEnginePebble) }, "MatrixDBEngine has duplicate"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mustPanic(t, c.want, c.fn)
		})
	}
}

func TestMatrixOptionsPanicOnEmpty(t *testing.T) {
	cases := []struct {
		name string
		fn   func()
		want string
	}{
		{"MatrixArbOS()", func() { _ = MatrixArbOS() }, "MatrixArbOS requires"},
		{"MatrixStateScheme()", func() { _ = MatrixStateScheme() }, "MatrixStateScheme requires"},
		{"MatrixDBEngine()", func() { _ = MatrixDBEngine() }, "MatrixDBEngine requires"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mustPanic(t, c.want, c.fn)
		})
	}
}

// TestAxisVariantApply verifies that each variant produced by an axis
// constructor applies the correct value to a builder — exercises
// per-iteration closure capture.
func TestAxisVariantApply(t *testing.T) {
	t.Run("MatrixArbOS", func(t *testing.T) {
		d := mkArbOSAxis([]uint64{params.ArbosVersion_30, params.ArbosVersion_40, params.ArbosVersion_50})
		want := []uint64{params.ArbosVersion_30, params.ArbosVersion_40, params.ArbosVersion_50}
		for i, v := range d {
			b := newBuilder()
			v.apply(b)
			if b.arbOS.IsNone() || b.arbOS.Unwrap() != want[i] {
				t.Errorf("variant %d: arbOS=%v, want %d", i, b.arbOS, want[i])
			}
		}
	})
	t.Run("MatrixStateScheme", func(t *testing.T) {
		d := mkStateSchemeAxis([]StateScheme{StateSchemeHash, StateSchemePath})
		want := []StateScheme{StateSchemeHash, StateSchemePath}
		for i, v := range d {
			b := newBuilder()
			v.apply(b)
			if b.stateScheme.IsNone() || b.stateScheme.Unwrap() != want[i] {
				t.Errorf("variant %d: stateScheme=%v, want %q", i, b.stateScheme, want[i])
			}
		}
	})
	t.Run("MatrixDBEngine", func(t *testing.T) {
		d := mkDBEngineAxis([]DBEngine{DBEnginePebble, DBEngineLevelDB})
		want := []DBEngine{DBEnginePebble, DBEngineLevelDB}
		for i, v := range d {
			b := newBuilder()
			v.apply(b)
			if b.dbEngine.IsNone() || b.dbEngine.Unwrap() != want[i] {
				t.Errorf("variant %d: dbEngine=%v, want %q", i, b.dbEngine, want[i])
			}
		}
	})
}

func TestExpandMatrix(t *testing.T) {
	type tc struct {
		name      string
		setup     func(*builder)
		sp        scheduleParams
		wantNames []string
	}
	cases := []tc{
		{
			name:      "zero dims, no params",
			setup:     func(*builder) {},
			sp:        scheduleParams{},
			wantNames: []string{"X"},
		},
		{
			name: "one declared arbos dim",
			setup: func(b *builder) {
				MatrixArbOS(params.ArbosVersion_30, params.ArbosVersion_40)(b)
			},
			sp:        scheduleParams{},
			wantNames: []string{"X/arbos30", "X/arbos40"},
		},
		{
			name: "params matrix overrides declared dim",
			setup: func(b *builder) {
				MatrixArbOS(params.ArbosVersion_30)(b)
			},
			sp:        scheduleParams{MatrixArbOS: []uint64{params.ArbosVersion_40, params.ArbosVersion_50}},
			wantNames: []string{"X/arbos40", "X/arbos50"},
		},
		{
			name: "pin suppresses params matrix on same axis",
			setup: func(b *builder) {
				b.arbOS = containers.Some(params.ArbosVersion_31)
			},
			sp:        scheduleParams{MatrixArbOS: []uint64{params.ArbosVersion_40, params.ArbosVersion_50}},
			wantNames: []string{"X"},
		},
		{
			name:      "two params dims cartesian",
			setup:     func(*builder) {},
			sp:        scheduleParams{MatrixArbOS: []uint64{params.ArbosVersion_30, params.ArbosVersion_40}, MatrixStates: []StateScheme{StateSchemeHash, StateSchemePath}},
			wantNames: []string{"X/arbos30/hash", "X/arbos30/path", "X/arbos40/hash", "X/arbos40/path"},
		},
		{
			name: "three dims",
			setup: func(b *builder) {
				MatrixArbOS(params.ArbosVersion_30, params.ArbosVersion_40)(b)
				MatrixStateScheme("hash", "path")(b)
				MatrixDBEngine("pebble")(b)
			},
			sp: scheduleParams{},
			wantNames: []string{
				"X/arbos30/hash/pebble", "X/arbos30/path/pebble",
				"X/arbos40/hash/pebble", "X/arbos40/path/pebble",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := newBuilder()
			b.name = "X"
			c.setup(b)
			out := expandMatrix(b, c.sp)
			var got []string
			for _, e := range out {
				got = append(got, e.Spec.Name)
			}
			if !slices.Equal(got, c.wantNames) {
				t.Fatalf("got %v, want %v", got, c.wantNames)
			}
		})
	}
}

func TestOverridesPropagateThroughExpansion(t *testing.T) {
	b := newBuilder()
	b.name = "X"
	WithExecConfigOverride(func(*gethexec.Config) {})(b)
	out := expandMatrix(b, scheduleParams{})
	if len(out) != 1 {
		t.Fatalf("expansion: got %d cells, want 1", len(out))
	}
	if got := len(out[0].overrides.Exec); got != 1 {
		t.Fatalf("Exec overrides in expanded: got %d, want 1", got)
	}
}

func TestMergeParamsFillsEnvDefaultScheme(t *testing.T) {
	b := newBuilder()
	b.name = "E"
	out := expandMatrix(b, scheduleParams{DefaultStateScheme: containers.Some(StateSchemePath)})
	if len(out) != 1 {
		t.Fatalf("got %d cells, want 1", len(out))
	}
	if out[0].Spec.StateScheme.UnwrapOr("") != StateSchemePath {
		t.Fatalf("Spec.StateScheme = %q, want env default path", out[0].Spec.StateScheme.UnwrapOr(""))
	}
}

func TestMatrixCellConflictsWithCLIPin(t *testing.T) {
	b := newBuilder()
	b.name = "C"
	MatrixStateScheme(StateSchemeHash, StateSchemePath)(b)
	dims := b.dims
	clone := b.clone()
	clone.dims = dims

	out := expandMatrix(clone, scheduleParams{StateScheme: containers.Some(StateSchemeHash)})
	if len(out) != 2 {
		t.Fatalf("got %d cells, want 2", len(out))
	}
	for _, cell := range out {
		switch cell.Spec.StateScheme.Unwrap() {
		case StateSchemeHash:
			if cell.SkipReason != "" {
				t.Fatalf("hash cell skipped: %q", cell.SkipReason)
			}
		case StateSchemePath:
			if cell.SkipReason == "" {
				t.Fatal("path cell must skip under state scheme pin hash")
			}
		}
	}
}

func TestExpandMatrixPropagatesPostHooks(t *testing.T) {
	b := newBuilder()
	b.name = "X"
	MatrixArbOS(params.ArbosVersion_30, params.ArbosVersion_40)(b)
	WithPostHook(func(*Env) error { return nil })(b)
	WithPostHook(func(*Env) error { return nil })(b)

	out := expandMatrix(b, scheduleParams{})
	if len(out) != 2 {
		t.Fatalf("got %d cells, want 2", len(out))
	}
	for _, cell := range out {
		if len(cell.PostHooks) != 2 {
			t.Fatalf("cell %q: got %d post-hooks, want 2", cell.Spec.Name, len(cell.PostHooks))
		}
	}
}
