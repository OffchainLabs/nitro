// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l1pricing

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
)

func TestL1PriceUpdate(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	initialPriceEstimate := big.NewInt(123 * params.GWei)
	err := InitializeL1PricingState(sto, common.Address{}, initialPriceEstimate)
	Require(t, err)
	ps := OpenL1PricingState(sto)

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
