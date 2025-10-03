// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// The constraints package tracks the multi-dimensional gas usage to apply constraint-based pricing.
package constraints

import (
	"iter"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
)

// PeriodSecs is the period in seconds for a resource constraint to reach the target.
// Going over the target for this given period will increase the gas price.
type PeriodSecs uint32

// ResourceSet is a set of resources.
type ResourceSet uint32

// EmptyResourceSet creates a new set.
func EmptyResourceSet() ResourceSet {
	return ResourceSet(0)
}

// WithResource adds a resource to the set.
func (s ResourceSet) WithResource(resource multigas.ResourceKind) ResourceSet {
	return s | (1 << resource)
}

// WithResources adds the list of resources to the set.
func (s ResourceSet) WithResources(resources []multigas.ResourceKind) ResourceSet {
	for _, resource := range resources {
		s = s.WithResource(resource)
	}
	return s
}

// HasResource returns whether the given resource is in the set.
func (s ResourceSet) HasResource(resource multigas.ResourceKind) bool {
	return (s & (1 << resource)) != 0
}

// GetResources returns the list of resources in the set.
func (s ResourceSet) GetResources() []multigas.ResourceKind {
	var resources []multigas.ResourceKind
	for resource := range multigas.NumResourceKind {
		if s.HasResource(resource) {
			resources = append(resources, resource)
		}
	}
	return resources
}

// ResourceConstraint defines the max gas target per second for the given period for a single resource.
type ResourceConstraint struct {
	Resources    ResourceSet
	Period       PeriodSecs
	TargetPerSec uint64
	Backlog      uint64
}

// constraintKey identifies a resource constraint. There can be only one constraint given the
// resource set and the period.
type constraintKey struct {
	resources ResourceSet
	period    PeriodSecs
}

// ResourceConstraints is a set of constraints for all resources.
//
// The chain owner defines constraints to limit the usage of each resource. A resource can have
// multiple constraints with different periods, but there may be a single constraint given the
// resource and period.
//
// Example constraints:
// - X amount of computation over 12 seconds so nodes can keep up.
// - Y amount of computation over 7 days so fresh nodes can catch up with the chain.
// - Z amount of history growth over one month to avoid bloat.
type ResourceConstraints struct {
	constraints map[constraintKey]*ResourceConstraint
}

// NewResourceConstraints creates a new set of constraints.
func NewResourceConstraints() *ResourceConstraints {
	c := &ResourceConstraints{}
	c.constraints = map[constraintKey]*ResourceConstraint{}
	return c
}

// Set adds or updates the given resource constraint.
// The set of resources and the period are the key that defines the constraint.
func (rc *ResourceConstraints) Set(
	resources ResourceSet, periodSecs PeriodSecs, targetPerSec uint64,
) {
	key := constraintKey{
		resources: resources,
		period:    periodSecs,
	}
	constraint := &ResourceConstraint{
		Resources:    resources,
		Period:       periodSecs,
		TargetPerSec: targetPerSec,
		Backlog:      0,
	}
	rc.constraints[key] = constraint
}

// Get gets the constraint given its key.
func (rc *ResourceConstraints) Get(resources ResourceSet, periodSecs PeriodSecs) *ResourceConstraint {
	key := constraintKey{
		resources: resources,
		period:    periodSecs,
	}
	return rc.constraints[key]
}

// Clear removes the given resource constraint.
func (rc *ResourceConstraints) Clear(resources ResourceSet, periodSecs PeriodSecs) {
	key := constraintKey{
		resources: resources,
		period:    periodSecs,
	}
	delete(rc.constraints, key)
}

// All iterates over the resource constraints.
func (rc *ResourceConstraints) All() iter.Seq[*ResourceConstraint] {
	return func(yield func(*ResourceConstraint) bool) {
		for _, constraint := range rc.constraints {
			if !yield(constraint) {
				return
			}
		}
	}
}
