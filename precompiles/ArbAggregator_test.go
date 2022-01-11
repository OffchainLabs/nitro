//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestDefaultAggregator(t *testing.T) {
	evm := newMockEVMForTesting(t)
	agg := ArbAggregator{}
	context := testContext(common.Address{}, evm)

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	// initial default aggregator should be zero address
	def, err := agg.GetDefaultAggregator(context, evm)
	Require(t, err)
	if def != (common.Address{}) {
		t.Fatal()
	}

	// set default aggregator to addr
	Require(t, agg.SetDefaultAggregator(context, evm, addr))

	// default aggregator should now be addr
	res, err := agg.GetDefaultAggregator(context, evm)
	Require(t, err)
	if res != addr {
		t.Fatal()
	}
}

func TestPreferredAggregator(t *testing.T) {
	evm := newMockEVMForTesting(t)
	agg := ArbAggregator{}

	userAddr := common.BytesToAddress(crypto.Keccak256([]byte{0})[:20])
	defaultAggAddr := common.BytesToAddress(crypto.Keccak256([]byte{1})[:20])
	prefAggAddr := common.BytesToAddress(crypto.Keccak256([]byte{2})[:20])

	callerCtx := testContext(common.Address{}, evm)
	userCtx := testContext(userAddr, evm)

	// initial preferred aggregator should be the default of zero address
	res, isNonDefault, err := agg.GetPreferredAggregator(callerCtx, evm, userAddr)
	Require(t, err)
	if isNonDefault {
		t.Fatal()
	}
	if res != (common.Address{}) {
		t.Fatal()
	}

	// set default aggregator
	if err := agg.SetDefaultAggregator(callerCtx, evm, defaultAggAddr); err != nil {
		t.Fatal(err)
	}

	// preferred aggregator should be the new default address
	res, isNonDefault, err = agg.GetPreferredAggregator(callerCtx, evm, userAddr)
	Require(t, err)
	if isNonDefault {
		t.Fatal()
	}
	if res != defaultAggAddr {
		t.Fatal()
	}

	// set preferred aggregator
	Require(t, agg.SetPreferredAggregator(userCtx, evm, prefAggAddr))

	// preferred aggregator should now be prefAggAddr
	res, isNonDefault, err = agg.GetPreferredAggregator(callerCtx, evm, userAddr)
	Require(t, err)
	if !isNonDefault {
		t.Fatal()
	}
	if res != prefAggAddr {
		t.Fatal()
	}
}

func TestFeeCollector(t *testing.T) {
	evm := newMockEVMForTesting(t)
	agg := ArbAggregator{}

	aggAddr := common.BytesToAddress(crypto.Keccak256([]byte{0})[:20])
	collectorAddr := common.BytesToAddress(crypto.Keccak256([]byte{1})[:20])
	impostorAddr := common.BytesToAddress(crypto.Keccak256([]byte{2})[:20])

	aggCtx := testContext(aggAddr, evm)
	callerCtx := testContext(common.Address{}, evm)
	collectorCtx := testContext(collectorAddr, evm)
	imposterCtx := testContext(impostorAddr, evm)

	// initial result should be addr
	coll, err := agg.GetFeeCollector(callerCtx, evm, aggAddr)
	Require(t, err)
	if coll != aggAddr {
		t.Fatal()
	}

	// set fee collector to collectorAddr
	Require(t, agg.SetFeeCollector(aggCtx, evm, aggAddr, collectorAddr))

	// fee collector should now be collectorAddr
	coll, err = agg.GetFeeCollector(callerCtx, evm, aggAddr)
	Require(t, err)
	if coll != collectorAddr {
		t.Fatal()
	}

	// trying to set someone else's collector is an error
	shouldErr := agg.SetFeeCollector(imposterCtx, evm, aggAddr, impostorAddr)
	if shouldErr == nil {
		t.Fatal()
	}

	// but the fee collector can replace itself
	Require(t, agg.SetFeeCollector(collectorCtx, evm, aggAddr, impostorAddr))
}

func TestTxBaseFee(t *testing.T) {
	evm := newMockEVMForTesting(t)
	agg := ArbAggregator{}

	aggAddr := common.BytesToAddress(crypto.Keccak256([]byte{0})[:20])
	targetFee := big.NewInt(973)

	aggCtx := testContext(aggAddr, evm)
	callerCtx := testContext(common.Address{}, evm)

	// initial result should be zero
	fee, err := agg.GetTxBaseFee(callerCtx, evm, aggAddr)
	Require(t, err)
	if fee.Cmp(big.NewInt(0)) != 0 {
		t.Fatal()
	}

	// set base fee to value
	if err := agg.SetTxBaseFee(aggCtx, evm, aggAddr, targetFee); err != nil {
		t.Fatal(err)
	}

	// base fee should now be targetFee
	fee, err = agg.GetTxBaseFee(callerCtx, evm, aggAddr)
	Require(t, err)
	if fee.Cmp(targetFee) != 0 {
		t.Fatal(fee, targetFee)
	}
}
