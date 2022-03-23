// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
)

func TestDefaultAggregator(t *testing.T) {
	evm := newMockEVMForTesting()
	context := testContext(common.Address{}, evm)

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	// initial default aggregator should be zero address
	def, err := ArbAggregator{}.GetDefaultAggregator(context, evm)
	Require(t, err)
	if def != (l1pricing.SequencerAddress) {
		Fail(t)
	}

	// set default aggregator to addr
	Require(t, ArbDebug{}.BecomeChainOwner(context, evm))
	Require(t, ArbAggregator{}.SetDefaultAggregator(context, evm, addr))

	// default aggregator should now be addr
	res, err := ArbAggregator{}.GetDefaultAggregator(context, evm)
	Require(t, err)
	if res != addr {
		Fail(t)
	}
}

func TestPreferredAggregator(t *testing.T) {
	evm := newMockEVMForTesting()
	agg := ArbAggregator{}

	userAddr := common.BytesToAddress(crypto.Keccak256([]byte{0})[:20])
	defaultAggAddr := common.BytesToAddress(crypto.Keccak256([]byte{1})[:20])
	prefAggAddr := common.BytesToAddress(crypto.Keccak256([]byte{2})[:20])

	callerCtx := testContext(common.Address{}, evm)
	userCtx := testContext(userAddr, evm)

	// initial preferred aggregator should be the default of zero address
	res, isDefault, err := ArbAggregator{}.GetPreferredAggregator(callerCtx, evm, userAddr)
	Require(t, err)
	if !isDefault {
		Fail(t)
	}
	if res != (l1pricing.SequencerAddress) {
		Fail(t)
	}

	// set default aggregator
	Require(t, ArbDebug{}.BecomeChainOwner(callerCtx, evm))
	Require(t, agg.SetDefaultAggregator(callerCtx, evm, defaultAggAddr))

	// preferred aggregator should be the new default address
	res, isDefault, err = agg.GetPreferredAggregator(callerCtx, evm, userAddr)
	Require(t, err)
	if !isDefault {
		Fail(t)
	}
	if res != defaultAggAddr {
		Fail(t)
	}

	// set preferred aggregator
	Require(t, agg.SetPreferredAggregator(userCtx, evm, prefAggAddr))

	// preferred aggregator should now be prefAggAddr
	res, isDefault, err = agg.GetPreferredAggregator(callerCtx, evm, userAddr)
	Require(t, err)
	if isDefault {
		Fail(t)
	}
	if res != prefAggAddr {
		Fail(t)
	}
}

func TestFeeCollector(t *testing.T) {
	evm := newMockEVMForTesting()
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
		Fail(t)
	}

	// set fee collector to collectorAddr
	Require(t, agg.SetFeeCollector(aggCtx, evm, aggAddr, collectorAddr))

	// fee collector should now be collectorAddr
	coll, err = agg.GetFeeCollector(callerCtx, evm, aggAddr)
	Require(t, err)
	if coll != collectorAddr {
		Fail(t)
	}

	// trying to set someone else's collector is an error
	shouldErr := agg.SetFeeCollector(imposterCtx, evm, aggAddr, impostorAddr)
	if shouldErr == nil {
		Fail(t)
	}

	// but the fee collector can replace itself
	Require(t, agg.SetFeeCollector(collectorCtx, evm, aggAddr, impostorAddr))
}

func TestTxBaseFee(t *testing.T) {
	evm := newMockEVMForTesting()
	agg := ArbAggregator{}

	aggAddr := common.BytesToAddress(crypto.Keccak256([]byte{0})[:20])
	targetFee := big.NewInt(973)

	aggCtx := testContext(aggAddr, evm)
	callerCtx := testContext(common.Address{}, evm)

	// initial result should be zero
	fee, err := agg.GetTxBaseFee(callerCtx, evm, aggAddr)
	Require(t, err)
	if fee.Cmp(big.NewInt(0)) != 0 {
		Fail(t, fee)
	}

	// set base fee to value -- should be ignored
	if err := agg.SetTxBaseFee(aggCtx, evm, aggAddr, targetFee); err != nil {
		Fail(t, err)
	}

	// base fee should still be zero
	fee, err = agg.GetTxBaseFee(callerCtx, evm, aggAddr)
	Require(t, err)
	if fee.Cmp(big.NewInt(0)) != 0 {
		Fail(t, fee)
	}
}
