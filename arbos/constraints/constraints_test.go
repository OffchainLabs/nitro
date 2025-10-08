// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package constraints

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
)

func TestResourceSetWithResources(t *testing.T) {
	s := EmptyResourceSet()
	resourceWeights := map[multigas.ResourceKind]uint8{
		multigas.ResourceKindComputation:   1,
		multigas.ResourceKindStorageAccess: 2,
	}
	s = s.WithResources(resourceWeights)
	for resource := range resourceWeights {
		require.True(t, s.HasResource(resource))
	}
}

func TestResourceSetHasResource(t *testing.T) {
	s := EmptyResourceSet()
	require.False(t, s.HasResource(multigas.ResourceKindComputation))
	s = s.WithResources(map[multigas.ResourceKind]uint8{
		multigas.ResourceKindComputation: 1,
	})
	require.True(t, s.HasResource(multigas.ResourceKindComputation))
}

func TestResourceSetGetResources(t *testing.T) {
	s := EmptyResourceSet()
	resourceWeights := map[multigas.ResourceKind]uint8{
		multigas.ResourceKindComputation:   1,
		multigas.ResourceKindStorageAccess: 1,
	}
	s = s.WithResources(resourceWeights)
	retrieved := s.GetResources()
	expectedResources := []multigas.ResourceKind{
		multigas.ResourceKindComputation,
		multigas.ResourceKindStorageAccess,
	}
	require.Equal(t, expectedResources, retrieved)
}

func TestOverrideResourceWeights(t *testing.T) {
	s := EmptyResourceSet()
	resourceWeights := map[multigas.ResourceKind]uint8{
		multigas.ResourceKindComputation:   1,
		multigas.ResourceKindStorageAccess: 2,
	}
	s = s.WithResources(resourceWeights)
	require.Equal(t, uint8(1), s.weights[multigas.ResourceKindComputation])
	require.Equal(t, uint8(2), s.weights[multigas.ResourceKindStorageAccess])

	s = s.WithResources(map[multigas.ResourceKind]uint8{
		multigas.ResourceKindComputation:   3,
		multigas.ResourceKindStorageAccess: 0,
	})
	require.Equal(t, uint8(3), s.weights[multigas.ResourceKindComputation])
	require.False(t, s.HasResource(multigas.ResourceKindStorageAccess))
}

func TestAddToBacklog(t *testing.T) {
	resources := EmptyResourceSet().WithResources(map[multigas.ResourceKind]uint8{
		multigas.ResourceKindComputation:   1,
		multigas.ResourceKindStorageAccess: 1,
	})
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
	resources := EmptyResourceSet().WithResources(map[multigas.ResourceKind]uint8{
		multigas.ResourceKindComputation:   2,
		multigas.ResourceKindStorageAccess: 3,
	})
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
	resources := EmptyResourceSet().WithResources(map[multigas.ResourceKind]uint8{
		multigas.ResourceKindComputation: 1,
	})
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
	resources := EmptyResourceSet().WithResources(map[multigas.ResourceKind]uint8{
		multigas.ResourceKindComputation: 1,
	})
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
	nonExistentResources := EmptyResourceSet().WithResources(map[multigas.ResourceKind]uint8{
		multigas.ResourceKindStorageAccess: 1,
	})
	constraint = rc.Get(nonExistentResources, periodSecs)
	require.Nil(t, constraint)
}

func TestClearResourceConstraints(t *testing.T) {
	rc := NewResourceConstraints()
	resources := EmptyResourceSet().WithResources(map[multigas.ResourceKind]uint8{
		multigas.ResourceKindComputation: 1,
	})
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
	resources1 := EmptyResourceSet().WithResources(map[multigas.ResourceKind]uint8{
		multigas.ResourceKindComputation: 1,
	})
	periodSecs1 := PeriodSecs(10)
	targetPerSec1 := uint64(100)

	resources2 := EmptyResourceSet().WithResources(map[multigas.ResourceKind]uint8{
		multigas.ResourceKindStorageAccess: 1,
	})
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
