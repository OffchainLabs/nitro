//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package l1pricing

import (
	"math"
	"math/big"
	"testing"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func TestTxFixedCost(t *testing.T) {
	maxChainId := new(big.Int).SetUint64(math.MaxUint64)
	maxValue := big.NewInt(1_000_000)
	maxValue.Mul(maxValue, big.NewInt(params.Ether))
	var address common.Address
	for i := range address {
		address[i] = 0xFF
	}
	maxSigVal := big.NewInt(2)
	maxSigVal.Exp(maxSigVal, big.NewInt(256), nil)
	maxSigVal.Sub(maxSigVal, common.Big1)
	maxGasPrice := big.NewInt(1000 * params.GWei)
	largeTx := types.NewTx(&types.DynamicFeeTx{
		ChainID:    maxChainId,
		Nonce:      1 << 32,
		GasTipCap:  maxGasPrice,
		GasFeeCap:  maxGasPrice,
		Gas:        100_000_000,
		To:         &address,
		Value:      maxValue,
		Data:       []byte{},
		AccessList: []types.AccessTuple{},
		V:          common.Big1,
		R:          maxSigVal,
		S:          maxSigVal,
	})
	largeTxEncoded, err := largeTx.MarshalBinary()
	Require(t, err)

	if len(largeTxEncoded) > TxFixedCost {
		Fail(t, "large tx is", len(largeTxEncoded), "bytes but tx fixed cost is", TxFixedCost)
	}
}

func TestL1PriceUpdate(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(false))
	err := InitializeL1PricingState(sto)
	Require(t, err)
	ps := OpenL1PricingState(sto)

	tyme, err := ps.LastL1BaseFeeUpdateTime()
	Require(t, err)
	if tyme != 0 {
		Fail(t)
	}

	priceEstimate, err := ps.L1BaseFeeEstimateWei()
	Require(t, err)
	if priceEstimate.Cmp(big.NewInt(InitialL1BaseFeeEstimate)) != 0 {
		Fail(t)
	}

	newPrice := big.NewInt(20 * params.GWei)
	ps.UpdatePricingModel(newPrice, 2)
	priceEstimate, err = ps.L1BaseFeeEstimateWei()
	Require(t, err)

	if priceEstimate.Cmp(newPrice) <= 0 || priceEstimate.Cmp(big.NewInt(InitialL1BaseFeeEstimate)) >= 0 {
		Fail(t)
	}

	ps.UpdatePricingModel(newPrice, uint64(1)<<63)
	priceEstimate, err = ps.L1BaseFeeEstimateWei()
	Require(t, err)

	priceLimit := new(big.Int).Add(newPrice, big.NewInt(300))
	if priceEstimate.Cmp(priceLimit) > 0 || priceEstimate.Cmp(newPrice) < 0 {
		Fail(t, priceEstimate)
	}
}
