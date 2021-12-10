//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package l2pricing

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
	"math/big"
)

type L2PricingState struct {
	storage        *storage.Storage
	price          *big.Int
	gasPool        *storage.StorageBackedInt64
	smallGasPool   *storage.StorageBackedInt64
	fastStorageAvg *storage.StorageBackedUint64
	slowStorageAvg *storage.StorageBackedUint64
}

const (
	gasPoolOffset uint64 = iota
	priceOffset
	smallGasPoolOffset
	fastStorageOffset
	slowStorageOffset
)

func InitializeL2PricingState(sto *storage.Storage) {
	sto.SetByUint64(priceOffset, common.BigToHash(big.NewInt(InitialGasPriceWei)))
	sto.OpenStorageBackedInt64(util.UintToHash(gasPoolOffset)).Set(GasPoolMax)
	sto.OpenStorageBackedInt64(util.UintToHash(smallGasPoolOffset)).Set(SmallGasPoolMax)
	sto.OpenStorageBackedUint64(util.UintToHash(fastStorageOffset)).Set(0)
	sto.OpenStorageBackedUint64(util.UintToHash(slowStorageOffset)).Set(0)
}

func OpenL2PricingState(sto *storage.Storage) *L2PricingState {
	return &L2PricingState{
		sto,
		sto.GetByUint64(priceOffset).Big(),
		sto.OpenStorageBackedInt64(util.UintToHash(gasPoolOffset)),
		sto.OpenStorageBackedInt64(util.UintToHash(smallGasPoolOffset)),
		sto.OpenStorageBackedUint64(util.UintToHash(fastStorageOffset)),
		sto.OpenStorageBackedUint64(util.UintToHash(slowStorageOffset)),
	}
}

func (pricingState *L2PricingState) NotifyGasPricerThatTimeElapsed(secondsElapsed uint64) {
	startPrice := pricingState.Price()
	gasResult := pricingState.updateGasComponentForElapsedTime(secondsElapsed, startPrice)
	storageResult := pricingState.updateStorageComponentForElapsedTime(secondsElapsed, startPrice, gasResult)
	if gasResult.Cmp(storageResult) >= 0 {
		pricingState.SetPrice(gasResult)
	} else {
		pricingState.SetPrice(storageResult)
	}
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
	return pricingState.price
}

func (pricingState *L2PricingState) SetPrice(value *big.Int) {
	pricingState.price = value
	pricingState.storage.SetByUint64(priceOffset, common.BigToHash(value))
}
