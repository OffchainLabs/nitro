// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l2pricing

import "github.com/offchainlabs/nitro/arbos/storage"

const (
	gasConstraintTargetOffset uint64 = iota
	gasConstraintAdjustmentWindowOffset
	gasConstraintBacklogOffset
)

// GasConstraint tries to keep the gas backlog under the target (per second) for the given adjustment window.
// Target stands for gas usage per second
// Adjustment window is the time frame over which the price will rise by a factor of e if demand is 2x the target
type GasConstraint struct {
	target           storage.StorageBackedUint64
	adjustmentWindow storage.StorageBackedUint64
	backlog          storage.StorageBackedUint64
}

func OpenGasConstraint(storage *storage.Storage) *GasConstraint {
	return &GasConstraint{
		target:           storage.OpenStorageBackedUint64(gasConstraintTargetOffset),
		adjustmentWindow: storage.OpenStorageBackedUint64(gasConstraintAdjustmentWindowOffset),
		backlog:          storage.OpenStorageBackedUint64(gasConstraintBacklogOffset),
	}
}

func (c *GasConstraint) Clear() error {
	if err := c.target.Clear(); err != nil {
		return err
	}
	if err := c.adjustmentWindow.Clear(); err != nil {
		return err
	}
	if err := c.backlog.Clear(); err != nil {
		return err
	}
	return nil
}

func (c *GasConstraint) Target() (uint64, error) {
	return c.target.Get()
}

func (c *GasConstraint) SetTarget(val uint64) error {
	return c.target.Set(val)
}

func (c *GasConstraint) AdjustmentWindow() (uint64, error) {
	return c.adjustmentWindow.Get()
}

func (c *GasConstraint) SetAdjustmentWindow(val uint64) error {
	return c.adjustmentWindow.Set(val)
}

func (c *GasConstraint) Backlog() (uint64, error) {
	return c.backlog.Get()
}

func (c *GasConstraint) SetBacklog(val uint64) error {
	return c.backlog.Set(val)
}
