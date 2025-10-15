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
				res := NewWeightedResourceSet().
					WithResource(multigas.ResourceKindComputation, 1)
				rc.Set(res, PeriodSecs(10), 5_000_000)
				rc.Get(res.WithoutWeights(), PeriodSecs(10)).Backlog = 12
				return rc
			},
		},
		{
			name: "ComplexResourceConstraints",
			buildFunc: func() *ResourceConstraints {
				rc := NewResourceConstraints()

				res1 := NewWeightedResourceSet().
					WithResource(multigas.ResourceKindComputation, 1).
					WithResource(multigas.ResourceKindHistoryGrowth, 2)
				rc.Set(res1, PeriodSecs(12), 7_000_000)
				rc.Get(res1.WithoutWeights(), PeriodSecs(12)).Backlog = 42

				res2 := NewWeightedResourceSet().
					WithResource(multigas.ResourceKindWasmComputation, 3).
					WithResource(multigas.ResourceKindStorageGrowth, 5)
				rc.Set(res2, PeriodSecs(60), 3_500_000)
				rc.Get(res2.WithoutWeights(), PeriodSecs(60)).Backlog = 99

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

func TestStorageResourceConstraintsRLPBacklogPersistence(t *testing.T) {
	rc := NewResourceConstraints()
	res := NewWeightedResourceSet().
		WithResource(multigas.ResourceKindComputation, 1)
	rc.Set(res, PeriodSecs(30), 10_000_000)
	ptr := rc.Get(res.WithoutWeights(), PeriodSecs(30))
	ptr.Backlog = 12345 // initial backlog value

	store := &mockStorageBytes{}
	src := NewStorageResourceConstraints(store)

	require.NoError(t, src.Write(rc))
	require.Greater(t, len(store.buf), 0, "storage should contain RLP bytes after Write")

	loaded, err := src.Load()
	require.NoError(t, err, "Load() failed")
	require.Equal(t, 1, len(loaded.constraints), "expected one constraint")

	for got := range loaded.All() {
		require.Equal(t, uint64(12345), got.Backlog,
			"initial backlog not persisted correctly")
	}

	ptr.Backlog = 99999
	require.NoError(t, src.Write(rc), "Write() after backlog update failed")

	loaded2, err := src.Load()
	require.NoError(t, err, "Load() after backlog update failed")
	require.Equal(t, 1, len(loaded2.constraints), "expected one constraint")

	for got := range loaded2.All() {
		require.Equal(t, uint64(99999), got.Backlog,
			"updated backlog not persisted correctly")
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

func TestStorageDoesNotDependOnWeights(t *testing.T) {
	store := &mockStorageBytes{}
	src := NewStorageResourceConstraints(store)

	res1 := NewWeightedResourceSet().
		WithResource(multigas.ResourceKindComputation, 1).
		WithResource(multigas.ResourceKindStorageAccess, 2)

	err := src.SetConstraint(res1, 200, 10_000_000)
	require.NoError(t, err)

	list, err := src.ListConstraints()
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, res1, list[0].Resources)

	res2 := NewWeightedResourceSet().
		WithResource(multigas.ResourceKindComputation, 50).
		WithResource(multigas.ResourceKindStorageAccess, 10)

	// with same period should override existing entity
	err = src.SetConstraint(res2, 200, 1_000_000)
	require.NoError(t, err)

	list, err = src.ListConstraints()
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, res2, list[0].Resources)
}
