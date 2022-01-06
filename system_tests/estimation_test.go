//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/precompiles"
	"github.com/offchainlabs/arbstate/solgen/go/mocksgen"
	"github.com/offchainlabs/arbstate/solgen/go/precompilesgen"
)

func TestDeploy(t *testing.T) {
	precompiles.AllowDebugPrecompiles = true
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, client, auth := CreateTestL2(t, ctx)

	_, tx, simple, err := mocksgen.DeploySimple(auth, client)
	Require(t, err, "could not deploy contract")
	_, err = arbnode.EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	tx, err = simple.Increment(auth)
	Require(t, err, "failed to call Increment()")
	_, err = arbnode.EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	counter, err := simple.Counter(&bind.CallOpts{})
	Require(t, err, "failed to get counter")

	if counter != 1 {
		Fail(t, "Unexpected counter value", counter)
	}
}

func TestEstimate(t *testing.T) {
	precompiles.AllowDebugPrecompiles = true
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, client, auth := CreateTestL2(t, ctx)

	gasPrice := big.NewInt(2 * params.GWei)

	// set the gas price
	arbowner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), client)
	Require(t, err, "could not deploy ArbOwner contract")
	tx, err := arbowner.SetL2GasPrice(auth, gasPrice)
	Require(t, err, "could not set L2 gas price")
	_, err = arbnode.EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	// get the gas price
	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), client)
	Require(t, err, "could not deploy contract")
	_, _, _, _, _, setPrice, err := arbGasInfo.GetPricesInWei(&bind.CallOpts{})
	Require(t, err, "could not get L2 gas price")
	if gasPrice.Cmp(setPrice) != 0 {
		Fail(t, "L2 gas price was not set correctly", gasPrice, setPrice)
	}

	initialBalance, err := client.BalanceAt(ctx, auth.From, nil)
	Require(t, err, "could not get balance")

	// deploy a test contract
	_, tx, simple, err := mocksgen.DeploySimple(auth, client)
	Require(t, err, "could not deploy contract")
	receipt, err := arbnode.EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	header, err := client.HeaderByNumber(ctx, receipt.BlockNumber)
	Require(t, err, "could not get header")
	if header.BaseFee.Cmp(gasPrice) != 0 {
		Fail(t, "Header has wrong basefee", header.BaseFee, gasPrice)
	}

	balance, err := client.BalanceAt(ctx, auth.From, nil)
	Require(t, err, "could not get balance")
	expectedCost := receipt.GasUsed * gasPrice.Uint64()
	observedCost := initialBalance.Uint64() - balance.Uint64()
	if expectedCost != observedCost {
		Fail(t, "Expected deployment to cost", expectedCost, "instead of", observedCost)
	}

	tx, err = simple.Increment(auth)
	Require(t, err, "failed to call Increment()")
	_, err = arbnode.EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	counter, err := simple.Counter(&bind.CallOpts{})
	Require(t, err, "failed to get counter")

	if counter != 1 {
		Fail(t, "Unexpected counter value", counter)
	}
}
