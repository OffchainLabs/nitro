//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestTips(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2info, _, l2client, l1info, _, l1client, stack := CreateTestNodeOnL1(t, ctx, true)
	defer stack.Close()

	auth := l2info.GetDefaultTransactOpts("Owner")
	callOpts := l2info.GetDefaultCallOpts("Owner")
	aggregator := testhelpers.RandomAddress()

	// get the network fee account
	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(common.HexToAddress("0x6b"), l2client)
	Require(t, err, "could not deploy ArbOwner contract")
	networkFeeAccount, err := arbOwnerPublic.GetNetworkFeeAccount(callOpts)
	Require(t, err, "could not get the network fee account")

	// set a preferred aggregator who won't be the one to post the tx
	arbAggregator, err := precompilesgen.NewArbAggregator(common.HexToAddress("0x6d"), l2client)
	Require(t, err, "could not deploy ArbAggregator contract")
	tx, err := arbAggregator.SetPreferredAggregator(&auth, aggregator)
	Require(t, err, "could not set L2 gas price")
	_, err = arbutil.EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	basefee := GetBaseFee(t, l2client, ctx)
	auth.GasFeeCap = util.BigMulByUfrac(basefee, 5, 4) // add room for a 20% tip
	auth.GasTipCap = util.BigMulByUfrac(basefee, 1, 4) // add a 20% tip

	networkBefore := GetBalance(t, ctx, l2client, networkFeeAccount)

	// use L1 to post a message since the sequencer won't do it
	nosend := auth
	nosend.NoSend = true
	tx, err = arbAggregator.SetPreferredAggregator(&nosend, aggregator)
	Require(t, err)
	receipt := SendSignedTxViaL1(t, ctx, l1info, l1client, l2client, tx)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fail(t, "failed to prefer the sequencer")
	}

	networkAfter := GetBalance(t, ctx, l2client, networkFeeAccount)
	colors.PrintMint("network: ", networkFeeAccount, networkBefore, networkAfter)
	colors.PrintBlue("pricing: ", l2info.GasPrice, auth.GasFeeCap, auth.GasTipCap)
	colors.PrintBlue("payment: ", tx.GasPrice(), tx.GasFeeCap(), tx.GasTipCap())

	if !util.BigEquals(tx.GasPrice(), auth.GasFeeCap) {
		Fail(t, "user did not pay the tip")
	}

	tip := util.BigMulByUint(util.BigSub(tx.GasPrice(), basefee), receipt.GasUsed)
	full := util.BigMulByUint(tx.GasPrice(), receipt.GasUsed)
	networkRevenue := util.BigSub(networkAfter, networkBefore)
	colors.PrintMint("tip: ", tip, full, networkRevenue)

	colors.PrintRed("used: ", receipt.GasUsed, basefee)

	if !util.BigEquals(tip, util.BigMulByFrac(networkRevenue, 1, 5)) {
		Fail(t, "1/5th of the network's revenue should be the tip")
	}
	if !util.BigEquals(full, networkRevenue) {
		Fail(t, "the network didn't receive the tip")
	}
}

// Test that the sequencer won't subvert a user's aggregation preferences
func TestSequencerWontPostWhenNotPreferred(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2info, _, client := CreateTestL2(t, ctx)
	auth := l2info.GetDefaultTransactOpts("Owner")

	// prefer a 3rd party aggregator
	arbAggregator, err := precompilesgen.NewArbAggregator(common.HexToAddress("0x6d"), client)
	Require(t, err, "could not deploy ArbAggregator contract")
	tx, err := arbAggregator.SetPreferredAggregator(&auth, testhelpers.RandomAddress())
	Require(t, err, "could not set L2 gas price")
	_, err = arbutil.EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	// get the network fee account
	_, err = arbAggregator.SetPreferredAggregator(&auth, testhelpers.RandomAddress())
	colors.PrintBlue("expecting error: ", err)
	if err == nil {
		Fail(t, "the sequencer should have rejected this tx")
	}
}
