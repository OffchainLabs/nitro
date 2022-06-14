// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package l1pricing

import (
	am "github.com/offchainlabs/nitro/util/arbmath"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
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
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	err := InitializeL1PricingState(sto)
	Require(t, err)
	ps := OpenL1PricingState(sto)

	tyme, err := ps.LastUpdateTime()
	Require(t, err)
	if tyme != 0 {
		Fail(t)
	}

	initialPriceEstimate := am.UintToBig(InitialPricePerUnitWei)
	priceEstimate, err := ps.PricePerUnit()
	Require(t, err)
	if priceEstimate.Cmp(initialPriceEstimate) != 0 {
		Fail(t)
	}
}
