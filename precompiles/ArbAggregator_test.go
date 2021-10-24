//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"testing"
)

func TestDefaultAggregator(t *testing.T) {
	caller := common.Address{}
	st := newStateDBForTesting()
	agg := ArbAggregator{}

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	// initial default aggregator should be zero address
	def, err := agg.GetDefaultAggregator(caller, st)
	if err != nil {
		t.Fatal(err)
	}
	if def != (common.Address{}) {
		t.Fatal()
	}

	// set default aggregator to addr
	if err := agg.SetDefaultAggregator(caller, st, addr); err != nil {
		t.Fatal(err)
	}

	// default aggregator should now be addr
	res, err := agg.GetDefaultAggregator(caller, st)
	if err != nil {
		t.Fatal(err)
	}
	if res != addr {
		t.Fatal()
	}
}

func TestPreferredAggregator(t *testing.T) {
	caller := common.Address{}
	st := newStateDBForTesting()
	agg := ArbAggregator{}

	userAddr := common.BytesToAddress(crypto.Keccak256([]byte{0})[:20])
	defaultAggAddr := common.BytesToAddress(crypto.Keccak256([]byte{1})[:20])
	prefAggAddr := common.BytesToAddress(crypto.Keccak256([]byte{2})[:20])

	// initial preferred aggregator should be the default of zero address
	res, isNonDefault, err := agg.GetPreferredAggregator(caller, st, userAddr)
	if err != nil {
		t.Fatal(err)
	}
	if isNonDefault {
		t.Fatal()
	}
	if res != (common.Address{}) {
		t.Fatal()
	}

	// set default aggregator
	if err := agg.SetDefaultAggregator(caller, st, defaultAggAddr); err != nil {
		t.Fatal(err)
	}

	// preferred aggregator should be the new default address
	res, isNonDefault, err = agg.GetPreferredAggregator(caller, st, userAddr)
	if err != nil {
		t.Fatal(err)
	}
	if isNonDefault {
		t.Fatal()
	}
	if res != defaultAggAddr {
		t.Fatal()
	}

	// set preferred aggregator
	if err := agg.SetPreferredAggregator(userAddr, st, prefAggAddr); err != nil {
		t.Fatal(err)
	}

	// preferred aggregator should now be prefAggAddr
	res, isNonDefault, err = agg.GetPreferredAggregator(caller, st, userAddr)
	if err != nil {
		t.Fatal(err)
	}
	if !isNonDefault {
		t.Fatal()
	}
	if res != prefAggAddr {
		t.Fatal()
	}
}

func TestFeeCollector(t *testing.T) {
	caller := common.Address{}
	st := newStateDBForTesting()
	agg := ArbAggregator{}

	aggAddr := common.BytesToAddress(crypto.Keccak256([]byte{0})[:20])
	collectorAddr := common.BytesToAddress(crypto.Keccak256([]byte{1})[:20])
	impostorAddr := common.BytesToAddress(crypto.Keccak256([]byte{2})[:20])

	// initial result should be addr
	coll, err := agg.GetFeeCollector(caller, st, aggAddr)
	if err != nil {
		t.Fatal(err)
	}
	if coll != aggAddr {
		t.Fatal()
	}

	// set fee collector to collectorAddr
	if err := agg.SetFeeCollector(aggAddr, st, aggAddr, collectorAddr); err != nil {
		t.Fatal(err)
	}

	// fee collector should now be collectorAddr
	coll, err = agg.GetFeeCollector(caller, st, aggAddr)
	if err != nil {
		t.Fatal(err)
	}
	if coll != collectorAddr {
		t.Fatal()
	}

	// trying to set someone else's collector is an error
	err = agg.SetFeeCollector(impostorAddr, st, aggAddr, impostorAddr)
	if err == nil {
		t.Fatal()
	}

	// but the fee collector can replace itself
	err = agg.SetFeeCollector(collectorAddr, st, aggAddr, impostorAddr)
	if err != nil {
		t.Fatal()
	}
}

func TestTxBaseFee(t *testing.T) {
	caller := common.Address{}
	st := newStateDBForTesting()
	agg := ArbAggregator{}

	aggAddr := common.BytesToAddress(crypto.Keccak256([]byte{0})[:20])
	targetFee := big.NewInt(973)

	// initial result should be zero
	fee, err := agg.GetTxBaseFee(caller, st, aggAddr)
	if err != nil {
		t.Fatal(err)
	}
	if fee.Cmp(big.NewInt(0)) != 0 {
		t.Fatal()
	}

	// set base fee to value
	if err := agg.SetTxBaseFee(aggAddr, st, aggAddr, targetFee); err != nil {
		t.Fatal(err)
	}

	// base fee should now be targetFee
	fee, err = agg.GetTxBaseFee(caller, st, aggAddr)
	if err != nil {
		t.Fatal(err)
	}
	if fee.Cmp(targetFee) != 0 {
		t.Fatal(fee, targetFee)
	}
}
