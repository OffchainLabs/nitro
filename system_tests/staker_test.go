// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package arbtest

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func makeBackgroundTxs(ctx context.Context, l2info *BlockchainTestInfo, l2clientA arbutil.L1Interface, l2clientB arbutil.L1Interface, faultyStaker bool) error {
	for i := uint64(0); ctx.Err() == nil; i++ {
		l2info.Accounts["BackgroundUser"].Nonce = i
		tx := l2info.PrepareTx("BackgroundUser", "BackgroundUser", l2info.TransferGas, common.Big0, nil)
		err := l2clientA.SendTransaction(ctx, tx)
		if err != nil {
			return err
		}
		_, err = EnsureTxSucceeded(ctx, l2clientA, tx)
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
			_, err = EnsureTxSucceeded(ctx, l2clientB, tx)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func stakerTestImpl(t *testing.T, faultyStaker bool, honestStakerInactive bool) {
	t.Parallel()
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	l2info, l2nodeA, l2clientA, l1info, _, l1client, l1stack := createTestNodeOnL1(t, ctx, true)
	defer requireClose(t, l1stack)
	defer l2nodeA.StopAndWait()

	if faultyStaker {
		l2info.GenerateGenesisAccount("FaultyAddr", common.Big1)
	}
	l2clientB, l2nodeB := Create2ndNodeWithConfig(t, ctx, l2nodeA, l1stack, l1info, &l2info.ArbInitData, arbnode.ConfigDefaultL1Test())
	defer l2nodeB.StopAndWait()

	nodeAGenesis := l2nodeA.Backend.APIBackend().CurrentHeader().Hash()
	nodeBGenesis := l2nodeB.Backend.APIBackend().CurrentHeader().Hash()
	if faultyStaker {
		if nodeAGenesis == nodeBGenesis {
			Fail(t, "node A L2 genesis hash", nodeAGenesis, "== node B L2 genesis hash", nodeBGenesis)
		}
	} else {
		if nodeAGenesis != nodeBGenesis {
			Fail(t, "node A L2 genesis hash", nodeAGenesis, "!= node B L2 genesis hash", nodeBGenesis)
		}
	}

	deployAuth := l1info.GetDefaultTransactOpts("RollupOwner", ctx)

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	l1info.GenerateAccount("ValidatorA")
	TransferBalance(t, "Faucet", "ValidatorA", balance, l1info, l1client, ctx)
	l1authA := l1info.GetDefaultTransactOpts("ValidatorA", ctx)

	l1info.GenerateAccount("ValidatorB")
	TransferBalance(t, "Faucet", "ValidatorB", balance, l1info, l1client, ctx)
	l1authB := l1info.GetDefaultTransactOpts("ValidatorB", ctx)

	valWalletAddrAPtr, err := staker.GetValidatorWalletContract(ctx, l2nodeA.DeployInfo.ValidatorWalletCreator, 0, &l1authA, l2nodeA.L1Reader, true)
	Require(t, err)
	valWalletAddrA := *valWalletAddrAPtr
	valWalletAddrCheck, err := staker.GetValidatorWalletContract(ctx, l2nodeA.DeployInfo.ValidatorWalletCreator, 0, &l1authA, l2nodeA.L1Reader, true)
	Require(t, err)
	if valWalletAddrA == *valWalletAddrCheck {
		Require(t, err, "didn't cache validator wallet address", valWalletAddrA.String(), "vs", valWalletAddrCheck.String())
	}

	rollup, err := rollupgen.NewRollupAdminLogic(l2nodeA.DeployInfo.Rollup, l1client)
	Require(t, err)
	tx, err := rollup.SetValidator(&deployAuth, []common.Address{valWalletAddrA, l1authB.From}, []bool{true, true})
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)

	tx, err = rollup.SetMinimumAssertionPeriod(&deployAuth, big.NewInt(1))
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)

	valConfig := staker.L1ValidatorConfig{}

	valWalletA, err := staker.NewContractValidatorWallet(nil, l2nodeA.DeployInfo.ValidatorWalletCreator, l2nodeA.DeployInfo.Rollup, l2nodeA.L1Reader, &l1authA, 0, func(common.Address) {})
	Require(t, err)
	if honestStakerInactive {
		valConfig.Strategy = "Defensive"
	} else {
		valConfig.Strategy = "MakeNodes"
	}

	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	blockValidatorConfig := staker.TestBlockValidatorConfig
	blockValidatorConfig.URL = valStack.WSEndpoint()

	statelessA, err := staker.NewStatelessBlockValidator(
		l2nodeA.InboxReader,
		l2nodeA.InboxTracker,
		l2nodeA.TxStreamer,
		l2nodeA.ArbInterface.BlockChain(),
		l2nodeA.ChainDB,
		l2nodeA.ArbDB,
		nil,
		&blockValidatorConfig,
	)
	Require(t, err)
	stakerA, err := staker.NewStaker(
		l2nodeA.L1Reader,
		valWalletA,
		bind.CallOpts{},
		valConfig,
		nil,
		statelessA,
		l2nodeA.DeployInfo.ValidatorUtils,
	)
	Require(t, err)
	err = stakerA.Initialize(ctx)
	Require(t, err)

	valWalletB, err := staker.NewEoaValidatorWallet(l2nodeB.DeployInfo.Rollup, l2nodeB.L1Reader.Client(), &l1authB)
	Require(t, err)
	valConfig.Strategy = "MakeNodes"
	statelessB, err := staker.NewStatelessBlockValidator(
		l2nodeB.InboxReader,
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		l2nodeB.ArbInterface.BlockChain(),
		l2nodeB.ChainDB,
		l2nodeB.ArbDB,
		nil,
		&blockValidatorConfig,
	)
	Require(t, err)
	stakerB, err := staker.NewStaker(
		l2nodeB.L1Reader,
		valWalletB,
		bind.CallOpts{},
		valConfig,
		nil,
		statelessB,
		l2nodeB.DeployInfo.ValidatorUtils,
	)
	Require(t, err)
	err = stakerB.Initialize(ctx)
	Require(t, err)

	valWalletC, err := staker.NewContractValidatorWallet(nil, l2nodeA.DeployInfo.ValidatorWalletCreator, l2nodeA.DeployInfo.Rollup, l2nodeA.L1Reader, nil, 0, func(common.Address) {})
	Require(t, err)
	valConfig.Strategy = "Watchtower"
	stakerC, err := staker.NewStaker(
		l2nodeA.L1Reader,
		valWalletC,
		bind.CallOpts{},
		valConfig,
		nil,
		statelessA,
		l2nodeA.DeployInfo.ValidatorUtils,
	)
	Require(t, err)
	err = stakerC.Initialize(ctx)
	Require(t, err)

	l2info.GenerateAccount("BackgroundUser")
	tx = l2info.PrepareTx("Faucet", "BackgroundUser", l2info.TransferGas, balance, nil)
	err = l2clientA.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2clientA, tx)
	Require(t, err)
	if faultyStaker {
		err = l2clientB.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, l2clientB, tx)
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
			log.Warn("error making background txs", "err", err)
		}
	})()

	stakerATxs := 0
	stakerAWasStaked := false
	stakerBTxs := 0
	stakerBWasStaked := false
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

		if err != nil && strings.Contains(err.Error(), "waiting") {
			colors.PrintRed("retrying ", err.Error(), i)
			time.Sleep(20 * time.Millisecond)
			i--
			continue
		}

		if err != nil && faultyStaker && i%2 == 1 {
			// Check if this is an expected error from the faulty staker.
			if strings.Contains(err.Error(), "agreed with entire challenge") {
				// Expected error upon realizing you're losing the challenge. Get ready for a timeout.
				for j := 0; j < 200; j++ {
					TransferBalance(t, "Faucet", "Faucet", common.Big0, l1info, l1client, ctx)
				}
			} else if strings.Contains(err.Error(), "insufficient funds") && sawStakerZombie {
				// Expected error when trying to re-stake after losing initial stake.
			} else if strings.Contains(err.Error(), "unknown start block hash") && sawStakerZombie {
				// Expected error when trying to re-stake after the challenger's nodes getting confirmed.
			} else {
				Require(t, err, "Faulty staker failed to act")
			}
			t.Log("got expected faulty staker error", err)
			err = nil
			tx = nil
		}
		Require(t, err, "Staker", stakerName, "failed to act")
		if tx != nil {
			_, err = EnsureTxSucceeded(ctx, l1client, tx)
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
			sawStakerZombie, err = rollup.IsZombie(&bind.CallOpts{}, l1authB.From)
			Require(t, err)
		}
		isHonestZombie, err := rollup.IsZombie(&bind.CallOpts{}, valWalletAddrA)
		Require(t, err)
		if isHonestZombie {
			Fail(t, "staker A became a zombie")
		}
		watchTx, err := stakerC.Act(ctx)
		if err != nil && !strings.Contains(err.Error(), "catch up") {
			Require(t, err, "watchtower staker failed to act")
		}
		if watchTx != nil {
			Fail(t, "watchtower staker made a transaction")
		}
		if !stakerAWasStaked {
			stakerAWasStaked, err = rollup.IsStaked(&bind.CallOpts{}, valWalletAddrA)
			Require(t, err)
		}
		if !stakerBWasStaked {
			stakerBWasStaked, err = rollup.IsStaked(&bind.CallOpts{}, l1authB.From)
			Require(t, err)
		}
		for j := 0; j < 5; j++ {
			TransferBalance(t, "Faucet", "Faucet", common.Big0, l1info, l1client, ctx)
		}
	}

	if stakerATxs == 0 || stakerBTxs == 0 {
		Fail(t, "staker didn't make txs: staker A made", stakerATxs, "staker B made", stakerBTxs)
	}

	latestConfirmedNode, err := rollup.LatestConfirmed(&bind.CallOpts{})
	Require(t, err)

	if latestConfirmedNode <= 1 && !honestStakerInactive {
		latestCreatedNode, err := rollup.LatestNodeCreated(&bind.CallOpts{})
		Require(t, err)
		Fail(t, "latest confirmed node didn't advance:", latestConfirmedNode, latestCreatedNode)
	}

	if faultyStaker && !sawStakerZombie {
		Fail(t, "staker B didn't become a zombie despite being faulty")
	}

	if !stakerAWasStaked {
		Fail(t, "staker A was never staked")
	}
	if !stakerBWasStaked {
		Fail(t, "staker B was never staked")
	}
}

func TestStakersCooperative(t *testing.T) {
	stakerTestImpl(t, false, false)
}
