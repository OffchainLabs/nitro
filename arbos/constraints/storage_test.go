// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package constraints

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
)

// mockStorageBytes is a simple in-memory implementation of storageBytes.
type mockStorageBytes struct {
	buf []byte
}

func (m *mockStorageBytes) Get() ([]byte, error) { return m.buf, nil }
func (m *mockStorageBytes) Set(val []byte) error { m.buf = val; return nil }

func TestStorageResourceConstraintsRLPRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		buildFunc func() *ResourceConstraints
	}{
		{
			name: "Empty",
			buildFunc: func() *ResourceConstraints {
				return NewResourceConstraints()
			},
		},
		{
			name: "SimpleResourceConstraints",
			buildFunc: func() *ResourceConstraints {
				rc := NewResourceConstraints()
				res := EmptyResourceSet().
					WithResource(multigas.ResourceKindComputation, 1)
				rc.Set(res, PeriodSecs(10), 5_000_000)
				rc.Get(res, PeriodSecs(10)).Backlog = 12
				return rc
			},
		},
		{
			name: "ComplexResourceConstraints",
			buildFunc: func() *ResourceConstraints {
				rc := NewResourceConstraints()

				res1 := EmptyResourceSet().
					WithResource(multigas.ResourceKindComputation, 1).
					WithResource(multigas.ResourceKindHistoryGrowth, 2)
				rc.Set(res1, PeriodSecs(12), 7_000_000)
				rc.Get(res1, PeriodSecs(12)).Backlog = 42

				res2 := EmptyResourceSet().
					WithResource(multigas.ResourceKindWasmComputation, 3).
					WithResource(multigas.ResourceKindStorageGrowth, 5)
				rc.Set(res2, PeriodSecs(60), 3_500_000)
				rc.Get(res2, PeriodSecs(60)).Backlog = 99

				return rc
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := tt.buildFunc()

			store := &mockStorageBytes{}
			src := NewStorageResourceConstraints(store)
			require.NoError(t, src.Write(rc))
			if len(rc.constraints) == 0 {
				require.Len(t, store.buf, 0, "empty ResourceConstraints should not write data")
			} else {
				require.Greater(t, len(store.buf), 0, "storage should contain RLP bytes after Write")
			}

			loaded, err := src.Load()
			require.NoError(t, err, "Load() failed")
			require.Equal(t, len(rc.constraints), len(loaded.constraints), "constraint count mismatch")

			for orig := range rc.All() {
				found := false
				for got := range loaded.All() {
					if got.Period != orig.Period ||
						got.TargetPerSec != orig.TargetPerSec ||
						got.Backlog != orig.Backlog {
						continue
					}
					eq := true
					for i := range orig.Resources.weights {
						if orig.Resources.weights[i] != got.Resources.weights[i] {
							eq = false
							break
						}
					}
					if eq {
						found = true
						break
					}
				}
				require.Truef(t, found,
					"missing constraint after Load (period=%d target=%d)",
					orig.Period, orig.TargetPerSec)
			}
		})
	}
}

func TestStorageResourceConstraintsEmptyLoad(t *testing.T) {
	store := &mockStorageBytes{}
	src := NewStorageResourceConstraints(store)

	loaded, err := src.Load()
	require.NoError(t, err, "Load() failed")
	require.NotNil(t, loaded, "Load() returned nil ResourceConstraints")
	require.Empty(t, loaded.constraints, "Loaded ResourceConstraints should be empty")
}
