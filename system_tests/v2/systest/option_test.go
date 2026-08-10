// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/containers"
)

func TestDoublePinPanics(t *testing.T) {
	cases := []struct {
		name string
		opts []TestOption
		want string
	}{
		{"WithArbOS twice", []TestOption{WithArbOS(params.ArbosVersion_30), WithArbOS(params.ArbosVersion_40)}, "WithArbOS applied twice"},
		{"WithStateScheme twice", []TestOption{WithStateScheme(StateSchemeHash), WithStateScheme(StateSchemePath)}, "WithStateScheme applied twice"},
		{"WithDBEngine twice", []TestOption{WithDBEngine(DBEnginePebble), WithDBEngine(DBEngineLevelDB)}, "WithDBEngine applied twice"},
		{"Named twice", []TestOption{Named("A"), Named("B")}, "Named applied twice"},
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

func TestSpecWeightDerivation(t *testing.T) {
	tests := []struct {
		name     string
		topology Topology
		want     weight
	}{
		{"L2-only", TopologyL2Only, weightLight},
		{"L1L2", TopologyL1L2, weightMedium},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := specWeight(tt.topology); got != tt.want {
				t.Fatalf("specWeight(%v) = %d, want %d", tt.topology, got, tt.want)
			}
		})
	}
}

func TestWithTimeoutCarriesToSpec(t *testing.T) {
	b := newBuilder()
	if got := b.freeze("").Timeout; got != 0 {
		t.Fatalf("default Spec.Timeout = %v, want 0 (scheduleParams default applies)", got)
	}
	WithTimeout(3 * time.Second)(b)
	if got := b.freeze("").Timeout; got != 3*time.Second {
		t.Fatalf("Spec.Timeout = %v, want 3s", got)
	}
}

