// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
)

func TestArbAggregatorBatchPosters(t *testing.T) {
	evm := newMockEVMForTesting()
	context := testContext(common.Address{}, evm)

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	// initially should have one batch poster
	bps, err := ArbAggregator{}.GetBatchPosters(context, evm)
	Require(t, err)
	if len(bps) != 1 {
		Fail(t)
	}

	// add addr as a batch poster
	Require(t, ArbDebug{}.BecomeChainOwner(context, evm))
	Require(t, ArbAggregator{}.AddBatchPoster(context, evm, addr))

	// there should now be two batch posters, and addr should be one of them
	bps, err = ArbAggregator{}.GetBatchPosters(context, evm)
	Require(t, err)
	if len(bps) != 2 {
		Fail(t)
	}
	if bps[0] != addr && bps[1] != addr {
		Fail(t)
	}
}

func TestFeeCollector(t *testing.T) {
	evm := newMockEVMForTesting()
	agg := ArbAggregator{}

	aggAddr := l1pricing.BatchPosterAddress
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
