//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbutil"
	"github.com/offchainlabs/arbstate/solgen/go/rollupgen"
	"github.com/offchainlabs/arbstate/validator"
)

func stakerTestImpl(t *testing.T, createNodesFlaky bool, stakeLatestFlaky bool) {
	ctx := context.Background()
	l2info, l2nodeA, l2clientA, l1info, _, l1client, l1stack := CreateTestNodeOnL1(t, ctx, true)
	defer l1stack.Close()

	l2clientB, l2nodeB := Create2ndNode(t, ctx, l2nodeA, l1stack, &l2info.ArbInitData, false)

	deployAuth := l1info.GetDefaultTransactOpts("RollupOwner")

	valWalletFactory, tx, _, err := rollupgen.DeployValidatorWalletCreator(&deployAuth, l1client)
	Require(t, err)
	_, err = arbutil.EnsureTxSucceededWithTimeout(ctx, l1client, tx, time.Second*5)
	Require(t, err)

	valUtils, tx, _, err := rollupgen.DeployValidatorUtils(&deployAuth, l1client)
	Require(t, err)
	_, err = arbutil.EnsureTxSucceededWithTimeout(ctx, l1client, tx, time.Second*5)
	Require(t, err)

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	l1info.GenerateAccount("ValidatorA")
	TransferBalance(t, "Faucet", "ValidatorA", balance, l1info, l1client, ctx)
	l1authA := l1info.GetDefaultTransactOpts("ValidatorA")

	l1info.GenerateAccount("ValidatorB")
	TransferBalance(t, "Faucet", "ValidatorB", balance, l1info, l1client, ctx)
	l1authB := l1info.GetDefaultTransactOpts("ValidatorB")

	valWalletA, err := validator.NewValidatorWallet(nil, valWalletFactory, l2nodeA.DeployInfo.Rollup, l1client, &l1authA, 0, func(common.Address) {})
	Require(t, err)
	stakerA, err := validator.NewStaker(
		ctx,
		l1client,
		valWalletA,
		0,
		valUtils,
		validator.MakeNodesStrategy,
		bind.CallOpts{},
		&l1authA,
		validator.ValidatorConfig{},
		l2nodeA.ArbInterface.BlockChain(),
		l2nodeA.InboxReader,
		l2nodeA.InboxTracker,
		l2nodeA.TxStreamer,
		l2nodeA.BlockValidator,
	)
	Require(t, err)

	valWalletB, err := validator.NewValidatorWallet(nil, valWalletFactory, l2nodeB.DeployInfo.Rollup, l1client, &l1authB, 0, func(common.Address) {})
	Require(t, err)
	stakerB, err := validator.NewStaker(
		ctx,
		l1client,
		valWalletB,
		0,
		valUtils,
		validator.MakeNodesStrategy,
		bind.CallOpts{},
		&l1authB,
		validator.ValidatorConfig{},
		l2nodeB.ArbInterface.BlockChain(),
		l2nodeB.InboxReader,
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		l2nodeB.BlockValidator,
	)
	Require(t, err)

	_, _, _, _ = l2clientA, l2clientB, stakerA, stakerB
}

func TestStakersCooperative(t *testing.T) {
	stakerTestImpl(t, false, false)
}
