// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestDeploy(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2info, _, client := CreateTestL2(t, ctx)
	auth := l2info.GetDefaultTransactOpts("Owner", ctx)
	auth.GasMargin = 0 // don't adjust, we want to see if the estimate alone is sufficient

	_, tx, simple, err := mocksgen.DeploySimple(&auth, client)
	Require(t, err, "could not deploy contract")
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	tx, err = simple.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	counter, err := simple.Counter(&bind.CallOpts{})
	Require(t, err, "failed to get counter")

	if counter != 1 {
		Fail(t, "Unexpected counter value", counter)
	}
}

func TestEstimate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2info, _, client := CreateTestL2(t, ctx)
	auth := l2info.GetDefaultTransactOpts("Owner", ctx)
	auth.GasMargin = 0 // don't adjust, we want to see if the estimate alone is sufficient

	gasPrice := big.NewInt(params.GWei / 10)

	// set the gas price
	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), client)
	Require(t, err, "could not deploy ArbOwner contract")
	tx, err := arbOwner.SetMinimumL2BaseFee(&auth, gasPrice)
	Require(t, err, "could not set L2 gas price")
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	// connect to arbGasInfo precompile
	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), client)
	Require(t, err, "could not deploy contract")

	// wait for price to come to equilibrium
	equilibrated := false
	numTriesLeft := 20
	for !equilibrated && numTriesLeft > 0 {
		// make an empty block to let the gas price update
		l2info.GasPrice = new(big.Int).Mul(l2info.GasPrice, big.NewInt(2))
		TransferBalance(t, "Owner", "Owner", common.Big0, l2info, client, ctx)

		// check if the price has equilibrated
		_, _, _, _, _, setPrice, err := arbGasInfo.GetPricesInWei(&bind.CallOpts{})
		Require(t, err, "could not get L2 gas price")
		if gasPrice.Cmp(setPrice) == 0 {
			equilibrated = true
		}
		numTriesLeft--
	}
	if !equilibrated {
		Fail(t, "L2 gas price did not converge", gasPrice)
	}

	initialBalance, err := client.BalanceAt(ctx, auth.From, nil)
	Require(t, err, "could not get balance")

	// deploy a test contract
	_, tx, simple, err := mocksgen.DeploySimple(&auth, client)
	Require(t, err, "could not deploy contract")
	receipt, err := EnsureTxSucceeded(ctx, client, tx)
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

	tx, err = simple.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	counter, err := simple.Counter(&bind.CallOpts{})
	Require(t, err, "failed to get counter")

	if counter != 1 {
		Fail(t, "Unexpected counter value", counter)
	}
}

func TestComponentEstimate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2info, node, client := CreateTestL2(t, ctx)
	auth := l2info.GetDefaultTransactOpts("Owner", ctx)

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, client)
	Require(t, err)
	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), client)
	Require(t, err)

	l1BaseFee := big.NewInt(2e8)
	l2BaseFee := GetBaseFee(t, client, ctx)

	colors.PrintGrey("l1 basefee ", l1BaseFee)
	colors.PrintGrey("l2 basefee ", l2BaseFee)

	// set the l1 base fee
	tx, err := arbOwner.SetL1BaseFeeEstimate(&auth, l1BaseFee)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	userBalance := big.NewInt(1e16)
	maxPriorityFeePerGas := big.NewInt(0)
	maxFeePerGas := arbmath.BigMulByUfrac(l2BaseFee, 3, 2)

	l2info.GenerateAccount("User")
	TransferBalance(t, "Owner", "User", userBalance, l2info, client, ctx)

	from := l2info.GetAddress("User")
	to := testhelpers.RandomAddress()
	gas := uint64(1000000)
	data := []byte{0x00, 0x12}
	value := big.NewInt(4096)

	tx = l2info.SignTxAs("User", &types.DynamicFeeTx{
		ChainID:   node.ArbInterface.BlockChain().Config().ChainID,
		Nonce:     0,
		GasTipCap: maxPriorityFeePerGas,
		GasFeeCap: maxFeePerGas,
		Gas:       gas,
		To:        (*common.Address)(&to),
		Value:     value,
		Data:      data,
	})

	estimates, err := nodeInterface.GasEstimateComponents(
		&bind.CallOpts{}, from, to, gas, maxFeePerGas, maxPriorityFeePerGas, value, data,
	)
	Require(t, err)

	l2Estimate := estimates.GasEstimate - estimates.GasEstimateForL1

	colors.PrintBlue("Total ", estimates.GasEstimate)
	colors.PrintBlue("L1    ", estimates.GasEstimateForL1)
	colors.PrintBlue("L2    ", l2Estimate)

	if estimates.L1BaseFeeEstimate != l1BaseFee.Uint64() {
		Fail(t, estimates.L1BaseFeeEstimate, l1BaseFee.Uint64())
	}
	if estimates.BaseFee != l2BaseFee.Uint64() {
		Fail(t, estimates.BaseFee, l2BaseFee.Uint64())
	}

	Require(t, client.SendTransaction(ctx, tx))
	receipt, err := EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	l2Used := receipt.GasUsed - receipt.GasUsedForL1
	colors.PrintBlue("Used  ", l2Used)
}
