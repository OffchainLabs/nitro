// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package arbtest

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestInfraFee(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nodeconfig := arbnode.ConfigDefaultL2Test()

	l2info, _, client, stack := CreateTestL2WithConfig(t, ctx, nil, nodeconfig, true)
	defer requireClose(t, stack)

	l2info.GenerateAccount("User2")

	ownerTxOpts := l2info.GetDefaultTransactOpts("Owner", ctx)
	ownerTxOpts.Context = ctx
	ownerCallOpts := l2info.GetDefaultCallOpts("Owner", ctx)

	arbowner, err := precompilesgen.NewArbOwner(common.HexToAddress("70"), client)
	Require(t, err)
	arbownerPublic, err := precompilesgen.NewArbOwnerPublic(common.HexToAddress("6b"), client)
	Require(t, err)
	networkFeeAddr, err := arbownerPublic.GetNetworkFeeAccount(ownerCallOpts)
	Require(t, err)
	infraFeeAddr := common.BytesToAddress(crypto.Keccak256([]byte{3, 2, 6}))
	tx, err := arbowner.SetInfraFeeAccount(&ownerTxOpts, infraFeeAddr)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	_, simple := deploySimple(t, ctx, ownerTxOpts, client)

	netFeeBalanceBefore, err := client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)
	infraFeeBalanceBefore, err := client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)

	tx, err = simple.Increment(&ownerTxOpts)
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
	l2GasUsed := receipt.GasUsed - receipt.GasUsedForL1
	expectedFunds := arbmath.BigMulByUint(arbmath.UintToBig(l2pricing.InitialBaseFeeWei), l2GasUsed)
	expectedBalanceAfter := arbmath.BigAdd(infraFeeBalanceBefore, expectedFunds)

	netFeeBalanceAfter, err := client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)
	infraFeeBalanceAfter, err := client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)

	if !arbmath.BigEquals(netFeeBalanceBefore, netFeeBalanceAfter) {
		Fail(t, netFeeBalanceBefore, netFeeBalanceAfter)
	}
	if !arbmath.BigEquals(infraFeeBalanceAfter, expectedBalanceAfter) {
		Fail(t, infraFeeBalanceBefore, expectedFunds, infraFeeBalanceAfter, expectedBalanceAfter)
	}
}
