// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package l2pricing

import (
	"math/big"

	"github.com/offchainlabs/nitro/arbos/storage"
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
}

const (
	speedLimitPerSecondOffset uint64 = iota
	perBlockGasLimitOffset
	baseFeeWeiOffset
	minBaseFeeWeiOffset
	gasBacklogOffset
	pricingInertiaOffset
	backlogToleranceOffset
)

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
		sto,
		sto.OpenStorageBackedUint64(speedLimitPerSecondOffset),
		sto.OpenStorageBackedUint64(perBlockGasLimitOffset),
		sto.OpenStorageBackedBigUint(baseFeeWeiOffset),
		sto.OpenStorageBackedBigUint(minBaseFeeWeiOffset),
		sto.OpenStorageBackedUint64(gasBacklogOffset),
		sto.OpenStorageBackedUint64(pricingInertiaOffset),
		sto.OpenStorageBackedUint64(backlogToleranceOffset),
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
