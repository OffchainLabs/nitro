// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package constraints

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
)

func TestResourceSetWithResource(t *testing.T) {
	s := EmptyResourceSet().
		WithResource(multigas.ResourceKindComputation, 1).
		WithResource(multigas.ResourceKindStorageAccess, 2)
	for resource, weight := range s.All() {
		require.True(t, s.HasResource(resource))
		require.Equal(t, weight, s.weights[resource])
	}
}

func TestResourceSetGetResources(t *testing.T) {
	s := EmptyResourceSet().
		WithResource(multigas.ResourceKindComputation, 1).
		WithResource(multigas.ResourceKindStorageAccess, 1)

	expected := map[multigas.ResourceKind]ResourceWeight{
		multigas.ResourceKindComputation:   1,
		multigas.ResourceKindStorageAccess: 1,
	}

	actual := make(map[multigas.ResourceKind]ResourceWeight)
	for resource, weight := range s.All() {
		actual[resource] = weight
	}

	require.Equal(t, expected, actual)
}

func TestOverrideResourceWeights(t *testing.T) {
	s := EmptyResourceSet().
		WithResource(multigas.ResourceKindComputation, 1).
		WithResource(multigas.ResourceKindStorageAccess, 2)
	require.Equal(t, ResourceWeight(1), s.weights[multigas.ResourceKindComputation])
	require.Equal(t, ResourceWeight(2), s.weights[multigas.ResourceKindStorageAccess])

	s = s.WithResource(multigas.ResourceKindComputation, 3).
		WithResource(multigas.ResourceKindStorageAccess, 0)
	require.Equal(t, ResourceWeight(3), s.weights[multigas.ResourceKindComputation])
	require.False(t, s.HasResource(multigas.ResourceKindStorageAccess))
}

func TestAddToBacklog(t *testing.T) {
	resources := EmptyResourceSet().
		WithResource(multigas.ResourceKindComputation, 1).
		WithResource(multigas.ResourceKindStorageAccess, 1)
	c := &ResourceConstraint{
		Resources: resources,
		Backlog:   0,
	}

	gasUsed := multigas.MultiGasFromPairs(
		multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 50},
		multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 75},
		multigas.Pair{Kind: multigas.ResourceKindStorageGrowth, Amount: 100},
	)

	c.AddToBacklog(gasUsed)
	require.Equal(t, uint64(125), c.Backlog) // 50 + 75

	// Test saturation
	c.Backlog = math.MaxUint64 - 10
	c.AddToBacklog(gasUsed)
	require.Equal(t, c.Backlog, uint64(math.MaxUint64))
}

func TestAddToBacklogWithWeights(t *testing.T) {
	resources := EmptyResourceSet().
		WithResource(multigas.ResourceKindComputation, 2).
		WithResource(multigas.ResourceKindStorageAccess, 3)
	c := &ResourceConstraint{
		Resources: resources,
		Backlog:   0,
	}

	gasUsed := multigas.MultiGasFromPairs(
		multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 10},    // 10 * 2 = 20
		multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 20},  // 20 * 3 = 60
		multigas.Pair{Kind: multigas.ResourceKindStorageGrowth, Amount: 100}, // ignored
	)

	c.AddToBacklog(gasUsed)
	require.Equal(t, uint64(80), c.Backlog) // 20 + 60 = 80
}

func TestRemoveFromBacklog(t *testing.T) {
	c := &ResourceConstraint{
		Backlog:      1000,
		TargetPerSec: 50,
	}

	// Remove a small amount
	c.RemoveFromBacklog(10) // Remove 10 * 50 = 500
	require.Equal(t, uint64(500), c.Backlog)

	// Remove the rest
	c.RemoveFromBacklog(10) // Remove 10 * 50 = 500
	require.Equal(t, uint64(0), c.Backlog)

	// Test saturation (underflow)
	c.Backlog = 100
	c.RemoveFromBacklog(10) // Attempt to remove 500
	require.Equal(t, uint64(0), c.Backlog)
}

func TestNewResourceConstraints(t *testing.T) {
	rc := NewResourceConstraints()
	require.NotNil(t, rc)
	require.NotNil(t, rc.constraints)
	require.Empty(t, rc.constraints)
}

func TestSetResourceConstraints(t *testing.T) {
	rc := NewResourceConstraints()
	resources := EmptyResourceSet().
		WithResource(multigas.ResourceKindComputation, 1)
	periodSecs := PeriodSecs(10)
	targetPerSec := uint64(100)

	rc.Set(resources, periodSecs, targetPerSec)

	constraint := rc.Get(resources, periodSecs)
	require.NotNil(t, constraint)
	require.Equal(t, resources, constraint.Resources)
	require.Equal(t, periodSecs, constraint.Period)
	require.Equal(t, targetPerSec, constraint.TargetPerSec)
}

