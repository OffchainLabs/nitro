// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l2pricing

import (
	"fmt"
	"math/big"

	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

const (
	gasConstraintTargetOffset uint64 = iota
	gasConstraintInertiaOffset
	gasConstraintBacklogOffset
)

// GasConstraint tries to keep the gas backlog under the target (per second) for the given inertia factor.
// The inertia can be computed as 30*sqrt(period), where period is the number of seconds which the
// constraint is acting over. For instance, the default inertia of 102 is an approximation of 30*sqrt(12).
type GasConstraint struct {
	target  storage.StorageBackedUint64
	inertia storage.StorageBackedUint64
	backlog storage.StorageBackedUint64
}

func OpenGasConstraint(storage *storage.Storage) *GasConstraint {
	return &GasConstraint{
		target:  storage.OpenStorageBackedUint64(gasConstraintTargetOffset),
		inertia: storage.OpenStorageBackedUint64(gasConstraintInertiaOffset),
		backlog: storage.OpenStorageBackedUint64(gasConstraintBacklogOffset),
	}
}

func (c *GasConstraint) Clear() error {
	if err := c.target.Clear(); err != nil {
		return err
	}
	if err := c.inertia.Clear(); err != nil {
		return err
	}
	if err := c.backlog.Clear(); err != nil {
		return err
	}
	return nil
}

type L2PricingState struct {
	storage             *storage.Storage
	speedLimitPerSecond storage.StorageBackedUint64
	perBlockGasLimit    storage.StorageBackedUint64
	baseFeeWei          storage.StorageBackedBigUint
	minBaseFeeWei       storage.StorageBackedBigUint
	gasBacklog          storage.StorageBackedUint64
	pricingInertia      storage.StorageBackedUint64
	backlogTolerance    storage.StorageBackedUint64
	perTxGasLimit       storage.StorageBackedUint64
	constraints         *storage.SubStorageVector
}

const (
	speedLimitPerSecondOffset uint64 = iota
	perBlockGasLimitOffset
	baseFeeWeiOffset
	minBaseFeeWeiOffset
	gasBacklogOffset
	pricingInertiaOffset
	backlogToleranceOffset
	perTxGasLimitOffset
)

var constraintsKey []byte = []byte{0}

const GethBlockGasLimit = 1 << 50

func InitializeL2PricingState(sto *storage.Storage) error {
	_ = sto.SetUint64ByUint64(speedLimitPerSecondOffset, InitialSpeedLimitPerSecondV0)
	_ = sto.SetUint64ByUint64(perBlockGasLimitOffset, InitialPerBlockGasLimitV0)
	_ = sto.SetUint64ByUint64(baseFeeWeiOffset, InitialBaseFeeWei)
	_ = sto.SetUint64ByUint64(gasBacklogOffset, 0)
	_ = sto.SetUint64ByUint64(pricingInertiaOffset, InitialPricingInertia)
	_ = sto.SetUint64ByUint64(backlogToleranceOffset, InitialBacklogTolerance)
	return sto.SetUint64ByUint64(minBaseFeeWeiOffset, InitialMinimumBaseFeeWei)
}

func OpenL2PricingState(sto *storage.Storage) *L2PricingState {
	return &L2PricingState{
		storage:             sto,
		speedLimitPerSecond: sto.OpenStorageBackedUint64(speedLimitPerSecondOffset),
		perBlockGasLimit:    sto.OpenStorageBackedUint64(perBlockGasLimitOffset),
		baseFeeWei:          sto.OpenStorageBackedBigUint(baseFeeWeiOffset),
		minBaseFeeWei:       sto.OpenStorageBackedBigUint(minBaseFeeWeiOffset),
		gasBacklog:          sto.OpenStorageBackedUint64(gasBacklogOffset),
		pricingInertia:      sto.OpenStorageBackedUint64(pricingInertiaOffset),
		backlogTolerance:    sto.OpenStorageBackedUint64(backlogToleranceOffset),
		perTxGasLimit:       sto.OpenStorageBackedUint64(perTxGasLimitOffset),
		constraints:         storage.OpenSubStorageVector(sto.OpenSubStorage(constraintsKey)),
	}
}

func (ps *L2PricingState) BaseFeeWei() (*big.Int, error) {
	return ps.baseFeeWei.Get()
}

func (ps *L2PricingState) SetBaseFeeWei(val *big.Int) error {
	return ps.baseFeeWei.SetSaturatingWithWarning(val, "L2 base fee")
}

func (ps *L2PricingState) MinBaseFeeWei() (*big.Int, error) {
	return ps.minBaseFeeWei.Get()
}

func (ps *L2PricingState) SetMinBaseFeeWei(val *big.Int) error {
	// This modifies the "minimum basefee" parameter, but doesn't modify the current basefee.
	// If this increases the minimum basefee, then the basefee might be below the minimum for a little while.
	// If so, the basefee will increase by up to a factor of two per block, until it reaches the minimum.
	return ps.minBaseFeeWei.SetChecked(val)
}

func (ps *L2PricingState) SpeedLimitPerSecond() (uint64, error) {
	return ps.speedLimitPerSecond.Get()
}

func (ps *L2PricingState) SetSpeedLimitPerSecond(limit uint64) error {
	return ps.speedLimitPerSecond.Set(limit)
}

func (ps *L2PricingState) PerBlockGasLimit() (uint64, error) {
	return ps.perBlockGasLimit.Get()
}

func (ps *L2PricingState) SetMaxPerBlockGasLimit(limit uint64) error {
	return ps.perBlockGasLimit.Set(limit)
}

func (ps *L2PricingState) PerTxGasLimit() (uint64, error) {
	return ps.perTxGasLimit.Get()
}

func (ps *L2PricingState) SetMaxPerTxGasLimit(limit uint64) error {
	return ps.perTxGasLimit.Set(limit)
}

func (ps *L2PricingState) GasBacklog() (uint64, error) {
	return ps.gasBacklog.Get()
}

func (ps *L2PricingState) SetGasBacklog(backlog uint64) error {
	return ps.gasBacklog.Set(backlog)
}

func (ps *L2PricingState) PricingInertia() (uint64, error) {
	return ps.pricingInertia.Get()
}

func (ps *L2PricingState) SetPricingInertia(val uint64) error {
	return ps.pricingInertia.Set(val)
}

func (ps *L2PricingState) BacklogTolerance() (uint64, error) {
	return ps.backlogTolerance.Get()
}

func (ps *L2PricingState) SetBacklogTolerance(val uint64) error {
	return ps.backlogTolerance.Set(val)
}

func (ps *L2PricingState) Restrict(err error) {
	ps.storage.Burner().Restrict(err)
}

func (ps *L2PricingState) SetConstraintsFromLegacy() error {
	if err := ps.ClearConstraints(); err != nil {
		return err
	}
	target, err := ps.SpeedLimitPerSecond()
	if err != nil {
		return err
	}
	inertia, err := ps.PricingInertia()
	if err != nil {
		return err
	}
	oldBacklog, err := ps.GasBacklog()
	if err != nil {
		return err
	}
	backlogTolerance, err := ps.BacklogTolerance()
	if err != nil {
		return err
	}
	backlog := arbmath.SaturatingUSub(oldBacklog, arbmath.SaturatingUMul(backlogTolerance, target))
	return ps.AddConstraint(target, inertia, backlog)
}

func (ps *L2PricingState) AddConstraint(target uint64, inertia uint64, backlog uint64) error {
	subStorage, err := ps.constraints.Push()
	if err != nil {
		return fmt.Errorf("failed to push constraint: %w", err)
	}
	constraint := OpenGasConstraint(subStorage)
	if err := constraint.target.Set(target); err != nil {
		return fmt.Errorf("failed to set target: %w", err)
	}
	if err := constraint.inertia.Set(inertia); err != nil {
		return fmt.Errorf("failed to set inertia: %w", err)
	}
	if err := constraint.backlog.Set(backlog); err != nil {
		return fmt.Errorf("failed to set backlog: %w", err)
	}
	return nil
}

func (ps *L2PricingState) ConstraintsLength() (uint64, error) {
	return ps.constraints.Length()
}

func (ps *L2PricingState) OpenConstraintAt(i uint64) *GasConstraint {
	return OpenGasConstraint(ps.constraints.At(i))
}

func (ps *L2PricingState) ClearConstraints() error {
	length, err := ps.ConstraintsLength()
	if err != nil {
		return err
	}
	for range length {
		subStorage, err := ps.constraints.Pop()
		if err != nil {
			return err
		}
		constraint := OpenGasConstraint(subStorage)
		if err := constraint.Clear(); err != nil {
			return err
		}
	}
	return nil
}
