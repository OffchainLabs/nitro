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
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
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

	valWalletAddrA, err := validator.CreateValidatorWallet(ctx, valWalletFactory, 0, &l1authA, l1client)
	Require(t, err)
	valWalletAddrCheck, err := validator.CreateValidatorWallet(ctx, valWalletFactory, 0, &l1authA, l1client)
	Require(t, err)
	if valWalletAddrA == valWalletAddrCheck {
		Require(t, err, "didn't cache validator wallet address", valWalletAddrA.String(), "vs", valWalletAddrCheck.String())
	}

	valWalletAddrB, err := validator.CreateValidatorWallet(ctx, valWalletFactory, 0, &l1authB, l1client)
	Require(t, err)

	rollup, err := rollupgen.NewRollupAdminLogic(l2nodeA.DeployInfo.Rollup, l1client)
	Require(t, err)
	tx, err = rollup.SetValidator(&deployAuth, []common.Address{valWalletAddrA, valWalletAddrB}, []bool{true, true})
	Require(t, err)
	_, err = arbutil.EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)

	valConfig := validator.ValidatorConfig{
		UtilsAddress:      valUtils.Hex(),
		TargetNumMachines: 4,
	}

	valWalletA, err := validator.NewValidatorWallet(nil, valWalletFactory, l2nodeA.DeployInfo.Rollup, l1client, &l1authA, 0, func(common.Address) {})
	Require(t, err)
	stakerA, err := validator.NewStaker(
		ctx,
		l1client,
		valWalletA,
		0,
		validator.MakeNodesStrategy,
		bind.CallOpts{},
		&l1authA,
		valConfig,
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
		validator.MakeNodesStrategy,
		bind.CallOpts{},
		&l1authB,
		valConfig,
		l2nodeB.ArbInterface.BlockChain(),
		l2nodeB.InboxReader,
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		l2nodeB.BlockValidator,
	)
	Require(t, err)

	// Continually make L2 transactions in a background thread
	l2info.GenerateAccount("BackgroundUser")
	tx = l2info.PrepareTx("Faucet", "BackgroundUser", l2info.TransferGas, balance, nil)
	err = l2clientA.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = arbutil.EnsureTxSucceeded(ctx, l2clientA, tx)
	Require(t, err)
	err = l2clientB.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = arbutil.EnsureTxSucceeded(ctx, l2clientB, tx)
	Require(t, err)
	go (func() {
		for i := uint64(0); ctx.Err() == nil; i++ {
			l2info.Accounts["BackgroundUser"].Nonce = i
			tx := l2info.PrepareTx("BackgroundUser", "BackgroundUser", l2info.TransferGas, common.Big0, nil)
			err := l2clientA.SendTransaction(ctx, tx)
			Require(t, err)
			_, err = arbutil.EnsureTxSucceeded(ctx, l2clientA, tx)
			Require(t, err)
			if createNodesFlaky || stakeLatestFlaky {
				l2info.Accounts["BackgroundUser"].Nonce = i
				tx = l2info.PrepareTx("BackgroundUser", "BackgroundUser", l2info.TransferGas, common.Big1, nil)
			}
			err = l2clientB.SendTransaction(ctx, tx)
			Require(t, err)
			_, err = arbutil.EnsureTxSucceeded(ctx, l2clientB, tx)
			Require(t, err)
		}
	})()

	for i := 0; i < 100; i++ {
		var stakerName string
		if i%2 == 0 {
			stakerName = "A"
			tx, err = stakerA.Act(ctx)
		} else {
			stakerName = "B"
			tx, err = stakerB.Act(ctx)
		}
		Require(t, err, "Staker", stakerName, "failed to act")
		if tx != nil {
			_, err = arbutil.EnsureTxSucceeded(ctx, l1client, tx)
			Require(t, err, "EnsureTxSucceeded failed for staker", stakerName, "tx")
		}
		for j := 0; j < 20; j++ {
			TransferBalance(t, "Faucet", "Faucet", common.Big0, l1info, l1client, ctx)
		}
	}
}

func TestStakersCooperative(t *testing.T) {
	stakerTestImpl(t, false, false)
}
