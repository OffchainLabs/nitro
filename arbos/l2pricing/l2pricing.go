//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package l2pricing

import (
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
	"math/big"
)

type L2PricingState struct {
	storage            *storage.Storage
	price              *storage.StorageBackedUbig
	gasPool            *storage.StorageBackedInt64
	smallGasPool       *storage.StorageBackedInt64
	storageLoadLimiter *LoadLimiter
}

const (
	gasPoolOffset uint64 = iota
	priceOffset
	smallGasPoolOffset
)

var storageLimiterKey = []byte{0}

func InitializeL2PricingState(sto *storage.Storage) {
	sto.OpenStorageBackedUbig(util.UintToHash(priceOffset)).Set(big.NewInt(InitialGasPriceWei))
	sto.OpenStorageBackedInt64(util.UintToHash(gasPoolOffset)).Set(GasPoolMax)
	sto.OpenStorageBackedInt64(util.UintToHash(smallGasPoolOffset)).Set(SmallGasPoolMax)
}

func OpenL2PricingState(sto *storage.Storage) *L2PricingState {
	return &L2PricingState{
		sto,
		sto.OpenStorageBackedUbig(util.UintToHash(priceOffset)),
		sto.OpenStorageBackedInt64(util.UintToHash(gasPoolOffset)),
		sto.OpenStorageBackedInt64(util.UintToHash(smallGasPoolOffset)),
		OpenLoadLimiter(sto.OpenSubStorage(storageLimiterKey), StorageLimiterParams),
	}
}

func (pricingState *L2PricingState) NotifyGasPricerThatTimeElapsed(secondsElapsed uint64) {
	startPrice := pricingState.Price()
	gasResult := pricingState.updateGasComponentForElapsedTime(secondsElapsed, startPrice)
	storageResult := pricingState.storageLoadLimiter.updateStorageComponentForElapsedTime(secondsElapsed, startPrice, gasResult)
	if gasResult.Cmp(storageResult) >= 0 {
		pricingState.SetPrice(gasResult)
	} else {
		pricingState.SetPrice(storageResult)
	}
}

func (pricingState *L2PricingState) NotifyStorageUsageChange(delta int64) {
	pricingState.storageLoadLimiter.NotifyUsageChange(delta)
}

func (pricingState *L2PricingState) GasPool() int64 {
	return pricingState.gasPool.Get()
}

func (pricingState *L2PricingState) SetGasPool(val int64) {
	pricingState.gasPool.Set(val)
}

func (pricingState *L2PricingState) SmallGasPool() int64 {
	return pricingState.smallGasPool.Get()
}

func (pricingState *L2PricingState) SetSmallGasPool(val int64) {
	pricingState.smallGasPool.Set(val)
}

func (pricingState *L2PricingState) Price() *big.Int {
	return pricingState.price.Get()
}

func (pricingState *L2PricingState) SetPrice(value *big.Int) {
	pricingState.price.Set(value)
}
