// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l1pricing

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
)

func TestL1PriceUpdate(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	initialPriceEstimate := big.NewInt(123 * params.GWei)
	err := InitializeL1PricingState(sto, common.Address{}, initialPriceEstimate)
	Require(t, err)
	ps := OpenL1PricingState(sto, params.MaxDebugArbosVersionSupported)

	tyme, err := ps.LastUpdateTime()
	Require(t, err)
	if tyme != 0 {
		Fail(t)
	}

	priceEstimate, err := ps.PricePerUnit()
	Require(t, err)
	if priceEstimate.Cmp(initialPriceEstimate) != 0 {
		Fail(t)
	}
}

func Test_getPosterUnitsWithoutCache(t *testing.T) {
	depositorAddr := common.HexToAddress("0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF")
	txData := &types.ArbitrumDepositTx{
		From:  depositorAddr,
		To:    common.Address{},
		Value: big.NewInt(1),
	}
	tx := types.NewTx(types.TxData(txData))
	pricingState := &L1PricingState{}
	posterAddr := common.Address{}
	// Only txs that come from the batch poster address will be checked for a poster cost.
	units := pricingState.getPosterUnitsWithoutCache(tx, posterAddr, 11)
	if units != 0 {
		Fail(t)
	}

	// This can never happen in prod, but even if the batch poster sends a
	// deposit tx, the poster costs should still be 0.
	posterAddr = BatchPosterAddress
	// Only txs that come from the batch poster address will be checked for a poster cost.
	units = pricingState.getPosterUnitsWithoutCache(tx, posterAddr, 11)
	if units != 0 {
		Fail(t)
	}
}
