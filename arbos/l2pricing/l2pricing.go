//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package l2pricing

import (
	"errors"
	"math"
	"math/big"

	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/util"
)

type L2PricingState struct {
	storage             *storage.Storage
	gasPool             storage.StorageBackedInt64
	gasPoolLastBlock    storage.StorageBackedInt64
	gasPoolSeconds      storage.StorageBackedUint64
	gasPoolTarget       storage.StorageBackedUint64
	gasPoolWeight       storage.StorageBackedUint64
	rateEstimate        storage.StorageBackedUint64
	rateEstimateInertia storage.StorageBackedUint64
	speedLimitPerSecond storage.StorageBackedUint64
	maxPerBlockGasLimit storage.StorageBackedUint64
	gasPriceWei         storage.StorageBackedBigInt
	minGasPriceWei      storage.StorageBackedBigInt
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
	gasPriceWeiOffset
	minGasPriceWeiOffset
)

const GethBlockGasLimit = 1 << 63

func InitializeL2PricingState(sto *storage.Storage) error {
	_ = sto.SetUint64ByUint64(gasPoolOffset, InitialGasPoolSeconds*InitialSpeedLimitPerSecond)
	_ = sto.SetUint64ByUint64(gasPoolLastBlockOffset, InitialGasPoolSeconds*InitialSpeedLimitPerSecond)
	_ = sto.SetUint64ByUint64(gasPoolSecondsOffset, InitialGasPoolSeconds)
	_ = sto.SetUint64ByUint64(gasPoolTargetOffset, InitialGasPoolTarget)
	_ = sto.SetUint64ByUint64(gasPoolWeightOffset, InitialGasPoolWeight)
	_ = sto.SetUint64ByUint64(rateEstimateOffset, InitialSpeedLimitPerSecond)
	_ = sto.SetUint64ByUint64(rateEstimateInertiaOffset, InitialRateEstimateInertia)
	_ = sto.SetUint64ByUint64(speedLimitPerSecondOffset, InitialSpeedLimitPerSecond)
	_ = sto.SetUint64ByUint64(maxPerBlockGasLimitOffset, InitialPerBlockGasLimit)
	_ = sto.SetUint64ByUint64(gasPriceWeiOffset, InitialBaseFeeWei)
	return sto.SetUint64ByUint64(minGasPriceWeiOffset, InitialMinimumGasPriceWei)
}

func OpenL2PricingState(sto *storage.Storage) *L2PricingState {
	return &L2PricingState{
		sto,
		sto.OpenStorageBackedInt64(gasPoolOffset),
		sto.OpenStorageBackedInt64(gasPoolLastBlockOffset),
		sto.OpenStorageBackedUint64(gasPoolSecondsOffset),
		sto.OpenStorageBackedUint64(gasPoolTargetOffset),
		sto.OpenStorageBackedUint64(gasPoolWeightOffset),
		sto.OpenStorageBackedUint64(rateEstimateOffset),
		sto.OpenStorageBackedUint64(rateEstimateInertiaOffset),
		sto.OpenStorageBackedUint64(speedLimitPerSecondOffset),
		sto.OpenStorageBackedUint64(maxPerBlockGasLimitOffset),
		sto.OpenStorageBackedBigInt(gasPriceWeiOffset),
		sto.OpenStorageBackedBigInt(minGasPriceWeiOffset),
	}
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
	if seconds == 0 || seconds > 3*60*60 || util.SaturatingUMul(seconds, limit) > math.MaxInt64 {
		return errors.New("GasPoolSeconds is out of bounds")
	}
	if err := ps.clipGasPool(seconds, limit); err != nil {
		return err
	}
	return ps.gasPoolSeconds.Set(seconds)
}

func (ps *L2PricingState) GasPoolTarget() (uint64, error) {
	return ps.gasPoolTarget.Get()
}

func (ps *L2PricingState) SetGasPoolTarget(target uint64) error {
	if target > 10000 {
		return errors.New("GasPoolTarget is out of bounds")
	}
	return ps.gasPoolTarget.Set(target)
}

func (ps *L2PricingState) GasPoolWeight() (uint64, error) {
	return ps.gasPoolWeight.Get()
}

func (ps *L2PricingState) SetGasPoolWeight(weight uint64) error {
	if weight > 10000 {
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

func (ps *L2PricingState) GasPriceWei() (*big.Int, error) {
	return ps.gasPriceWei.Get()
}

func (ps *L2PricingState) SetGasPriceWei(val *big.Int) error {
	return ps.gasPriceWei.Set(val)
}

func (ps *L2PricingState) MinGasPriceWei() (*big.Int, error) {
	return ps.minGasPriceWei.Get()
}

func (ps *L2PricingState) SetMinGasPriceWei(val *big.Int) error {
	err := ps.minGasPriceWei.Set(val)
	if err != nil {
		return err
	}

	// Check if the current gas price is below the new minimum.
	curGasPrice, err := ps.gasPriceWei.Get()
	if err != nil {
		return err
	}
	if util.BigLessThan(curGasPrice, val) {
		// The current gas price is less than the new minimum. Override it.
		return ps.gasPriceWei.Set(val)
	} else {
		// The current gas price is greater than the new minimum. Ignore it.
		return nil
	}
}

func (ps *L2PricingState) SpeedLimitPerSecond() (uint64, error) {
	return ps.speedLimitPerSecond.Get()
}

func (ps *L2PricingState) SetSpeedLimitPerSecond(limit uint64) error {
	seconds, err := ps.GasPoolSeconds()
	if err != nil {
		return err
	}
	if limit == 0 || util.SaturatingUMul(seconds, limit) > math.MaxInt64 {
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
	return util.SaturatingCast(seconds * speedLimit), nil
}

func (ps *L2PricingState) MaxPerBlockGasLimit() (uint64, error) {
	return ps.maxPerBlockGasLimit.Get()
}

func (ps *L2PricingState) SetMaxPerBlockGasLimit(limit uint64) error {
	return ps.maxPerBlockGasLimit.Set(limit)
}

// Ensure the gas pool is within the implied maximum capacity
func (ps *L2PricingState) clipGasPool(seconds, speedLimit uint64) error {
	pool, err := ps.GasPool()
	if err != nil {
		return err
	}
	newMax := util.SaturatingCast(util.SaturatingUMul(seconds, speedLimit))
	if pool > newMax {
		err = ps.SetGasPool(newMax)
	}
	return err
}

func (ps *L2PricingState) Restrict(err error) {
	ps.storage.Burner().Restrict(err)
}
