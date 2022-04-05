//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
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
	auth.GasFeeCap = arbmath.BigMulByUfrac(basefee, 5, 4) // add room for a 20% tip
	auth.GasTipCap = arbmath.BigMulByUfrac(basefee, 1, 4) // add a 20% tip

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

	if !arbmath.BigEquals(tx.GasPrice(), auth.GasFeeCap) {
		Fail(t, "user did not pay the tip")
	}

	tip := arbmath.BigMulByUint(arbmath.BigSub(tx.GasPrice(), basefee), receipt.GasUsed)
	full := arbmath.BigMulByUint(tx.GasPrice(), receipt.GasUsed)
	networkRevenue := arbmath.BigSub(networkAfter, networkBefore)
	colors.PrintMint("tip: ", tip, full, networkRevenue)

	colors.PrintRed("used: ", receipt.GasUsed, basefee)

	if !arbmath.BigEquals(tip, arbmath.BigMulByFrac(networkRevenue, 1, 5)) {
		Fail(t, "1/5th of the network's revenue should be the tip")
	}
	if !arbmath.BigEquals(full, networkRevenue) {
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

func TestSequencerFeePaid(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2info, _, l2client, _, _, _, stack := CreateTestNodeOnL1(t, ctx, true)
	defer stack.Close()

	callOpts := l2info.GetDefaultCallOpts("Owner")

	// get the network fee account
	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(common.HexToAddress("0x6b"), l2client)
	Require(t, err, "could not deploy ArbOwner contract")
	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), l2client)
	Require(t, err, "could not deploy ArbOwner contract")
	networkFeeAccount, err := arbOwnerPublic.GetNetworkFeeAccount(callOpts)
	Require(t, err, "could not get the network fee account")

	l1Estimate, err := arbGasInfo.GetL1GasPriceEstimate(callOpts)
	Require(t, err)
	networkBefore := GetBalance(t, ctx, l2client, networkFeeAccount)
	seqBefore := GetBalance(t, ctx, l2client, l1pricing.SequencerAddress)

	l2info.GasPrice = GetBaseFee(t, l2client, ctx)
	tx, receipt := TransferBalance(t, "Faucet", "Faucet", big.NewInt(0), l2info, l2client, ctx)

	networkAfter := GetBalance(t, ctx, l2client, networkFeeAccount)
	seqAfter := GetBalance(t, ctx, l2client, l1pricing.SequencerAddress)

	networkRevenue := arbmath.BigSub(networkAfter, networkBefore)
	seqRevenue := arbmath.BigSub(seqAfter, seqBefore)

	gasUsedForL2 := receipt.GasUsed - receipt.GasUsedForL1

	if !arbmath.BigEquals(seqRevenue, arbmath.BigMulByUint(tx.GasPrice(), receipt.GasUsedForL1)) {
		Fail(t, "sequencer didn't receive expected payment")
	}
	if !arbmath.BigEquals(networkRevenue, arbmath.BigMulByUint(tx.GasPrice(), gasUsedForL2)) {
		Fail(t, "network didn't receive expected payment")
	}

	paidBytes := arbmath.BigDiv(seqRevenue, l1Estimate).Uint64() / params.TxDataNonZeroGasEIP2028

	txBin, err := tx.MarshalBinary()
	Require(t, err)
	compressed, err := arbcompress.CompressFast(txBin)
	Require(t, err)

	if uint64(len(compressed)) != paidBytes {
		t.Fatal("unexpected number of bytes paid for")
	}

}
