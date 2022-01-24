//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package l2pricing

import (
	"math/big"

	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/util"
)

type L2PricingState struct {
	storage             *storage.Storage
	gasPool             storage.StorageBackedInt64
	smallGasPool        storage.StorageBackedInt64
	gasPoolSeconds      storage.StorageBackedUint64
	smallGasPoolSeconds storage.StorageBackedUint64
	gasPriceWei         storage.StorageBackedBigInt
	minGasPriceWei      storage.StorageBackedBigInt
	maxGasPriceWei      storage.StorageBackedBigInt // the maximum price ArbOS can set without breaking geth
	speedLimitPerSecond storage.StorageBackedUint64
	maxPerBlockGasLimit storage.StorageBackedUint64
}

const (
	gasPoolOffset uint64 = iota
	smallGasPoolOffset
	gasPoolSecondsOffset
	smallGasPoolSecondsOffset
	gasPriceWeiOffset
	minGasPriceWeiOffset
	maxGasPriceWeiOffset
	speedLimitPerSecondOffset
	maxPerBlockGasLimitOffset
)

const L2GasLimit = 1 << 63

func InitializeL2PricingState(sto *storage.Storage) error {
	_ = sto.SetUint64ByUint64(gasPoolOffset, InitialGasPoolSeconds*InitialSpeedLimitPerSecond)
	_ = sto.SetUint64ByUint64(smallGasPoolOffset, InitialSmallGasPoolSeconds*InitialSpeedLimitPerSecond)
	_ = sto.SetUint64ByUint64(gasPoolSecondsOffset, InitialGasPoolSeconds)
	_ = sto.SetUint64ByUint64(smallGasPoolSecondsOffset, InitialSmallGasPoolSeconds)
	_ = sto.SetUint64ByUint64(gasPriceWeiOffset, InitialGasPriceWei)
	_ = sto.SetUint64ByUint64(minGasPriceWeiOffset, InitialMinimumGasPriceWei)
	_ = sto.SetUint64ByUint64(maxGasPriceWeiOffset, 2*InitialGasPriceWei)
	_ = sto.SetUint64ByUint64(speedLimitPerSecondOffset, InitialSpeedLimitPerSecond)
	return sto.SetUint64ByUint64(maxPerBlockGasLimitOffset, InitialPerBlockGasLimit)
}

func OpenL2PricingState(sto *storage.Storage) *L2PricingState {
	return &L2PricingState{
		sto,
		sto.OpenStorageBackedInt64(gasPoolOffset),
		sto.OpenStorageBackedInt64(smallGasPoolOffset),
		sto.OpenStorageBackedUint64(gasPoolSecondsOffset),
		sto.OpenStorageBackedUint64(smallGasPoolSecondsOffset),
		sto.OpenStorageBackedBigInt(gasPriceWeiOffset),
		sto.OpenStorageBackedBigInt(minGasPriceWeiOffset),
		sto.OpenStorageBackedBigInt(maxGasPriceWeiOffset),
		sto.OpenStorageBackedUint64(speedLimitPerSecondOffset),
		sto.OpenStorageBackedUint64(maxPerBlockGasLimitOffset),
	}
}

func (ps *L2PricingState) GasPool() (int64, error) {
	return ps.gasPool.Get()
}

func (ps *L2PricingState) SetGasPool(val int64) error {
	return ps.gasPool.Set(val)
}

func (ps *L2PricingState) SmallGasPool() (int64, error) {
	return ps.smallGasPool.Get()
}

func (ps *L2PricingState) SetSmallGasPool(val int64) error {
	return ps.smallGasPool.Set(val)
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

func (ps *L2PricingState) SetMinGasPriceWei(val *big.Int) {
	ps.Restrict(ps.minGasPriceWei.Set(val))
}

func (ps *L2PricingState) MaxGasPriceWei() (*big.Int, error) { // the max gas price ArbOS can set without breaking geth
	return ps.maxGasPriceWei.Get()
}

func (ps *L2PricingState) SetMaxGasPriceWei(val *big.Int) {
	ps.Restrict(ps.maxGasPriceWei.Set(val))
}

func (ps *L2PricingState) SpeedLimitPerSecond() (uint64, error) {
	return ps.speedLimitPerSecond.Get()
}

func (ps *L2PricingState) SetSpeedLimitPerSecond(speedLimit uint64) error {
	return ps.speedLimitPerSecond.Set(speedLimit)
}

func (ps *L2PricingState) GasPoolSeconds() (uint64, error) {
	return ps.gasPoolSeconds.Get()
}

func (ps *L2PricingState) SetGasPoolSeconds(seconds uint64) error {
	return ps.gasPoolSeconds.Set(seconds)
}

func (ps *L2PricingState) SmallGasPoolSeconds() (uint64, error) {
	return ps.smallGasPoolSeconds.Get()
}

func (ps *L2PricingState) SetSmallGasPoolSeconds(seconds uint64) error {
	return ps.smallGasPoolSeconds.Set(seconds)
}

func (ps *L2PricingState) GasPoolMax() (int64, error) {
	speedLimit, _ := ps.SpeedLimitPerSecond()
	seconds, err := ps.GasPoolSeconds()
	if err != nil {
		return 0, err
	}
	return util.SaturatingCast(seconds * speedLimit), nil
}

func (ps *L2PricingState) SmallGasPoolMax() (int64, error) {
	speedLimit, _ := ps.SpeedLimitPerSecond()
	seconds, err := ps.SmallGasPoolSeconds()
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

func (ps *L2PricingState) Restrict(err error) {
	ps.storage.Burner().Restrict(err)
}
