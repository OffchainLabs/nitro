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
	resources := []multigas.ResourceKind{
		multigas.ResourceKindComputation,
		multigas.ResourceKindStorageAccess,
	}
	s = s.WithResources(resources...)
	for _, r := range resources {
		require.True(t, s.HasResource(r))
	}
}

func TestResourceSetHasResource(t *testing.T) {
	s := EmptyResourceSet()
	require.False(t, s.HasResource(multigas.ResourceKindComputation))
	s = s.WithResources(multigas.ResourceKindComputation)
	require.True(t, s.HasResource(multigas.ResourceKindComputation))
}

func TestResourceSetGetResources(t *testing.T) {
	s := EmptyResourceSet()
	resources := []multigas.ResourceKind{
		multigas.ResourceKindComputation,
		multigas.ResourceKindStorageAccess,
	}
	s = s.WithResources(resources...)
	retrieved := s.GetResources()
	require.Equal(t, resources, retrieved)
}

func TestAddToBacklog(t *testing.T) {
	resources := EmptyResourceSet().WithResources(multigas.ResourceKindComputation, multigas.ResourceKindStorageAccess)
	c := &ResourceConstraint{
		resources: resources,
		backlog:   0,
	}

	gasUsed := multigas.MultiGasFromPairs(
		multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 50},
		multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 75},
		multigas.Pair{Kind: multigas.ResourceKindStorageGrowth, Amount: 100},
	)

	c.AddToBacklog(gasUsed)
	require.Equal(t, uint64(125), c.backlog) // 50 + 75

	// Test saturation
	c.backlog = math.MaxUint64 - 10
	c.AddToBacklog(gasUsed)
	require.Equal(t, c.backlog, uint64(math.MaxUint64))
}

func TestRemoveFromBacklog(t *testing.T) {
	c := &ResourceConstraint{
		backlog:      1000,
		targetPerSec: 50,
	}

	// Remove a small amount
	c.RemoveFromBacklog(10) // Remove 10 * 50 = 500
	require.Equal(t, uint64(500), c.backlog)

	// Remove the rest
	c.RemoveFromBacklog(10) // Remove 10 * 50 = 500
	require.Equal(t, uint64(0), c.backlog)

	// Test saturation (underflow)
	c.backlog = 100
	c.RemoveFromBacklog(10) // Attempt to remove 500
	require.Equal(t, uint64(0), c.backlog)
}

func TestNewResourceConstraints(t *testing.T) {
	rc := NewResourceConstraints()
	require.NotNil(t, rc)
	require.NotNil(t, rc.constraints)
	require.Empty(t, rc.constraints)
}

func TestSetResourceConstraints(t *testing.T) {
	rc := NewResourceConstraints()
	resources := EmptyResourceSet().WithResources(multigas.ResourceKindComputation)
	periodSecs := PeriodSecs(10)
	targetPerSec := uint64(100)

	rc.Set(resources, periodSecs, targetPerSec)

	constraint := rc.Get(resources, periodSecs)
	require.NotNil(t, constraint)
	require.Equal(t, resources, constraint.resources)
	require.Equal(t, periodSecs, constraint.period)
	require.Equal(t, targetPerSec, constraint.targetPerSec)
}

func TestGetResourceConstraints(t *testing.T) {
	rc := NewResourceConstraints()
	resources := EmptyResourceSet().WithResources(multigas.ResourceKindComputation)
	periodSecs := PeriodSecs(10)
	targetPerSec := uint64(100)

	rc.Set(resources, periodSecs, targetPerSec)

	// Test getting an existing constraint
	constraint := rc.Get(resources, periodSecs)
	require.NotNil(t, constraint)
	require.Equal(t, resources, constraint.resources)
	require.Equal(t, periodSecs, constraint.period)
	require.Equal(t, targetPerSec, constraint.targetPerSec)
	require.Equal(t, uint64(0), constraint.backlog)

	// Test getting a non-existent constraint
	nonExistentResources := EmptyResourceSet().WithResources(multigas.ResourceKindStorageAccess)
	constraint = rc.Get(nonExistentResources, periodSecs)
	require.Nil(t, constraint)
}

func TestClearResourceConstraints(t *testing.T) {
	rc := NewResourceConstraints()
	resources := EmptyResourceSet().WithResources(multigas.ResourceKindComputation)
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
	resources1 := EmptyResourceSet().WithResources(multigas.ResourceKindComputation)
	periodSecs1 := PeriodSecs(10)
	targetPerSec1 := uint64(100)

	resources2 := EmptyResourceSet().WithResources(multigas.ResourceKindStorageAccess)
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
		if c.resources == resources1 && c.period == periodSecs1 {
			require.Equal(t, targetPerSec1, c.targetPerSec)
			found1 = true
		}
		if c.resources == resources2 && c.period == periodSecs2 {
			require.Equal(t, targetPerSec2, c.targetPerSec)
			found2 = true
		}
	}
	require.True(t, found1)
	require.True(t, found2)
}
