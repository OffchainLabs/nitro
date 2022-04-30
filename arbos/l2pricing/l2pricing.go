// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package l2pricing

import (
	"errors"
	"math"
	"math/big"

	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type L2PricingState struct {
	storage                       *storage.Storage
	gasPool                       storage.StorageBackedInt64
	gasPoolLastBlock              storage.StorageBackedInt64
	gasPoolSeconds                storage.StorageBackedUint64
	gasPoolTarget                 storage.StorageBackedBips
	gasPoolWeight                 storage.StorageBackedBips
	rateEstimate                  storage.StorageBackedUint64
	rateEstimateInertia           storage.StorageBackedUint64
	speedLimitPerSecond           storage.StorageBackedUint64
	maxPerBlockGasLimit           storage.StorageBackedUint64
	baseFeeWei                    storage.StorageBackedBigInt
	minBaseFeeWei                 storage.StorageBackedBigInt
	exponentialMechanismDenom     storage.StorageBackedUint64
	exponentialMechanismTolerance storage.StorageBackedUint64
}

const (
	gasPoolOffset uint64 = iota
	gasPoolLastBlockOffset
	gasPoolSecondsOffset
	gasPoolTargetOffset
	gasPoolWeightOffset
	rateEstimateOffset
	rateEstimateInertiaOffset
	speedLimitPerSecondOffset
	maxPerBlockGasLimitOffset
	baseFeeWeiOffset
	minBaseFeeWeiOffset
	exponentialMechanismDenomOffset
	exponentialToleranceOffset
)

const GethBlockGasLimit = 1 << 63

func InitializeL2PricingState(sto *storage.Storage, arbosVersion uint64) error {
	_ = sto.SetUint64ByUint64(gasPoolOffset, InitialGasPoolSeconds*InitialSpeedLimitPerSecond)
	_ = sto.SetUint64ByUint64(gasPoolLastBlockOffset, InitialGasPoolSeconds*InitialSpeedLimitPerSecond)
	_ = sto.SetUint64ByUint64(gasPoolSecondsOffset, InitialGasPoolSeconds)
	_ = sto.SetUint64ByUint64(gasPoolTargetOffset, uint64(InitialGasPoolTargetBips))
	_ = sto.SetUint64ByUint64(gasPoolWeightOffset, uint64(InitialGasPoolWeightBips))
	_ = sto.SetUint64ByUint64(rateEstimateOffset, InitialSpeedLimitPerSecond)
	_ = sto.SetUint64ByUint64(rateEstimateInertiaOffset, InitialRateEstimateInertia)
	_ = sto.SetUint64ByUint64(speedLimitPerSecondOffset, InitialSpeedLimitPerSecond)
	_ = sto.SetUint64ByUint64(maxPerBlockGasLimitOffset, InitialPerBlockGasLimit)
	_ = sto.SetUint64ByUint64(baseFeeWeiOffset, InitialBaseFeeWei)
	if arbosVersion >= FirstExponentialPricingVersion {
		_ = sto.SetUint64ByUint64(exponentialMechanismDenomOffset, InitialExponentialMechanismDenom)
		_ = sto.SetUint64ByUint64(exponentialToleranceOffset, InitialExponentialMechanismTolerance)
	}
	return sto.SetUint64ByUint64(minBaseFeeWeiOffset, InitialMinimumBaseFeeWei)
}

func OpenL2PricingState(sto *storage.Storage) *L2PricingState {
	return &L2PricingState{
		sto,
		sto.OpenStorageBackedInt64(gasPoolOffset),
		sto.OpenStorageBackedInt64(gasPoolLastBlockOffset),
		sto.OpenStorageBackedUint64(gasPoolSecondsOffset),
		sto.OpenStorageBackedBips(gasPoolTargetOffset),
		sto.OpenStorageBackedBips(gasPoolWeightOffset),
		sto.OpenStorageBackedUint64(rateEstimateOffset),
		sto.OpenStorageBackedUint64(rateEstimateInertiaOffset),
		sto.OpenStorageBackedUint64(speedLimitPerSecondOffset),
		sto.OpenStorageBackedUint64(maxPerBlockGasLimitOffset),
		sto.OpenStorageBackedBigInt(baseFeeWeiOffset),
		sto.OpenStorageBackedBigInt(minBaseFeeWeiOffset),
		sto.OpenStorageBackedUint64(exponentialMechanismDenomOffset),
		sto.OpenStorageBackedUint64(exponentialToleranceOffset),
	}
}

func (ps *L2PricingState) UpdateToVersion4() error {
	if err := ps.SetExponentialMechanismDenom(InitialExponentialMechanismDenom); err != nil {
		return err
	}
	return ps.SetExponentialMechanismTolerance(InitialExponentialMechanismTolerance)
}

func (ps *L2PricingState) GasPool() (int64, error) {
	return ps.gasPool.Get()
}

func (ps *L2PricingState) SetGasPool(val int64) error {
	return ps.gasPool.Set(val)
}

func (ps *L2PricingState) GasPoolLastBlock() (int64, error) {
	return ps.gasPoolLastBlock.Get()
}

func (ps *L2PricingState) SetGasPoolLastBlock(val int64) {
	ps.Restrict(ps.gasPoolLastBlock.Set(val))
}

func (ps *L2PricingState) GasPoolSeconds() (uint64, error) {
	return ps.gasPoolSeconds.Get()
}

func (ps *L2PricingState) SetGasPoolSeconds(seconds uint64) error {
	limit, err := ps.SpeedLimitPerSecond()
	if err != nil {
		return err
	}
	if seconds == 0 || seconds > 3*60*60 || arbmath.SaturatingUMul(seconds, limit) > math.MaxInt64 {
		return errors.New("GasPoolSeconds is out of bounds")
	}
	if err := ps.clipGasPool(seconds, limit); err != nil {
		return err
	}
	return ps.gasPoolSeconds.Set(seconds)
}

func (ps *L2PricingState) GasPoolTarget() (arbmath.Bips, error) {
	target, err := ps.gasPoolTarget.Get()
	return arbmath.Bips(target), err
}

func (ps *L2PricingState) SetGasPoolTarget(target arbmath.Bips) error {
	if target > arbmath.OneInBips {
		return errors.New("GasPoolTarget is out of bounds")
	}
	return ps.gasPoolTarget.Set(target)
}

func (ps *L2PricingState) GasPoolWeight() (arbmath.Bips, error) {
	return ps.gasPoolWeight.Get()
}

func (ps *L2PricingState) SetGasPoolWeight(weight arbmath.Bips) error {
	if weight > arbmath.OneInBips {
		return errors.New("GasPoolWeight is out of bounds")
	}
	return ps.gasPoolWeight.Set(weight)
}

func (ps *L2PricingState) RateEstimate() (uint64, error) {
	return ps.rateEstimate.Get()
}

func (ps *L2PricingState) SetRateEstimate(rate uint64) {
	ps.Restrict(ps.rateEstimate.Set(rate))
}

func (ps *L2PricingState) RateEstimateInertia() (uint64, error) {
	return ps.rateEstimateInertia.Get()
}

func (ps *L2PricingState) SetRateEstimateInertia(inertia uint64) error {
	return ps.rateEstimateInertia.Set(inertia)
}

func (ps *L2PricingState) BaseFeeWei() (*big.Int, error) {
	return ps.baseFeeWei.Get()
}

func (ps *L2PricingState) SetBaseFeeWei(val *big.Int) error {
	return ps.baseFeeWei.Set(val)
}

func (ps *L2PricingState) MinBaseFeeWei() (*big.Int, error) {
	return ps.minBaseFeeWei.Get()
}

func (ps *L2PricingState) SetMinBaseFeeWei(val *big.Int) error {
	// This modifies the "minimum basefee" parameter, but doesn't modify the current basefee.
	// If this increases the minimum basefee, then the basefee might be below the minimum for a little while.
	// If so, the basefee will increase by up to a factor of two per block, until it reaches the minimum.
	return ps.minBaseFeeWei.Set(val)
}

func (ps *L2PricingState) SpeedLimitPerSecond() (uint64, error) {
	return ps.speedLimitPerSecond.Get()
}

func (ps *L2PricingState) SetSpeedLimitPerSecond(limit uint64) error {
	seconds, err := ps.GasPoolSeconds()
	if err != nil {
		return err
	}
	if limit == 0 || arbmath.SaturatingUMul(seconds, limit) > math.MaxInt64 {
		return errors.New("SetSpeedLimitPerSecond is out of bounds")
	}
	if err := ps.clipGasPool(seconds, limit); err != nil {
		return err
	}
	return ps.speedLimitPerSecond.Set(limit)
}

func (ps *L2PricingState) GasPoolMax() (int64, error) {
	speedLimit, _ := ps.SpeedLimitPerSecond()
	seconds, err := ps.GasPoolSeconds()
	if err != nil {
		return 0, err
	}
	return arbmath.SaturatingCast(seconds * speedLimit), nil
}

func (ps *L2PricingState) MaxPerBlockGasLimit() (uint64, error) {
	return ps.maxPerBlockGasLimit.Get()
}

func (ps *L2PricingState) SetMaxPerBlockGasLimit(limit uint64) error {
	return ps.maxPerBlockGasLimit.Set(limit)
}

func (ps *L2PricingState) ExponentialMechanismDenom() (uint64, error) {
	return ps.exponentialMechanismDenom.Get()
}

func (ps *L2PricingState) SetExponentialMechanismDenom(val uint64) error {
	return ps.exponentialMechanismDenom.Set(val)
}

func (ps *L2PricingState) ExponentialMechanismTolerance() (uint64, error) {
	return ps.exponentialMechanismTolerance.Get()
}

func (ps *L2PricingState) SetExponentialMechanismTolerance(val uint64) error {
	return ps.exponentialMechanismTolerance.Set(val)
}

// Ensure the gas pool is within the implied maximum capacity
func (ps *L2PricingState) clipGasPool(seconds, speedLimit uint64) error {
	pool, err := ps.GasPool()
	if err != nil {
		return err
	}
	newMax := arbmath.SaturatingCast(arbmath.SaturatingUMul(seconds, speedLimit))
	if pool > newMax {
		err = ps.SetGasPool(newMax)
	}
	return err
}

func (ps *L2PricingState) Restrict(err error) {
	ps.storage.Burner().Restrict(err)
}
