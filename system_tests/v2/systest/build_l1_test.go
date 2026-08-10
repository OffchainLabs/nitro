// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

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

// TestExpandMatrixCarriesTopology guards against clone()/freeze dropping the
// topology: a WithL1 builder must yield L1L2 specs across every matrix cell.
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
