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
	err := InitializeL1PricingState(sto, common.Address{}, 11)
	Require(t, err)
	ps := OpenL1PricingState(sto)

	tyme, err := ps.LastUpdateTime()
	Require(t, err)
	if tyme != 0 {
		Fail(t)
	}

	initialPriceEstimate := am.UintToBig(InitialBasePricePerUnitWei)
	priceEstimate, err := ps.BasePricePerUnit()
	Require(t, err)
	if priceEstimate.Cmp(initialPriceEstimate) != 0 {
		Fail(t)
	}
}

func TestL1PriceVelocityControl(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	err := InitializeL1PricingState(sto, common.Address{}, 11)
	Require(t, err)
	ps := OpenL1PricingState(sto)

	initialPrice, err := ps.EffectiveL1UnitPrice(11)
	Require(t, err)
	Require(t, ps.AddToUnitsSinceUpdate(100000, 0))
	backlog, err := ps.L1UnitsBacklog()
	Require(t, err)
	if backlog != 100000 {
		Fail(t, backlog)
	}
	lastUpdate, err := ps.L1LastBacklogUpdate()
	Require(t, err)
	if lastUpdate != 0 {
		Fail(t, lastUpdate)
	}
	newPrice, err := ps.EffectiveL1UnitPrice(11)
	Require(t, err)
	if newPrice.Cmp(initialPrice) <= 0 {
		Fail(t)
	}
	Require(t, ps.AddToUnitsSinceUpdate(0, 1))
	backlog, err = ps.L1UnitsBacklog()
	Require(t, err)
	if backlog != 0 {
		Fail(t, backlog)
	}
	lastUpdate, err = ps.L1LastBacklogUpdate()
	Require(t, err)
	if lastUpdate != 1 {
		Fail(t, lastUpdate)
	}
	newPrice, err = ps.EffectiveL1UnitPrice(11)
	Require(t, err)
	if newPrice.Cmp(initialPrice) != 0 {
		Fail(t, newPrice, initialPrice)
	}
}
