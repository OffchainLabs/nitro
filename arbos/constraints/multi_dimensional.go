// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// The constraints package tracks the multi-dimensional gas usage to apply constraint-based pricing.
package constraints

import (
	"github.com/ethereum/go-ethereum/arbitrum/multigas"

	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// Fixed flat layout for a Multi-Constraint:
// [0] target (uint64)
// [1] adjustmentWindow (uint64)
// [2] backlog (uint64)
// [3..3+NumResourceKind-1] weighted resources (uint64 each)

const (
	targetOffset uint64 = iota
	adjustmentWindowOffset
	backlogOffset
	weightedResourcesBaseOffset
)

// MultiGasConstraint defines a pricing constraint that combines several
// gas resource types, each with a corresponding weight (0 = unused).
type MultiGasConstraint struct {
	target            storage.StorageBackedUint64
	adjustmentWindow  storage.StorageBackedUint64
	backlog           storage.StorageBackedUint64
	weightedResources [multigas.NumResourceKind]storage.StorageBackedUint64
}

// OpenMultiGasConstraint opens or initializes a constraint in the given storage subspace.
func OpenMultiGasConstraint(sto *storage.Storage) *MultiGasConstraint {
	c := &MultiGasConstraint{
		target:           sto.OpenStorageBackedUint64(targetOffset),
		adjustmentWindow: sto.OpenStorageBackedUint64(adjustmentWindowOffset),
		backlog:          sto.OpenStorageBackedUint64(backlogOffset),
	}
	for i := range int(multigas.NumResourceKind) {
		// #nosec G115 safe: NumResourceKind < 2^32
		offset := weightedResourcesBaseOffset + uint64(i)
		c.weightedResources[i] = sto.OpenStorageBackedUint64(offset)
	}
	return c
}

// Clear resets the constraint and all weighted resources.
func (c *MultiGasConstraint) Clear() error {
	if err := c.target.Clear(); err != nil {
		return err
	}
	if err := c.adjustmentWindow.Clear(); err != nil {
		return err
	}
	if err := c.backlog.Clear(); err != nil {
		return err
	}
	for i := range int(multigas.NumResourceKind) {
		if err := c.weightedResources[i].Clear(); err != nil {
			return err
		}
	}
	return nil
}

// ResourceWeightedBacklog returns the portion of the total backlog
// attributed to the given resource kind, proportional to its configured weight.
// If the resource kind is out of range or has zero weight, it returns zero.
func (c *MultiGasConstraint) ResourceWeightedBacklog(kind uint8) (uint64, error) {
	_, err := multigas.CheckResourceKind(kind)
	if err != nil {
		return 0, err
	}

	backlog, err := c.backlog.Get()
	if err != nil {
		return 0, err
	}
	if backlog == 0 {
		return 0, nil
	}

	weight, err := c.weightedResources[kind].Get()
	if err != nil {
		return 0, err
	}
	if weight == 0 {
		return 0, nil
	}

	// Compute total weight across all resources
	var totalWeight uint64
	for i := range int(multigas.NumResourceKind) {
		w, err := c.weightedResources[i].Get()
		if err != nil {
			return 0, err
		}
		totalWeight = arbmath.SaturatingUAdd(totalWeight, w)
	}

	// Compute proportional backlog share
	contrib := arbmath.SaturatingUMul(backlog, weight) / totalWeight
	return contrib, nil
}

// SetBacklogWithMultigas aggregates multi-dimensional gas usage into a single backlog value.
//
// Each resource kind's usage from the provided MultiGas vector is multiplied by its
// configured weight in this constraint. The weighted values are then summed (with
// saturating arithmetic) to form the total backlog, which represents the combined
// resource pressure for this constraint.
func (c *MultiGasConstraint) SetBacklogWithMultigas(multiGas multigas.MultiGas) error {
	var totalBacklog uint64
	for i := range uint8(multigas.NumResourceKind) {
		weight, err := c.weightedResources[i].Get()
		if err != nil {
			return err
		}
		if weight == 0 {
			continue
		}

		resourceAmount := multiGas.Get(multigas.ResourceKind(i))
		weightedAmount := arbmath.SaturatingUMul(weight, resourceAmount)

		totalBacklog = arbmath.SaturatingUAdd(totalBacklog, weightedAmount)
	}
	return c.SetBacklog(totalBacklog)
}

func (c *MultiGasConstraint) Target() (uint64, error) {
	return c.target.Get()
}

func (c *MultiGasConstraint) SetTarget(v uint64) error {
	return c.target.Set(v)
}

func (c *MultiGasConstraint) AdjustmentWindow() (uint64, error) {
	return c.adjustmentWindow.Get()
}

func (c *MultiGasConstraint) SetAdjustmentWindow(v uint64) error {
	return c.adjustmentWindow.Set(v)
}

func (c *MultiGasConstraint) Backlog() (uint64, error) {
	return c.backlog.Get()
}

func (c *MultiGasConstraint) SetBacklog(v uint64) error {
	return c.backlog.Set(v)
}

func (c *MultiGasConstraint) ResourceWeight(kind uint8) (uint64, error) {
	_, err := multigas.CheckResourceKind(kind)
	if err != nil {
		return 0, err
	}
	return c.weightedResources[kind].Get()
}

func (c *MultiGasConstraint) ResourcesWithWeights() (map[multigas.ResourceKind]uint64, error) {
	result := make(map[multigas.ResourceKind]uint64)
	for i := range uint8(multigas.NumResourceKind) {
		weight, err := c.weightedResources[i].Get()
		if err != nil {
			return nil, err
		}
		if weight != 0 {
			result[multigas.ResourceKind(i)] = weight
		}
	}
	return result, nil
}

func (c *MultiGasConstraint) SetResourceWeight(kind uint8, value uint64) error {
	_, err := multigas.CheckResourceKind(kind)
	if err != nil {
		return err
	}
	return c.weightedResources[kind].Set(value)
}