func TestDoubleAppliedSilentOptionsNowPanic(t *testing.T) {
	cases := []struct {
		name string
		opts []TestOption
		want string
	}{
		{"WithCategory twice", []TestOption{WithCategory("a"), WithCategory("b")}, "WithCategory applied twice"},
		{"WithCategory empty", []TestOption{WithCategory("")}, "the default category is implicit"},
		{"WithCategory default", []TestOption{WithCategory("default")}, "the default category is implicit"},
		{"MinArbOS twice", []TestOption{MinArbOS(params.ArbosVersion_30), MinArbOS(params.ArbosVersion_40)}, "MinArbOS applied twice"},
		{"MaxArbOS twice", []TestOption{MaxArbOS(params.ArbosVersion_30), MaxArbOS(params.ArbosVersion_40)}, "MaxArbOS applied twice"},
		{"WithTimeout twice", []TestOption{WithTimeout(time.Second), WithTimeout(2 * time.Second)}, "WithTimeout applied twice"},
		{"WithArbOSInit twice", []TestOption{WithArbOSInit(params.ArbOSInit{}), WithArbOSInit(params.ArbOSInit{})}, "WithArbOSInit applied twice"},
		{"WithoutChainOwner twice", []TestOption{WithoutChainOwner(), WithoutChainOwner()}, "WithoutChainOwner applied twice"},
		{"MinArbOS zero", []TestOption{MinArbOS(0)}, "MinArbOS version must be positive"},
		{"MaxArbOS zero", []TestOption{MaxArbOS(0)}, "MaxArbOS version must be positive"},
		{"Named empty", []TestOption{Named("")}, "must be non-empty"},
		{"Named with slash", []TestOption{Named("Foo/arbos30")}, "must not contain '/'"},
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

func TestSkipOnStateSchemesSkipsMatchingPin(t *testing.T) {
	b := newBuilder()
	SkipOnStateSchemes(StateSchemePath)(b)
	if got := b.shouldSkip(scheduleParams{StateScheme: containers.Some(StateSchemePath)}); got != `incompatible with state scheme "path"` {
		t.Fatalf("matching scheme pin: got %q, want skip reason", got)
	}
	if got := b.shouldSkip(scheduleParams{StateScheme: containers.Some(StateSchemeHash)}); got != "" {
		t.Fatalf("non-matching scheme pin must not skip, got %q", got)
	}
}

func TestSkipOnRace(t *testing.T) {
	b := newBuilder()
	SkipOnRace()(b)
	saved := raceEnabled
	defer func() { raceEnabled = saved }()
	raceEnabled = true
	if got := b.shouldSkip(scheduleParams{}); got != "skipped under -race" {
		t.Fatalf("race build: got %q, want race skip reason", got)
	}
	raceEnabled = false
	if got := b.shouldSkip(scheduleParams{}); got != "" {
		t.Fatalf("non-race build must not skip, got %q", got)
	}
}

func TestOptionsCarryToSpec(t *testing.T) {
	b := newBuilder()
	WithArbOSInit(params.ArbOSInit{TransactionFilteringEnabled: true})(b)
	WithoutChainOwner()(b)
	WithRPCEndpoints()(b)
	spec := b.freeze("")
	if spec.arbOSInit == nil || !spec.arbOSInit.TransactionFilteringEnabled {
		t.Fatalf("Spec.arbOSInit = %+v, want WithArbOSInit's value", spec.arbOSInit)
	}
	if !spec.SkipChainOwner {
		t.Fatal("WithoutChainOwner did not set Spec.SkipChainOwner")
	}
	if !spec.ExposeRPC {
		t.Fatal("WithRPCEndpoints did not set Spec.ExposeRPC")
	}
}

func TestWithInitDataOverrideAccumulates(t *testing.T) {
	b := newBuilder()
	WithInitDataOverride(func(*statetransfer.ArbosInitializationInfo) {})(b)
	WithInitDataOverride(func(*statetransfer.ArbosInitializationInfo) {})(b)
	if got := len(b.initDataOverrides); got != 2 {
		t.Fatalf("initDataOverrides: got %d, want 2", got)
	}
}

func TestComposeAppliesAllOptions(t *testing.T) {
	preset := Compose(WithCategory("challenge"), WithL1())
	b := newBuilder()
	preset(b)
	if b.category != "challenge" {
		t.Fatalf("category = %q, want challenge", b.category)
	}
	if b.topology != TopologyL1L2 {
		t.Fatalf("topology = %v, want TopologyL1L2", b.topology)
	}
}

func TestOptionValueValidation(t *testing.T) {
	cases := []struct {
		name string
		opts []TestOption
		want string
	}{
		{"WithStateScheme invalid", []TestOption{WithStateScheme("bogus")}, "WithStateScheme invalid"},
		{"SkipOnStateSchemes invalid", []TestOption{SkipOnStateSchemes("bogus")}, "SkipOnStateSchemes invalid"},
		{"WithDBEngine invalid", []TestOption{WithDBEngine("rocksdb")}, "WithDBEngine invalid"},
		{"WithTimeout zero", []TestOption{WithTimeout(0)}, "must be positive"},
		{"WithTimeout negative", []TestOption{WithTimeout(-time.Second)}, "must be positive"},
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

func TestWithL1(t *testing.T) {
	// Default builder is L2-only and freezes to TopologyL2Only.
	base := newBuilder()
	if base.topology != TopologyL2Only {
		t.Fatalf("default topology = %v, want TopologyL2Only", base.topology)
	}

	b := newBuilder()
	WithL1()(b)
	if b.topology != TopologyL1L2 {
		t.Fatalf("topology = %v, want TopologyL1L2", b.topology)
	}
	if w := specWeight(b.topology); w != weightMedium {
		t.Fatalf("weight = %d, want weightMedium (%d)", w, weightMedium)
	}
	if spec := b.freeze(""); spec.Topology != TopologyL1L2 {
		t.Fatalf("frozen Spec.Topology = %v, want TopologyL1L2", spec.Topology)
	}
}

func TestTopologyConflictPanics(t *testing.T) {
	cases := []struct {
		name string
		opts []TestOption
		want string
	}{
		{"WithL1 twice", []TestOption{WithL1(), WithL1()}, "WithL1 applied twice"},
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

func TestExpandMatrixCarriesTopology(t *testing.T) {
	b := newBuilder()
	b.name = "X"
	WithL1()(b)
	MatrixArbOS(params.ArbosVersion_30, params.ArbosVersion_40)(b)
	out := expandMatrix(b, scheduleParams{})
	if len(out) != 2 {
		t.Fatalf("got %d cells, want 2", len(out))
	}
	for _, e := range out {
		if e.Spec.Topology != TopologyL1L2 {
			t.Fatalf("cell %q topology = %v, want TopologyL1L2", e.Spec.Name, e.Spec.Topology)
		}
	}
}