func TestGetResourceConstraints(t *testing.T) {
	rc := NewResourceConstraints()
	resources := EmptyResourceSet().
		WithResource(multigas.ResourceKindComputation, 1)
	periodSecs := PeriodSecs(10)
	targetPerSec := uint64(100)

	rc.Set(resources, periodSecs, targetPerSec)

	// Test getting an existing constraint
	constraint := rc.Get(resources, periodSecs)
	require.NotNil(t, constraint)
	require.Equal(t, resources, constraint.Resources)
	require.Equal(t, periodSecs, constraint.Period)
	require.Equal(t, targetPerSec, constraint.TargetPerSec)
	require.Equal(t, uint64(0), constraint.Backlog)

	// Test getting a non-existent constraint
	nonExistentResources := EmptyResourceSet().
		WithResource(multigas.ResourceKindStorageAccess, 1)
	constraint = rc.Get(nonExistentResources, periodSecs)
	require.Nil(t, constraint)
}

func TestClearResourceConstraints(t *testing.T) {
	rc := NewResourceConstraints()
	resources := EmptyResourceSet().
		WithResource(multigas.ResourceKindComputation, 1)
	periodSecs := PeriodSecs(10)
	targetPerSec := uint64(100)

	rc.Set(resources, periodSecs, targetPerSec)

	// Ensure the constraint was set
	constraint := rc.Get(resources, periodSecs)
	require.NotNil(t, constraint)

	// Clear the constraint
	rc.Clear(resources, periodSecs)

	// Ensure the constraint is gone
	constraint = rc.Get(resources, periodSecs)
	require.Nil(t, constraint)
}

func TestAllResourceConstraints(t *testing.T) {
	rc := NewResourceConstraints()
	resources1 := EmptyResourceSet().
		WithResource(multigas.ResourceKindComputation, 1)
	periodSecs1 := PeriodSecs(10)
	targetPerSec1 := uint64(100)

	resources2 := EmptyResourceSet().
		WithResource(multigas.ResourceKindStorageAccess, 1)
	periodSecs2 := PeriodSecs(20)
	targetPerSec2 := uint64(200)

	rc.Set(resources1, periodSecs1, targetPerSec1)
	rc.Set(resources2, periodSecs2, targetPerSec2)

	var constraints []*ResourceConstraint
	for constraint := range rc.All() {
		constraints = append(constraints, constraint)
	}

	require.Len(t, constraints, 2)

	// Check if both constraints are present, order is not guaranteed
	found1 := false
	found2 := false
	for _, c := range constraints {
		if c.Resources == resources1 && c.Period == periodSecs1 {
			require.Equal(t, targetPerSec1, c.TargetPerSec)
			found1 = true
		}
		if c.Resources == resources2 && c.Period == periodSecs2 {
			require.Equal(t, targetPerSec2, c.TargetPerSec)
			found2 = true
		}
	}
	require.True(t, found1)
	require.True(t, found2)
}

// mockStorageBytes is a simple in-memory implementation of storageBytes.
type mockStorageBytes struct {
	buf []byte
}

func (m *mockStorageBytes) Get() ([]byte, error) { return m.buf, nil }
func (m *mockStorageBytes) Set(val []byte) error { m.buf = val; return nil }

func TestStorageResourceConstraintsRLPRoundTrip(t *testing.T) {
	rc := NewResourceConstraints()

	res1 := EmptyResourceSet().
		WithResource(multigas.ResourceKindComputation, 1).
		WithResource(multigas.ResourceKindHistoryGrowth, 2)
	rc.Set(res1, PeriodSecs(12), 7_000_000)
	rc.Get(res1, PeriodSecs(12)).Backlog = 42

	res2 := EmptyResourceSet().
		WithResource(multigas.ResourceKindWasmComputation, 3)
	rc.Set(res2, PeriodSecs(60), 3_500_000)
	rc.Get(res2, PeriodSecs(60)).Backlog = 99

	store := &mockStorageBytes{}
	src := NewStorageResourceConstraints(store)
	require.NoError(t, src.Write(rc))
	require.Greater(t, len(store.buf), 0, "storage should contain RLP bytes after Write")

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
		require.Truef(t, found, "missing constraint after Load (period=%d target=%d)", orig.Period, orig.TargetPerSec)
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
