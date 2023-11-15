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
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestInfraFee(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")

	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	ownerTxOpts.Context = ctx
	ownerCallOpts := builder.L2Info.GetDefaultCallOpts("Owner", ctx)

	arbowner, err := precompilesgen.NewArbOwner(common.HexToAddress("70"), builder.L2.Client)
	Require(t, err)
	arbownerPublic, err := precompilesgen.NewArbOwnerPublic(common.HexToAddress("6b"), builder.L2.Client)
	Require(t, err)
	networkFeeAddr, err := arbownerPublic.GetNetworkFeeAccount(ownerCallOpts)
	Require(t, err)
	infraFeeAddr := common.BytesToAddress(crypto.Keccak256([]byte{3, 2, 6}))
	tx, err := arbowner.SetInfraFeeAccount(&ownerTxOpts, infraFeeAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	_, simple := builder.L2.DeploySimple(t, ownerTxOpts)

	netFeeBalanceBefore, err := builder.L2.Client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)
	infraFeeBalanceBefore, err := builder.L2.Client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)

	tx, err = simple.Increment(&ownerTxOpts)
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	l2GasUsed := receipt.GasUsed - receipt.GasUsedForL1
	expectedFunds := arbmath.BigMulByUint(arbmath.UintToBig(l2pricing.InitialBaseFeeWei), l2GasUsed)
	expectedBalanceAfter := arbmath.BigAdd(infraFeeBalanceBefore, expectedFunds)

	netFeeBalanceAfter, err := builder.L2.Client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)
	infraFeeBalanceAfter, err := builder.L2.Client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)

	if !arbmath.BigEquals(netFeeBalanceBefore, netFeeBalanceAfter) {
		Fatal(t, netFeeBalanceBefore, netFeeBalanceAfter)
	}
	if !arbmath.BigEquals(infraFeeBalanceAfter, expectedBalanceAfter) {
		Fatal(t, infraFeeBalanceBefore, expectedFunds, infraFeeBalanceAfter, expectedBalanceAfter)
	}
}
