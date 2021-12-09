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
	storage      *storage.Storage
	gasPool      *storage.StorageBackedInt64
	smallGasPool *storage.StorageBackedInt64
	gasPriceWei  *big.Int
}

const (
	gasPoolOffset uint64 = iota
	smallGasPoolOffset
	gasPriceOffset
)

func InitializeL2PricingState(sto *storage.Storage) {
	sto.OpenStorageBackedInt64(util.UintToHash(gasPoolOffset)).Set(GasPoolMax)
	sto.OpenStorageBackedInt64(util.UintToHash(smallGasPoolOffset)).Set(SmallGasPoolMax)
	sto.SetByUint64(gasPriceOffset, util.IntToHash(InitialGasPriceWei))
}

func OpenL2PricingState(sto *storage.Storage) *L2PricingState {
	return &L2PricingState{
		sto,
		sto.OpenStorageBackedInt64(util.UintToHash(gasPoolOffset)),
		sto.OpenStorageBackedInt64(util.UintToHash(smallGasPoolOffset)),
		sto.GetByUint64(gasPriceOffset).Big(),
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

func (pricingState *L2PricingState) GasPriceWei() *big.Int {
	return pricingState.gasPriceWei
}

func (pricingState *L2PricingState) SetGasPriceWei(value *big.Int) {
	pricingState.gasPriceWei = value
	pricingState.storage.SetByUint64(gasPriceOffset, common.BigToHash(value))
}
