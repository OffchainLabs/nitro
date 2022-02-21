//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/solgen/go/mocksgen"
	"github.com/offchainlabs/arbstate/solgen/go/precompilesgen"
	"github.com/offchainlabs/arbstate/util"
	"github.com/offchainlabs/arbstate/util/colors"
	"github.com/offchainlabs/arbstate/util/testhelpers"
)

func TestTips(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2info, _, client := CreateTestL2(t, ctx)
	auth := l2info.GetDefaultTransactOpts("Owner")
	callOpts := l2info.GetDefaultCallOpts("Owner")
	aggregator := testhelpers.RandomAddress()

	// set a preferred aggregator who won't be the one to post the tx
	arbAggregator, err := precompilesgen.NewArbAggregator(common.HexToAddress("0x6d"), client)
	Require(t, err, "could not deploy ArbAggregator contract")
	tx, err := arbAggregator.SetPreferredAggregator(&auth, aggregator)
	Require(t, err, "could not set L2 gas price")
	_, err = arbnode.EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	// get the network fee account
	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), client)
	Require(t, err, "could not deploy ArbOwner contract")
	networkFeeAccount, err := arbOwner.GetNetworkFeeAccount(callOpts)
	Require(t, err, "could not get the network fee account")

	networkBefore := GetBalance(t, ctx, client, networkFeeAccount)
	colors.PrintMint("network: ", networkFeeAccount, networkBefore)

	auth.GasFeeCap = util.BigMulByUfrac(l2info.GasPrice, 5, 4) // add room for a 20% tip
	auth.GasTipCap = util.BigMulByUfrac(l2info.GasPrice, 1, 4) // add a 20% tip

	_, tx, _, err = mocksgen.DeploySimple(&auth, client)
	Require(t, err, "could not deploy contract")
	_, err = arbnode.EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	networkAfter := GetBalance(t, ctx, client, networkFeeAccount)
	colors.PrintMint("network: ", networkFeeAccount, networkAfter)

	colors.PrintBlue("pricing: ", l2info.GasPrice, auth.GasFeeCap, auth.GasTipCap)
	colors.PrintBlue("payment: ", tx.GasPrice(), tx.GasFeeCap(), tx.GasTipCap())

	if !util.BigEquals(tx.GasPrice(), auth.GasFeeCap) {
		Fail(t, "user did not pay the tip")
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
	_, err = arbnode.EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	// get the network fee account
	_, err = arbAggregator.SetPreferredAggregator(&auth, testhelpers.RandomAddress())
	colors.PrintBlue("expecting error: ", err)
	if err == nil {
		Fail(t, "the sequencer should have rejected this tx")
	}
}
