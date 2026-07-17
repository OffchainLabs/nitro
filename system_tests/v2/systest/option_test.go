// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/params"
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
		{"multi-node", TopologyMultiNode, weightHeavy},
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

func TestOptionValueValidation(t *testing.T) {
	cases := []struct {
		name string
		opts []TestOption
		want string
	}{
		{"WithStateScheme invalid", []TestOption{WithStateScheme("bogus")}, "WithStateScheme invalid"},
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
