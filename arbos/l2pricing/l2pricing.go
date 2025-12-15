// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l2pricing

import (
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"

	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

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
	gasConstraints      *storage.SubStorageVector
	multigasConstraints *storage.SubStorageVector

	ArbosVersion uint64
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

var gasConstraintsKey []byte = []byte{0}
var multigasConstraintsKey []byte = []byte{1}

const GethBlockGasLimit = 1 << 50

// TODO(NIT-4152): Number of constraints limited because of retryable redeem gas cost calculation.
const GasConstraintsMaxNum = 20
const MultiGasConstraintsMaxNum = 15

// MaxPricingExponentBips caps the basefee growth: exp(8.5) ~= x5,000 min base fee.
const MaxPricingExponentBips = arbmath.Bips(85_000)

func InitializeL2PricingState(sto *storage.Storage) error {
	_ = sto.SetUint64ByUint64(speedLimitPerSecondOffset, InitialSpeedLimitPerSecondV0)
	_ = sto.SetUint64ByUint64(perBlockGasLimitOffset, InitialPerBlockGasLimitV0)
	_ = sto.SetUint64ByUint64(baseFeeWeiOffset, InitialBaseFeeWei)
	_ = sto.SetUint64ByUint64(gasBacklogOffset, 0)
	_ = sto.SetUint64ByUint64(pricingInertiaOffset, InitialPricingInertia)
	_ = sto.SetUint64ByUint64(backlogToleranceOffset, InitialBacklogTolerance)
	return sto.SetUint64ByUint64(minBaseFeeWeiOffset, InitialMinimumBaseFeeWei)
}

func OpenL2PricingState(sto *storage.Storage, arbosVersion uint64) *L2PricingState {
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
		gasConstraints:      storage.OpenSubStorageVector(sto.OpenSubStorage(gasConstraintsKey)),
		multigasConstraints: storage.OpenSubStorageVector(sto.OpenSubStorage(multigasConstraintsKey)),
		ArbosVersion:        arbosVersion,
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

func (ps *L2PricingState) setGasConstraintsFromLegacy() error {
	if err := ps.ClearGasConstraints(); err != nil {
		return err
	}
	target, err := ps.SpeedLimitPerSecond()
	if err != nil {
		return err
	}
	adjustmentWindow, err := ps.PricingInertia()
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
	return ps.AddGasConstraint(target, adjustmentWindow, backlog)
}

func (ps *L2PricingState) setMultiGasConstraintsFromSingleGasConstraints() error {
	if err := ps.ClearMultiGasConstraints(); err != nil {
		return err
	}

	length, err := ps.GasConstraintsLength()
	if err != nil {
		return err
	}

	for i := range length {
		c := ps.OpenGasConstraintAt(i)

		target, err := c.Target()
		if err != nil {
			return fmt.Errorf("failed to read target from constraint %d: %w", i, err)
		}
		window, err := c.AdjustmentWindow()
		if err != nil {
			return fmt.Errorf("failed to read adjustment window from constraint %d: %w", i, err)
		}
		backlog, err := c.Backlog()
		if err != nil {
			return fmt.Errorf("failed to read backlog from constraint %d: %w", i, err)
		}

		// Transfer to multi-gas constraint with equal weights
		weights := map[uint8]uint64{
			uint8(multigas.ResourceKindComputation):     1,
			uint8(multigas.ResourceKindHistoryGrowth):   1,
			uint8(multigas.ResourceKindStorageAccess):   1,
			uint8(multigas.ResourceKindStorageGrowth):   1,
			uint8(multigas.ResourceKindL2Calldata):      1,
			uint8(multigas.ResourceKindWasmComputation): 1,
		}

		var adjustmentWindow uint32
		if window > math.MaxUint32 {
			adjustmentWindow = math.MaxUint32
		} else {
			adjustmentWindow = uint32(window)
		}

		if err := ps.AddMultiGasConstraint(
			target,
			adjustmentWindow,
			backlog,
			weights,
		); err != nil {
			return fmt.Errorf("failed to add multi-gas constraint %d: %w", i, err)
		}
	}
	return nil
}

func (ps *L2PricingState) AddGasConstraint(target uint64, adjustmentWindow uint64, backlog uint64) error {
	subStorage, err := ps.gasConstraints.Push()
	if err != nil {
		return fmt.Errorf("failed to push constraint: %w", err)
	}
	constraint := OpenGasConstraint(subStorage)
	if err := constraint.SetTarget(target); err != nil {
		return fmt.Errorf("failed to set target: %w", err)
	}
	if err := constraint.SetAdjustmentWindow(adjustmentWindow); err != nil {
		return fmt.Errorf("failed to set adjustment window: %w", err)
	}
	if err := constraint.SetBacklog(backlog); err != nil {
		return fmt.Errorf("failed to set backlog: %w", err)
	}
	return nil
}

func (ps *L2PricingState) GasConstraintsLength() (uint64, error) {
	return ps.gasConstraints.Length()
}

func (ps *L2PricingState) OpenGasConstraintAt(i uint64) *GasConstraint {
	return OpenGasConstraint(ps.gasConstraints.At(i))
}

func (ps *L2PricingState) ClearGasConstraints() error {
	length, err := ps.GasConstraintsLength()
	if err != nil {
		return err
	}
	for range length {
		subStorage, err := ps.gasConstraints.Pop()
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

func (ps *L2PricingState) MultiGasConstraintsLength() (uint64, error) {
	return ps.multigasConstraints.Length()
}

func (ps *L2PricingState) OpenMultiGasConstraintAt(i uint64) *MultiGasConstraint {
	return OpenMultiGasConstraint(ps.multigasConstraints.At(i))
}

func (ps *L2PricingState) AddMultiGasConstraint(
	target uint64,
	adjustmentWindow uint32,
	backlog uint64,
	weights map[uint8]uint64,
) error {
	subStorage, err := ps.multigasConstraints.Push()
	if err != nil {
		return fmt.Errorf("failed to push multi-gas constraint: %w", err)
	}

	constraint := OpenMultiGasConstraint(subStorage)
	if err := constraint.SetTarget(target); err != nil {
		return fmt.Errorf("failed to set target: %w", err)
	}
	if err := constraint.SetAdjustmentWindow(adjustmentWindow); err != nil {
		return fmt.Errorf("failed to set adjustment window: %w", err)
	}
	if err := constraint.SetBacklog(backlog); err != nil {
		return fmt.Errorf("failed to set backlog: %w", err)
	}
	if err := constraint.SetResourceWeights(weights); err != nil {
		return fmt.Errorf("failed to set resource weights: %w", err)
	}
	return nil
}

func (ps *L2PricingState) ClearMultiGasConstraints() error {
	length, err := ps.MultiGasConstraintsLength()
	if err != nil {
		return err
	}
	for range length {
		subStorage, err := ps.multigasConstraints.Pop()
		if err != nil {
			return err
		}
		constraint := OpenMultiGasConstraint(subStorage)
		if err := constraint.Clear(); err != nil {
			return err
		}
	}
	return nil
}
