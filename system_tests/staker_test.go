//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package arbtest

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbutil"
	"github.com/offchainlabs/arbstate/solgen/go/rollupgen"
	"github.com/offchainlabs/arbstate/validator"
)

func makeBackgroundTxs(ctx context.Context, l2info *BlockchainTestInfo, l2clientA arbutil.L1Interface, l2clientB arbutil.L1Interface, faultyStaker bool) error {
	for i := uint64(0); ctx.Err() == nil; i++ {
		l2info.Accounts["BackgroundUser"].Nonce = i
		tx := l2info.PrepareTx("BackgroundUser", "BackgroundUser", l2info.TransferGas, common.Big0, nil)
		err := l2clientA.SendTransaction(ctx, tx)
		if err != nil {
			return err
		}
		_, err = arbutil.EnsureTxSucceeded(ctx, l2clientA, tx)
		if err != nil {
			return err
		}
		if faultyStaker {
			// Create a different transaction for the second node
			l2info.Accounts["BackgroundUser"].Nonce = i
			tx = l2info.PrepareTx("BackgroundUser", "BackgroundUser", l2info.TransferGas, common.Big1, nil)
			err = l2clientB.SendTransaction(ctx, tx)
			if err != nil {
				return err
			}
			_, err = arbutil.EnsureTxSucceeded(ctx, l2clientB, tx)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func stakerTestImpl(t *testing.T, faultyStaker bool, honestStakerInactive bool) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	l2info, l2nodeA, l2clientA, l1info, _, l1client, l1stack := CreateTestNodeOnL1(t, ctx, true)
	defer l1stack.Close()

	if faultyStaker {
		l2info.GenerateGenesysAccount("FaultyAddr", common.Big1)
	}
	l2clientB, l2nodeB := Create2ndNode(t, ctx, l2nodeA, l1stack, &l2info.ArbInitData, false)

	nodeAGenesis := l2nodeA.Backend.APIBackend().CurrentHeader().Hash()
	nodeBGenesis := l2nodeB.Backend.APIBackend().CurrentHeader().Hash()
	if faultyStaker {
		if nodeAGenesis == nodeBGenesis {
			t.Fatal("node A L2 genesis hash", nodeAGenesis, "== node B L2 genesis hash", nodeBGenesis)
		}
	} else {
		if nodeAGenesis != nodeBGenesis {
			t.Fatal("node A L2 genesis hash", nodeAGenesis, "!= node B L2 genesis hash", nodeBGenesis)
		}
	}

	deployAuth := l1info.GetDefaultTransactOpts("RollupOwner")

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	l1info.GenerateAccount("ValidatorA")
	TransferBalance(t, "Faucet", "ValidatorA", balance, l1info, l1client, ctx)
	l1authA := l1info.GetDefaultTransactOpts("ValidatorA")

	l1info.GenerateAccount("ValidatorB")
	TransferBalance(t, "Faucet", "ValidatorB", balance, l1info, l1client, ctx)
	l1authB := l1info.GetDefaultTransactOpts("ValidatorB")

	valWalletAddrA, err := validator.CreateValidatorWallet(ctx, l2nodeA.DeployInfo.ValidatorWalletCreator, 0, &l1authA, l1client)
	Require(t, err)
	valWalletAddrCheck, err := validator.CreateValidatorWallet(ctx, l2nodeA.DeployInfo.ValidatorWalletCreator, 0, &l1authA, l1client)
	Require(t, err)
	if valWalletAddrA == valWalletAddrCheck {
		Require(t, err, "didn't cache validator wallet address", valWalletAddrA.String(), "vs", valWalletAddrCheck.String())
	}

	valWalletAddrB, err := validator.CreateValidatorWallet(ctx, l2nodeA.DeployInfo.ValidatorWalletCreator, 0, &l1authB, l1client)
	Require(t, err)

	rollup, err := rollupgen.NewRollupAdminLogic(l2nodeA.DeployInfo.Rollup, l1client)
	Require(t, err)
	tx, err := rollup.SetValidator(&deployAuth, []common.Address{valWalletAddrA, valWalletAddrB}, []bool{true, true})
	Require(t, err)
	_, err = arbutil.EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)

	tx, err = rollup.SetMinimumAssertionPeriod(&deployAuth, big.NewInt(1))
	Require(t, err)
	_, err = arbutil.EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)

	valConfig := validator.L1ValidatorConfig{
		TargetNumMachines: 4,
	}

	valWalletA, err := validator.NewValidatorWallet(nil, l2nodeA.DeployInfo.ValidatorWalletCreator, l2nodeA.DeployInfo.Rollup, l1client, &l1authA, 0, func(common.Address) {})
	Require(t, err)
	if honestStakerInactive {
		valConfig.Strategy = "Defensive"
	} else {
		valConfig.Strategy = "MakeNodes"
	}
	stakerA, err := validator.NewStaker(
		l1client,
		valWalletA,
		bind.CallOpts{},
		valConfig,
		l2nodeA.ArbInterface.BlockChain(),
		l2nodeA.InboxReader,
		l2nodeA.InboxTracker,
		l2nodeA.TxStreamer,
		l2nodeA.BlockValidator,
		l2nodeA.DeployInfo.ValidatorUtils,
	)
	Require(t, err)
	err = stakerA.Initialize(ctx)
	Require(t, err)

	valWalletB, err := validator.NewValidatorWallet(nil, l2nodeA.DeployInfo.ValidatorWalletCreator, l2nodeB.DeployInfo.Rollup, l1client, &l1authB, 0, func(common.Address) {})
	Require(t, err)
	valConfig.Strategy = "MakeNodes"
	stakerB, err := validator.NewStaker(
		l1client,
		valWalletB,
		bind.CallOpts{},
		valConfig,
		l2nodeB.ArbInterface.BlockChain(),
		l2nodeB.InboxReader,
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		l2nodeB.BlockValidator,
		l2nodeA.DeployInfo.ValidatorUtils,
	)
	Require(t, err)
	err = stakerB.Initialize(ctx)
	Require(t, err)

	l2info.GenerateAccount("BackgroundUser")
	tx = l2info.PrepareTx("Faucet", "BackgroundUser", l2info.TransferGas, balance, nil)
	err = l2clientA.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = arbutil.EnsureTxSucceeded(ctx, l2clientA, tx)
	Require(t, err)
	if faultyStaker {
		err = l2clientB.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = arbutil.EnsureTxSucceeded(ctx, l2clientB, tx)
		Require(t, err)
	}

	// Continually make L2 transactions in a background thread
	backgroundTxsCtx, cancelBackgroundTxs := context.WithCancel(ctx)
	backgroundTxsShutdownChan := make(chan struct{})
	defer (func() {
		cancelBackgroundTxs()
		<-backgroundTxsShutdownChan
	})()
	go (func() {
		defer close(backgroundTxsShutdownChan)
		err := makeBackgroundTxs(backgroundTxsCtx, l2info, l2clientA, l2clientB, faultyStaker)
		if !errors.Is(err, context.Canceled) {
			t.Error("error making background txs", err)
		}
	})()

	stakerATxs := 0
	stakerBTxs := 0
	sawStakerZombie := false
	for i := 0; i < 100; i++ {
		var stakerName string
		if i%2 == 0 {
			stakerName = "A"
			fmt.Printf("staker A acting:\n")
			tx, err = stakerA.Act(ctx)
			if tx != nil {
				stakerATxs++
			}
		} else {
			stakerName = "B"
			fmt.Printf("staker B acting:\n")
			tx, err = stakerB.Act(ctx)
			if tx != nil {
				stakerBTxs++
			}
		}
		Require(t, err, "Staker", stakerName, "failed to act")
		if tx != nil {
			_, err = arbutil.EnsureTxSucceeded(ctx, l1client, tx)
			Require(t, err, "EnsureTxSucceeded failed for staker", stakerName, "tx")
		}
		if faultyStaker {
			challengeAddr, err := rollup.CurrentChallenge(&bind.CallOpts{}, valWalletAddrA)
			Require(t, err)
			if challengeAddr != 0 {
				cancelBackgroundTxs()
			}
		}
		if faultyStaker && !sawStakerZombie {
			sawStakerZombie, err = rollup.IsZombie(&bind.CallOpts{}, valWalletAddrB)
			Require(t, err)
		}
		isHonestZombie, err := rollup.IsZombie(&bind.CallOpts{}, valWalletAddrA)
		Require(t, err)
		if isHonestZombie {
			t.Fatal("staker A became a zombie")
		}
		for j := 0; j < 5; j++ {
			TransferBalance(t, "Faucet", "Faucet", common.Big0, l1info, l1client, ctx)
		}
	}

	if stakerATxs == 0 || stakerBTxs == 0 {
		t.Fatal("staker didn't make txs: staker A made", stakerATxs, "staker B made", stakerBTxs)
	}

	latestConfirmedNode, err := rollup.LatestConfirmed(&bind.CallOpts{})
	Require(t, err)

	if latestConfirmedNode <= 1 {
		latestCreatedNode, err := rollup.LatestNodeCreated(&bind.CallOpts{})
		Require(t, err)
		t.Fatal("latest confirmed node didn't advance:", latestConfirmedNode, latestCreatedNode)
	}

	if faultyStaker && !sawStakerZombie {
		t.Fatal("staker B didn't become a zombie despite being faulty")
	}

	isStaked, err := rollup.IsStaked(&bind.CallOpts{}, valWalletAddrA)
	Require(t, err)
	if !isStaked {
		t.Fatal("staker A isn't staked")
	}

	if !faultyStaker {
		isStaked, err := rollup.IsStaked(&bind.CallOpts{}, valWalletAddrB)
		Require(t, err)
		if !isStaked {
			t.Fatal("staker B isn't staked")
		}
	}
}

func TestStakersCooperative(t *testing.T) {
	stakerTestImpl(t, false, false)
}
