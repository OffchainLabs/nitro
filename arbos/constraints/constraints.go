// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// The constraints package tracks the multi-dimensional gas usage to apply constraint-based pricing.
package constraints

import (
	"io"
	"iter"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/util/arbmath"
)

// PeriodSecs is the period in seconds for a resource constraint to reach the target.
// Going over the target for this given period will increase the gas price.
type PeriodSecs uint32

type ResourceWeight uint64

// ResourceSet tracks resource weights for constraint calculation.
type ResourceSet struct {
	weights [multigas.NumResourceKind]ResourceWeight
}

// EmptyResourceSet creates a new set with all weights initialized to zero.
func EmptyResourceSet() ResourceSet {
	return ResourceSet{
		weights: [multigas.NumResourceKind]ResourceWeight{},
	}
}

// WithResource sets the weight for a single resource.
func (s ResourceSet) WithResource(resource multigas.ResourceKind, weight ResourceWeight) ResourceSet {
	s.weights[resource] = weight
	return s
}

// HasResource returns true if the resource has a non-zero weight in the set.
func (s ResourceSet) HasResource(resource multigas.ResourceKind) bool {
	return s.weights[resource] != 0
}

// All returns all resources with non-zero weights.
func (s ResourceSet) All() iter.Seq2[multigas.ResourceKind, ResourceWeight] {
	return func(yield func(multigas.ResourceKind, ResourceWeight) bool) {
		for i, weight := range s.weights {
			if weight != 0 {
				//nolint:gosec // G115: Safe conversion, s.weights length is multigas.NumResourceKind
				resource := multigas.ResourceKind(i)
				if !yield(resource, weight) {
					break
				}
			}
		}
	}
}

// ResourceConstraint defines the max gas target per second for the given period for a single resource.
type ResourceConstraint struct {
	Resources    ResourceSet
	Period       PeriodSecs
	TargetPerSec uint64
	Backlog      uint64
}

// AddToBacklog increases the constraint backlog given the multi-dimensional gas used multiplied by their weights.
func (c *ResourceConstraint) AddToBacklog(gasUsed multigas.MultiGas) {
	for resource, weight := range c.Resources.All() {
		weightedGas := gasUsed.Get(resource) * uint64(weight)
		c.Backlog = arbmath.SaturatingUAdd(c.Backlog, weightedGas)
	}
}

// RemoveFromBacklog decreases the backlog by its target given the amount of time passed.
func (c *ResourceConstraint) RemoveFromBacklog(timeElapsed uint64) {
	c.Backlog = arbmath.SaturatingUSub(c.Backlog, timeElapsed*c.TargetPerSec)
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

// storageBytes defines the interface for ArbOS storage.
type storageBytes interface {
	Get() ([]byte, error)
	Set(val []byte) error
}

// StorageResourceConstraints defines a storage-backed ResourceConstraints.
type StorageResourceConstraints struct {
	storage storageBytes
}

// NewStorageResourceConstraints creates a new storage-backed ResourceConstraints.
func NewStorageResourceConstraints(storage storageBytes) *StorageResourceConstraints {
	return &StorageResourceConstraints{
		storage: storage,
	}
}

type resourceConstraintRLP struct {
	Resources    []ResourceWeight
	Period       PeriodSecs
	TargetPerSec uint64
	Backlog      uint64
}

// EncodeRLP encodes ResourceConstraint deterministically,
// ensuring the fixed-length weights array is preserved.
func (c *ResourceConstraint) EncodeRLP(w io.Writer) error {
	weights := make([]ResourceWeight, len(c.Resources.weights))
	copy(weights, c.Resources.weights[:])
	return rlp.Encode(w, resourceConstraintRLP{
		Resources:    weights,
		Period:       c.Period,
		TargetPerSec: c.TargetPerSec,
		Backlog:      c.Backlog,
	})
}

// DecodeRLP decodes ResourceConstraint deterministically,
// padding or truncating the weights slice to the correct array length.
func (c *ResourceConstraint) DecodeRLP(s *rlp.Stream) error {
	var raw resourceConstraintRLP
	if err := s.Decode(&raw); err != nil {
		return err
	}
	c.Period = raw.Period
	c.TargetPerSec = raw.TargetPerSec
	c.Backlog = raw.Backlog

	for i := range c.Resources.weights {
		if i < len(raw.Resources) {
			c.Resources.weights[i] = raw.Resources[i]
		} else {
			c.Resources.weights[i] = 0
		}
	}
	return nil
}

// Load decodes ResourceConstraints from storage using RLP.
// If storage is empty, returns an empty ResourceConstraints.
func (src *StorageResourceConstraints) Load() (*ResourceConstraints, error) {
	data, err := src.storage.Get()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return NewResourceConstraints(), nil
	}

	var list []*ResourceConstraint
	if err := rlp.DecodeBytes(data, &list); err != nil {
		return nil, err
	}

	rc := NewResourceConstraints()
	for _, c := range list {
		rc.Set(c.Resources, c.Period, c.TargetPerSec)
		ptr := rc.Get(c.Resources, c.Period)
		ptr.Backlog = c.Backlog
	}

	return rc, nil
}

// Write encodes ResourceConstraints into storage using RLP.
func (src *StorageResourceConstraints) Write(rc *ResourceConstraints) error {
	var list []*ResourceConstraint
	for c := range rc.All() {
		list = append(list, c)
	}

	data, err := rlp.EncodeToBytes(list)
	if err != nil {
		return err
	}
	return src.storage.Set(data)
}
