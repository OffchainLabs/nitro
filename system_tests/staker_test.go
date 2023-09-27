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
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/validatorwallet"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func makeBackgroundTxs(ctx context.Context, l2info *BlockchainTestInfo, l2clientA arbutil.L1Interface) error {
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
	}
	return nil
}

func stakerTestImpl(t *testing.T, faultyStaker bool, honestStakerInactive bool) {
	t.Parallel()
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs
	l2chainConfig := params.ArbitrumDevTestChainConfig()
	l2info := NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(l2chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	testNodeA := NewNodeBuilder(ctx).SetIsSequencer(true).SetChainConfig(l2chainConfig).SetL2Info(l2info).CreateTestNodeOnL1AndL2(t)
	defer requireClose(t, testNodeA.L1Stack)
	defer testNodeA.L2Node.StopAndWait()

	if faultyStaker {
		l2info.GenerateGenesisAccount("FaultyAddr", common.Big1)
	}
	config := arbnode.ConfigDefaultL1Test()
	config.Sequencer.Enable = false
	config.DelayedSequencer.Enable = false
	config.BatchPoster.Enable = false
	_, l2nodeB := Create2ndNodeWithConfig(t, ctx, testNodeA.L2Node, testNodeA.L1Stack, testNodeA.L1Info, &l2info.ArbInitData, config, nil)
	defer l2nodeB.StopAndWait()

	nodeAGenesis := testNodeA.L2Node.Execution.Backend.APIBackend().CurrentHeader().Hash()
	nodeBGenesis := l2nodeB.Execution.Backend.APIBackend().CurrentHeader().Hash()
	if faultyStaker {
		if nodeAGenesis == nodeBGenesis {
			Fatal(t, "node A L2 genesis hash", nodeAGenesis, "== node B L2 genesis hash", nodeBGenesis)
		}
	} else {
		if nodeAGenesis != nodeBGenesis {
			Fatal(t, "node A L2 genesis hash", nodeAGenesis, "!= node B L2 genesis hash", nodeBGenesis)
		}
	}

	testNodeA.BridgeBalance(t, "Faucet", big.NewInt(1).Mul(big.NewInt(params.Ether), big.NewInt(10000)))

	deployAuth := testNodeA.L1Info.GetDefaultTransactOpts("RollupOwner", ctx)

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	testNodeA.L1Info.GenerateAccount("ValidatorA")
	testNodeA.TransferBalanceViaL1(t, "Faucet", "ValidatorA", balance)
	l1authA := testNodeA.L1Info.GetDefaultTransactOpts("ValidatorA", ctx)

	testNodeA.L1Info.GenerateAccount("ValidatorB")
	testNodeA.TransferBalanceViaL1(t, "Faucet", "ValidatorB", balance)
	l1authB := testNodeA.L1Info.GetDefaultTransactOpts("ValidatorB", ctx)

	valWalletAddrAPtr, err := validatorwallet.GetValidatorWalletContract(ctx, testNodeA.L2Node.DeployInfo.ValidatorWalletCreator, 0, &l1authA, testNodeA.L2Node.L1Reader, true)
	Require(t, err)
	valWalletAddrA := *valWalletAddrAPtr
	valWalletAddrCheck, err := validatorwallet.GetValidatorWalletContract(ctx, testNodeA.L2Node.DeployInfo.ValidatorWalletCreator, 0, &l1authA, testNodeA.L2Node.L1Reader, true)
	Require(t, err)
	if valWalletAddrA == *valWalletAddrCheck {
		Require(t, err, "didn't cache validator wallet address", valWalletAddrA.String(), "vs", valWalletAddrCheck.String())
	}

	rollup, err := rollupgen.NewRollupAdminLogic(testNodeA.L2Node.DeployInfo.Rollup, testNodeA.L1Client)
	Require(t, err)
	tx, err := rollup.SetValidator(&deployAuth, []common.Address{valWalletAddrA, l1authB.From}, []bool{true, true})
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, testNodeA.L1Client, tx)
	Require(t, err)

	tx, err = rollup.SetMinimumAssertionPeriod(&deployAuth, big.NewInt(1))
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, testNodeA.L1Client, tx)
	Require(t, err)

	validatorUtils, err := rollupgen.NewValidatorUtils(testNodeA.L2Node.DeployInfo.ValidatorUtils, testNodeA.L1Client)
	Require(t, err)

	valConfig := staker.TestL1ValidatorConfig

	dpA, err := arbnode.StakerDataposter(ctx, rawdb.NewTable(l2nodeB.ArbDB, storage.StakerPrefix), testNodeA.L2Node.L1Reader, &l1authA, NewFetcherFromConfig(arbnode.ConfigDefaultL1NonSequencerTest()), nil)
	if err != nil {
		t.Fatalf("Error creating validator dataposter: %v", err)
	}
	valWalletA, err := validatorwallet.NewContract(dpA, nil, testNodeA.L2Node.DeployInfo.ValidatorWalletCreator, testNodeA.L2Node.DeployInfo.Rollup, testNodeA.L2Node.L1Reader, &l1authA, 0, func(common.Address) {}, func() uint64 { return valConfig.ExtraGas })
	Require(t, err)
	if honestStakerInactive {
		valConfig.Strategy = "Defensive"
	} else {
		valConfig.Strategy = "MakeNodes"
	}

	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	blockValidatorConfig := staker.TestBlockValidatorConfig

	statelessA, err := staker.NewStatelessBlockValidator(
		testNodeA.L2Node.InboxReader,
		testNodeA.L2Node.InboxTracker,
		testNodeA.L2Node.TxStreamer,
		testNodeA.L2Node.Execution.Recorder,
		testNodeA.L2Node.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	err = statelessA.Start(ctx)
	Require(t, err)
	stakerA, err := staker.NewStaker(
		testNodeA.L2Node.L1Reader,
		valWalletA,
		bind.CallOpts{},
		valConfig,
		nil,
		statelessA,
		nil,
		nil,
		testNodeA.L2Node.DeployInfo.ValidatorUtils,
		nil,
	)
	Require(t, err)
	err = stakerA.Initialize(ctx)
	if stakerA.Strategy() != staker.WatchtowerStrategy {
		err = valWalletA.Initialize(ctx)
		Require(t, err)
	}
	Require(t, err)

	dpB, err := arbnode.StakerDataposter(ctx, rawdb.NewTable(l2nodeB.ArbDB, storage.StakerPrefix), l2nodeB.L1Reader, &l1authB, NewFetcherFromConfig(arbnode.ConfigDefaultL1NonSequencerTest()), nil)
	if err != nil {
		t.Fatalf("Error creating validator dataposter: %v", err)
	}
	valWalletB, err := validatorwallet.NewEOA(dpB, l2nodeB.DeployInfo.Rollup, l2nodeB.L1Reader.Client(), &l1authB, func() uint64 { return 0 })
	Require(t, err)
	valConfig.Strategy = "MakeNodes"
	statelessB, err := staker.NewStatelessBlockValidator(
		l2nodeB.InboxReader,
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		l2nodeB.Execution.Recorder,
		l2nodeB.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	err = statelessB.Start(ctx)
	Require(t, err)
	stakerB, err := staker.NewStaker(
		l2nodeB.L1Reader,
		valWalletB,
		bind.CallOpts{},
		valConfig,
		nil,
		statelessB,
		nil,
		nil,
		l2nodeB.DeployInfo.ValidatorUtils,
		nil,
	)
	Require(t, err)
	err = stakerB.Initialize(ctx)
	Require(t, err)
	if stakerB.Strategy() != staker.WatchtowerStrategy {
		err = valWalletB.Initialize(ctx)
		Require(t, err)
	}
	dpC, err := arbnode.StakerDataposter(ctx, rawdb.NewTable(l2nodeB.ArbDB, storage.StakerPrefix), testNodeA.L2Node.L1Reader, &l1authA, NewFetcherFromConfig(arbnode.ConfigDefaultL1NonSequencerTest()), nil)
	if err != nil {
		t.Fatalf("Error creating validator dataposter: %v", err)
	}
	valWalletC, err := validatorwallet.NewContract(dpC, nil, testNodeA.L2Node.DeployInfo.ValidatorWalletCreator, testNodeA.L2Node.DeployInfo.Rollup, testNodeA.L2Node.L1Reader, nil, 0, func(common.Address) {}, func() uint64 { return 10000 })
	Require(t, err)
	valConfig.Strategy = "Watchtower"
	stakerC, err := staker.NewStaker(
		testNodeA.L2Node.L1Reader,
		valWalletC,
		bind.CallOpts{},
		valConfig,
		nil,
		statelessA,
		nil,
		nil,
		testNodeA.L2Node.DeployInfo.ValidatorUtils,
		nil,
	)
	Require(t, err)
	if stakerC.Strategy() != staker.WatchtowerStrategy {
		err = valWalletC.Initialize(ctx)
		Require(t, err)
	}
	err = stakerC.Initialize(ctx)
	Require(t, err)

	l2info.GenerateAccount("BackgroundUser")
	tx = l2info.PrepareTx("Faucet", "BackgroundUser", l2info.TransferGas, balance, nil)
	err = testNodeA.L2Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, testNodeA.L2Client, tx)
	Require(t, err)

	// Continually make L2 transactions in a background thread
	backgroundTxsCtx, cancelBackgroundTxs := context.WithCancel(ctx)
	backgroundTxsShutdownChan := make(chan struct{})
	defer (func() {
		cancelBackgroundTxs()
		<-backgroundTxsShutdownChan
	})()
	go (func() {
		defer close(backgroundTxsShutdownChan)
		err := makeBackgroundTxs(backgroundTxsCtx, l2info, testNodeA.L2Client)
		if !errors.Is(err, context.Canceled) {
			log.Warn("error making background txs", "err", err)
		}
	})()

	stakerATxs := 0
	stakerAWasStaked := false
	stakerBTxs := 0
	stakerBWasStaked := false
	sawStakerZombie := false
	challengeMangerTimedOut := false
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
			if strings.Contains(err.Error(), "agreed with entire challenge") || strings.Contains(err.Error(), "after msg 0 expected global state") {
				// Expected error upon realizing you're losing the challenge. Get ready for a timeout.
				if !challengeMangerTimedOut {
					// Upgrade the ChallengeManager contract to an implementation which says challenges are always timed out

					mockImpl, tx, _, err := mocksgen.DeployTimedOutChallengeManager(&deployAuth, testNodeA.L1Client)
					Require(t, err)
					_, err = EnsureTxSucceeded(ctx, testNodeA.L1Client, tx)
					Require(t, err)

					managerAddr := valWalletA.ChallengeManagerAddress()
					// 0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103
					proxyAdminSlot := common.BigToHash(arbmath.BigSub(crypto.Keccak256Hash([]byte("eip1967.proxy.admin")).Big(), common.Big1))
					proxyAdminBytes, err := testNodeA.L1Client.StorageAt(ctx, managerAddr, proxyAdminSlot, nil)
					Require(t, err)
					proxyAdminAddr := common.BytesToAddress(proxyAdminBytes)
					if proxyAdminAddr == (common.Address{}) {
						Fatal(t, "failed to get challenge manager proxy admin")
					}

					proxyAdmin, err := mocksgen.NewProxyAdminForBinding(proxyAdminAddr, testNodeA.L1Client)
					Require(t, err)
					tx, err = proxyAdmin.Upgrade(&deployAuth, managerAddr, mockImpl)
					Require(t, err)
					_, err = EnsureTxSucceeded(ctx, testNodeA.L1Client, tx)
					Require(t, err)

					challengeMangerTimedOut = true
				}
			} else if strings.Contains(err.Error(), "insufficient funds") && sawStakerZombie {
				// Expected error when trying to re-stake after losing initial stake.
			} else if strings.Contains(err.Error(), "start state not in chain") && sawStakerZombie {
				// Expected error when trying to re-stake after the challenger's nodes getting confirmed.
			} else if strings.Contains(err.Error(), "STAKER_IS_ZOMBIE") && sawStakerZombie {
				// Expected error when the staker is a zombie and thus can't advance its stake.
			} else {
				Require(t, err, "Faulty staker failed to act")
			}
			t.Log("got expected faulty staker error", err)
			err = nil
			tx = nil
		}
		Require(t, err, "Staker", stakerName, "failed to act")
		if tx != nil {
			_, err = EnsureTxSucceeded(ctx, testNodeA.L1Client, tx)
			Require(t, err, "EnsureTxSucceeded failed for staker", stakerName, "tx")
		}
		if faultyStaker {
			conflictInfo, err := validatorUtils.FindStakerConflict(&bind.CallOpts{}, testNodeA.L2Node.DeployInfo.Rollup, l1authA.From, l1authB.From, big.NewInt(1024))
			Require(t, err)
			if staker.ConflictType(conflictInfo.Ty) == staker.CONFLICT_TYPE_FOUND {
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
			Fatal(t, "staker A became a zombie")
		}
		fmt.Printf("watchtower staker acting:\n")
		watchTx, err := stakerC.Act(ctx)
		if err != nil && !strings.Contains(err.Error(), "catch up") {
			Require(t, err, "watchtower staker failed to act")
		}
		if watchTx != nil {
			Fatal(t, "watchtower staker made a transaction")
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
			testNodeA.TransferBalanceViaL1(t, "Faucet", "Faucet", common.Big0)
		}
	}

	if stakerATxs == 0 || stakerBTxs == 0 {
		Fatal(t, "staker didn't make txs: staker A made", stakerATxs, "staker B made", stakerBTxs)
	}

	latestConfirmedNode, err := rollup.LatestConfirmed(&bind.CallOpts{})
	Require(t, err)

	if latestConfirmedNode <= 1 && !honestStakerInactive {
		latestCreatedNode, err := rollup.LatestNodeCreated(&bind.CallOpts{})
		Require(t, err)
		Fatal(t, "latest confirmed node didn't advance:", latestConfirmedNode, latestCreatedNode)
	}

	if faultyStaker && !sawStakerZombie {
		Fatal(t, "staker B didn't become a zombie despite being faulty")
	}

	if !stakerAWasStaked {
		Fatal(t, "staker A was never staked")
	}
	if !stakerBWasStaked {
		Fatal(t, "staker B was never staked")
	}
}

func TestStakersCooperative(t *testing.T) {
	stakerTestImpl(t, false, false)
}
