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
	evm := newMockEVMForTesting(t)
	agg := ArbAggregator{}

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	// initial default aggregator should be zero address
	def, err := agg.GetDefaultAggregator(fakeBurnGas, caller, evm)
	if err != nil {
		t.Fatal(err)
	}
	if def != (common.Address{}) {
		t.Fatal()
	}

	// set default aggregator to addr
	if err := agg.SetDefaultAggregator(fakeBurnGas, caller, evm, addr); err != nil {
		t.Fatal(err)
	}

	// default aggregator should now be addr
	res, err := agg.GetDefaultAggregator(fakeBurnGas, caller, evm)
	if err != nil {
		t.Fatal(err)
	}
	if res != addr {
		t.Fatal()
	}
}

func TestPreferredAggregator(t *testing.T) {
	caller := common.Address{}
	evm := newMockEVMForTesting(t)
	agg := ArbAggregator{}

	userAddr := common.BytesToAddress(crypto.Keccak256([]byte{0})[:20])
	defaultAggAddr := common.BytesToAddress(crypto.Keccak256([]byte{1})[:20])
	prefAggAddr := common.BytesToAddress(crypto.Keccak256([]byte{2})[:20])

	// initial preferred aggregator should be the default of zero address
	res, isNonDefault, err := agg.GetPreferredAggregator(fakeBurnGas, caller, evm, userAddr)
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
	if err := agg.SetDefaultAggregator(fakeBurnGas, caller, evm, defaultAggAddr); err != nil {
		t.Fatal(err)
	}

	// preferred aggregator should be the new default address
	res, isNonDefault, err = agg.GetPreferredAggregator(fakeBurnGas, caller, evm, userAddr)
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
	if err := agg.SetPreferredAggregator(fakeBurnGas, userAddr, evm, prefAggAddr); err != nil {
		t.Fatal(err)
	}

	// preferred aggregator should now be prefAggAddr
	res, isNonDefault, err = agg.GetPreferredAggregator(fakeBurnGas, caller, evm, userAddr)
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
	evm := newMockEVMForTesting(t)
	agg := ArbAggregator{}

	aggAddr := common.BytesToAddress(crypto.Keccak256([]byte{0})[:20])
	collectorAddr := common.BytesToAddress(crypto.Keccak256([]byte{1})[:20])
	impostorAddr := common.BytesToAddress(crypto.Keccak256([]byte{2})[:20])

	// initial result should be addr
	coll, err := agg.GetFeeCollector(fakeBurnGas, caller, evm, aggAddr)
	if err != nil {
		t.Fatal(err)
	}
	if coll != aggAddr {
		t.Fatal()
	}

	// set fee collector to collectorAddr
	if err := agg.SetFeeCollector(fakeBurnGas, aggAddr, evm, aggAddr, collectorAddr); err != nil {
		t.Fatal(err)
	}

	// fee collector should now be collectorAddr
	coll, err = agg.GetFeeCollector(fakeBurnGas, caller, evm, aggAddr)
	if err != nil {
		t.Fatal(err)
	}
	if coll != collectorAddr {
		t.Fatal()
	}

	// trying to set someone else's collector is an error
	err = agg.SetFeeCollector(fakeBurnGas, impostorAddr, evm, aggAddr, impostorAddr)
	if err == nil {
		t.Fatal()
	}

	// but the fee collector can replace itself
	err = agg.SetFeeCollector(fakeBurnGas, collectorAddr, evm, aggAddr, impostorAddr)
	if err != nil {
		t.Fatal()
	}
}

func TestTxBaseFee(t *testing.T) {
	caller := common.Address{}
	evm := newMockEVMForTesting(t)
	agg := ArbAggregator{}

	aggAddr := common.BytesToAddress(crypto.Keccak256([]byte{0})[:20])
	targetFee := big.NewInt(973)

	// initial result should be zero
	fee, err := agg.GetTxBaseFee(fakeBurnGas, caller, evm, aggAddr)
	if err != nil {
		t.Fatal(err)
	}
	if fee.Cmp(big.NewInt(0)) != 0 {
		t.Fatal()
	}

	// set base fee to value
	if err := agg.SetTxBaseFee(fakeBurnGas, aggAddr, evm, aggAddr, targetFee); err != nil {
		t.Fatal(err)
	}

	// base fee should now be targetFee
	fee, err = agg.GetTxBaseFee(fakeBurnGas, caller, evm, aggAddr)
	if err != nil {
		t.Fatal(err)
	}
	if fee.Cmp(targetFee) != 0 {
		t.Fatal(fee, targetFee)
	}
}
