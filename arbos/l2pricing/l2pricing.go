//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package l2pricing

import (
	"math/big"

	"github.com/offchainlabs/arbstate/arbos/storage"
)

type L2PricingState struct {
	storage        *storage.Storage
	gasPool        storage.StorageBackedInt64
	smallGasPool   storage.StorageBackedInt64
	gasPriceWei    storage.StorageBackedBigInt
	maxGasPriceWei storage.StorageBackedBigInt // the maximum price ArbOS can set without breaking geth
}

const (
	gasPoolOffset uint64 = iota
	smallGasPoolOffset
	gasPriceWeiOffset
	maxGasPriceWeiOffset
)

func InitializeL2PricingState(sto *storage.Storage) error {
	_ = sto.SetUint64ByUint64(uint64(gasPoolOffset), GasPoolMax)
	_ = sto.SetUint64ByUint64(uint64(smallGasPoolOffset), SmallGasPoolMax)
	_ = sto.SetUint64ByUint64(uint64(gasPriceWeiOffset), InitialGasPriceWei)
	return sto.SetUint64ByUint64(uint64(maxGasPriceWeiOffset), 2*InitialGasPriceWei)
}

func OpenL2PricingState(sto *storage.Storage) *L2PricingState {
	return &L2PricingState{
		sto,
		sto.OpenStorageBackedInt64(gasPoolOffset),
		sto.OpenStorageBackedInt64(smallGasPoolOffset),
		sto.OpenStorageBackedBigInt(gasPriceWeiOffset),
		sto.OpenStorageBackedBigInt(maxGasPriceWeiOffset),
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

func (ps *L2PricingState) MaxGasPriceWei() (*big.Int, error) { // the max gas price ArbOS can set without breaking geth
	return ps.maxGasPriceWei.Get()
}

func (ps *L2PricingState) SetMaxGasPriceWei(val *big.Int) {
	ps.Restrict(ps.maxGasPriceWei.Set(val))
}

func (ps *L2PricingState) Restrict(err error) {
	ps.storage.Burner().Restrict(err)
}
