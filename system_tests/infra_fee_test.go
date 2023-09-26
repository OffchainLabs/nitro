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

	testNode := NewNodeBuilder(ctx).SetNodeConfig(nodeconfig).CreateTestNodeOnL2Only(t, true)
	defer testNode.L2Node.StopAndWait()

	testNode.L2Info.GenerateAccount("User2")

	ownerTxOpts := testNode.L2Info.GetDefaultTransactOpts("Owner", ctx)
	ownerTxOpts.Context = ctx
	ownerCallOpts := testNode.L2Info.GetDefaultCallOpts("Owner", ctx)

	arbowner, err := precompilesgen.NewArbOwner(common.HexToAddress("70"), testNode.L2Client)
	Require(t, err)
	arbownerPublic, err := precompilesgen.NewArbOwnerPublic(common.HexToAddress("6b"), testNode.L2Client)
	Require(t, err)
	networkFeeAddr, err := arbownerPublic.GetNetworkFeeAccount(ownerCallOpts)
	Require(t, err)
	infraFeeAddr := common.BytesToAddress(crypto.Keccak256([]byte{3, 2, 6}))
	tx, err := arbowner.SetInfraFeeAccount(&ownerTxOpts, infraFeeAddr)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, testNode.L2Client, tx)
	Require(t, err)

	_, simple := testNode.DeploySimple(t, ownerTxOpts)

	netFeeBalanceBefore, err := testNode.L2Client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)
	infraFeeBalanceBefore, err := testNode.L2Client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)

	tx, err = simple.Increment(&ownerTxOpts)
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, testNode.L2Client, tx)
	Require(t, err)
	l2GasUsed := receipt.GasUsed - receipt.GasUsedForL1
	expectedFunds := arbmath.BigMulByUint(arbmath.UintToBig(l2pricing.InitialBaseFeeWei), l2GasUsed)
	expectedBalanceAfter := arbmath.BigAdd(infraFeeBalanceBefore, expectedFunds)

	netFeeBalanceAfter, err := testNode.L2Client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)
	infraFeeBalanceAfter, err := testNode.L2Client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)

	if !arbmath.BigEquals(netFeeBalanceBefore, netFeeBalanceAfter) {
		Fatal(t, netFeeBalanceBefore, netFeeBalanceAfter)
	}
	if !arbmath.BigEquals(infraFeeBalanceAfter, expectedBalanceAfter) {
		Fatal(t, infraFeeBalanceBefore, expectedFunds, infraFeeBalanceAfter, expectedBalanceAfter)
	}
}
