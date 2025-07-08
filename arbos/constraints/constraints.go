// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// The constraints package tracks the multi-dimensional gas usage to apply constraint-based pricing.
package constraints

import (
	"time"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
)

// The period duration for a resource constraint.
type PeriodSecs uint32

// resourceConstraint defines the max gas target per second for the given period for a single resource.
type resourceConstraint struct {
	period time.Duration
	target uint64
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
type ResourceConstraints map[multigas.ResourceKind]map[PeriodSecs]resourceConstraint

// NewResourceConstraints creates a new set of constraints.
// This type can be used as a reference.
func NewResourceConstraints() ResourceConstraints {
	c := ResourceConstraints{}
	for resource := multigas.ResourceKindUnknown + 1; resource < multigas.NumResourceKind; resource++ {
		c[resource] = map[PeriodSecs]resourceConstraint{}
	}
	return c
}

// SetConstraint adds or updates the given resource constraint.
func (rc ResourceConstraints) SetConstraint(
	resource multigas.ResourceKind, periodSecs PeriodSecs, targetPerPeriod uint64,
) {
	rc[resource][periodSecs] = resourceConstraint{
		period: time.Duration(periodSecs) * time.Second,
		target: targetPerPeriod / uint64(periodSecs),
	}
}

// ClearConstraint removes the given resource constraint.
func (rc ResourceConstraints) ClearConstraint(resource multigas.ResourceKind, periodSecs PeriodSecs) {
	delete(rc[resource], periodSecs)
}
