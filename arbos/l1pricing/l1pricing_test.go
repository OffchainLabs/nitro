// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package l1pricing

import (
	"testing"

	am "github.com/offchainlabs/nitro/util/arbmath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
)

func TestL1PriceUpdate(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	err := InitializeL1PricingState(sto, common.Address{})
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
