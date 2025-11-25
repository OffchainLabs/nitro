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
// [3] sumWeights (uint64)
// [4..4+NumResourceKind-1] weighted resources (uint64 each)

const (
	targetOffset uint64 = iota
	adjustmentWindowOffset
	backlogOffset
	sumWeightsOffset
	weightedResourcesBaseOffset
)

// MultiGasConstraint defines a pricing constraint that combines several
// gas resource types, each with a corresponding weight (0 = unused).
type MultiGasConstraint struct {
	target            storage.StorageBackedUint64
	adjustmentWindow  storage.StorageBackedUint32
	backlog           storage.StorageBackedUint64
	sumWeights        storage.StorageBackedUint64
	weightedResources [multigas.NumResourceKind]storage.StorageBackedUint64
}

// OpenMultiGasConstraint opens or initializes a constraint in the given storage subspace.
func OpenMultiGasConstraint(sto *storage.Storage) *MultiGasConstraint {
	c := &MultiGasConstraint{
		target:           sto.OpenStorageBackedUint64(targetOffset),
		adjustmentWindow: sto.OpenStorageBackedUint32(adjustmentWindowOffset),
		backlog:          sto.OpenStorageBackedUint64(backlogOffset),
		sumWeights:       sto.OpenStorageBackedUint64(sumWeightsOffset),
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
	if err := c.sumWeights.Clear(); err != nil {
		return err
	}
	for i := range int(multigas.NumResourceKind) {
		if err := c.weightedResources[i].Clear(); err != nil {
			return err
		}
	}
	return nil
}

// SetResourceWeights assigns per-resource weight multipliers for this constraint.
func (c *MultiGasConstraint) SetResourceWeights(weights map[uint8]uint64) error {
	var total uint64
	for kind, weight := range weights {
		if _, err := multigas.CheckResourceKind(kind); err != nil {
			return err
		}
		total = arbmath.SaturatingUAdd(total, weight)
	}
	for i := range int(multigas.NumResourceKind) {
		// #nosec G115 safe: NumResourceKind < 2^32
		weight := weights[uint8(i)]
		if err := c.weightedResources[i].Set(weight); err != nil {
			return err
		}
	}
	return c.sumWeights.Set(total)
}

// IncrementBacklog increments the constraint backlog based on multi-dimensional gas usage
func (c *MultiGasConstraint) IncrementBacklog(multiGas multigas.MultiGas) error {
	totalBacklog, err := c.backlog.Get()
	if err != nil {
		return err
	}

	for i := range uint8(multigas.NumResourceKind) {
		weight, err := c.weightedResources[i].Get()
		if err != nil {
			return err
		}
		if weight == 0 {
			continue
		}

		resourceAmount := multiGas.Get(multigas.ResourceKind(i))
		weightedAmount := arbmath.SaturatingUMul(resourceAmount, uint64(weight))

		totalBacklog = arbmath.SaturatingUAdd(totalBacklog, weightedAmount)
	}
	return c.SetBacklog(totalBacklog)
}

// DecrementBacklog decreases the constraint backlog based on multi-dimensional gas usage
func (c *MultiGasConstraint) DecrementBacklog(multiGas multigas.MultiGas) error {
	totalBacklog, err := c.backlog.Get()
	if err != nil {
		return err
	}

	for i := range uint8(multigas.NumResourceKind) {
		weight, err := c.weightedResources[i].Get()
		if err != nil {
			return err
		}
		if weight == 0 {
			continue
		}

		resourceAmount := multiGas.Get(multigas.ResourceKind(i))
		weightedAmount := arbmath.SaturatingUMul(resourceAmount, uint64(weight))

		totalBacklog = arbmath.SaturatingUSub(totalBacklog, weightedAmount)
	}

	return c.SetBacklog(totalBacklog)
}

func (c *MultiGasConstraint) Target() (uint64, error) {
	return c.target.Get()
}

func (c *MultiGasConstraint) SetTarget(v uint64) error {
	return c.target.Set(v)
}

func (c *MultiGasConstraint) AdjustmentWindow() (uint32, error) {
	return c.adjustmentWindow.Get()
}

func (c *MultiGasConstraint) SetAdjustmentWindow(v uint32) error {
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

func (c *MultiGasConstraint) SumWeights() (uint64, error) {
	return c.sumWeights.Get()
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

func (c *MultiGasConstraint) UsedResources() ([]multigas.ResourceKind, error) {
	var result []multigas.ResourceKind
	for i := range uint8(multigas.NumResourceKind) {
		weight, err := c.weightedResources[i].Get()
		if err != nil {
			return nil, err
		}
		if weight != 0 {
			result = append(result, multigas.ResourceKind(i))
		}
	}
	return result, nil
}
