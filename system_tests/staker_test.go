// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package arbtest

import (
	"context"
	"encoding/json"
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
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	mocksgen_bold "github.com/OffchainLabs/bold/solgen/go/mocksgen"
	rollupgen_bold "github.com/OffchainLabs/bold/solgen/go/rollupgen"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/OffchainLabs/bold/testing/setup"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/validatorwallet"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/validator/server_common"
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
	_, l2nodeA, l2clientA, _, l1info, _, l1client, l1stack := createTestNodeOnL1WithConfigImpl(t, ctx, true, nil, nil, l2chainConfig, nil, l2info)
	defer requireClose(t, l1stack)
	defer l2nodeA.StopAndWait()
	execNodeA := getExecNode(t, l2nodeA)

	if faultyStaker {
		l2info.GenerateGenesisAccount("FaultyAddr", common.Big1)
	}
	config := arbnode.ConfigDefaultL1Test()
	execConfig := gethexec.ConfigDefaultTest()
	execConfig.Sequencer.Enable = false
	config.Sequencer = false
	config.DelayedSequencer.Enable = false
	config.BatchPoster.Enable = false
	_, l2nodeB := Create2ndNodeWithConfig(t, ctx, l2nodeA, l1stack, l1info, &l2info.ArbInitData, config, execConfig, nil)
	defer l2nodeB.StopAndWait()
	execNodeB := getExecNode(t, l2nodeB)

	nodeAGenesis := execNodeA.Backend.APIBackend().CurrentHeader().Hash()
	nodeBGenesis := execNodeB.Backend.APIBackend().CurrentHeader().Hash()
	if faultyStaker {
		if nodeAGenesis == nodeBGenesis {
			Fatal(t, "node A L2 genesis hash", nodeAGenesis, "== node B L2 genesis hash", nodeBGenesis)
		}
	} else {
		if nodeAGenesis != nodeBGenesis {
			Fatal(t, "node A L2 genesis hash", nodeAGenesis, "!= node B L2 genesis hash", nodeBGenesis)
		}
	}

	BridgeBalance(t, "Faucet", big.NewInt(1).Mul(big.NewInt(params.Ether), big.NewInt(10000)), l1info, l2info, l1client, l2clientA, ctx)

	deployAuth := l1info.GetDefaultTransactOpts("RollupOwner", ctx)

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	l1info.GenerateAccount("ValidatorA")
	TransferBalance(t, "Faucet", "ValidatorA", balance, l1info, l1client, ctx)
	l1authA := l1info.GetDefaultTransactOpts("ValidatorA", ctx)

	l1info.GenerateAccount("ValidatorB")
	TransferBalance(t, "Faucet", "ValidatorB", balance, l1info, l1client, ctx)
	l1authB := l1info.GetDefaultTransactOpts("ValidatorB", ctx)

	valWalletAddrAPtr, err := validatorwallet.GetValidatorWalletContract(ctx, l2nodeA.DeployInfo.ValidatorWalletCreator, 0, &l1authA, l2nodeA.L1Reader, true)
	Require(t, err)
	valWalletAddrA := *valWalletAddrAPtr
	valWalletAddrCheck, err := validatorwallet.GetValidatorWalletContract(ctx, l2nodeA.DeployInfo.ValidatorWalletCreator, 0, &l1authA, l2nodeA.L1Reader, true)
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

	validatorUtils, err := rollupgen.NewValidatorUtils(l2nodeA.DeployInfo.ValidatorUtils, l1client)
	Require(t, err)

	valConfig := staker.TestL1ValidatorConfig

	dpA, err := arbnode.StakerDataposter(ctx, rawdb.NewTable(l2nodeB.ArbDB, storage.StakerPrefix), l2nodeA.L1Reader, &l1authA, NewFetcherFromConfig(arbnode.ConfigDefaultL1NonSequencerTest()), nil)
	if err != nil {
		t.Fatalf("Error creating validator dataposter: %v", err)
	}
	valWalletA, err := validatorwallet.NewContract(dpA, nil, l2nodeA.DeployInfo.ValidatorWalletCreator, l2nodeA.DeployInfo.Rollup, l2nodeA.L1Reader, &l1authA, 0, func(common.Address) {}, func() uint64 { return valConfig.ExtraGas })
	Require(t, err)
	if honestStakerInactive {
		valConfig.Strategy = "Defensive"
	} else {
		valConfig.Strategy = "MakeNodes"
	}

	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	blockValidatorConfig := staker.TestBlockValidatorConfig

	statelessA, err := staker.NewStatelessBlockValidator(
		l2nodeA.InboxReader,
		l2nodeA.InboxTracker,
		l2nodeA.TxStreamer,
		execNodeA,
		l2nodeA.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	err = statelessA.Start(ctx)
	Require(t, err)
	stakerA, err := staker.NewStaker(
		l2nodeA.L1Reader,
		valWalletA,
		bind.CallOpts{},
		valConfig,
		nil,
		statelessA,
		nil,
		nil,
		l2nodeA.DeployInfo.ValidatorUtils,
		l2nodeA.DeployInfo.Bridge,
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
		execNodeB,
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
		l2nodeB.DeployInfo.Bridge,
		nil,
	)
	Require(t, err)
	err = stakerB.Initialize(ctx)
	Require(t, err)
	if stakerB.Strategy() != staker.WatchtowerStrategy {
		err = valWalletB.Initialize(ctx)
		Require(t, err)
	}
	valWalletC := validatorwallet.NewNoOp(l1client, l2nodeA.DeployInfo.Rollup)
	valConfig.Strategy = "Watchtower"
	stakerC, err := staker.NewStaker(
		l2nodeA.L1Reader,
		valWalletC,
		bind.CallOpts{},
		valConfig,
		nil,
		statelessA,
		nil,
		nil,
		l2nodeA.DeployInfo.ValidatorUtils,
		l2nodeA.DeployInfo.Bridge,
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
	err = l2clientA.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2clientA, tx)
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
		err := makeBackgroundTxs(backgroundTxsCtx, l2info, l2clientA)
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

					mockImpl, tx, _, err := mocksgen.DeployTimedOutChallengeManager(&deployAuth, l1client)
					Require(t, err)
					_, err = EnsureTxSucceeded(ctx, l1client, tx)
					Require(t, err)

					managerAddr := valWalletA.ChallengeManagerAddress()
					// 0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103
					proxyAdminSlot := common.BigToHash(arbmath.BigSub(crypto.Keccak256Hash([]byte("eip1967.proxy.admin")).Big(), common.Big1))
					proxyAdminBytes, err := l1client.StorageAt(ctx, managerAddr, proxyAdminSlot, nil)
					Require(t, err)
					proxyAdminAddr := common.BytesToAddress(proxyAdminBytes)
					if proxyAdminAddr == (common.Address{}) {
						Fatal(t, "failed to get challenge manager proxy admin")
					}

					proxyAdmin, err := mocksgen.NewProxyAdminForBinding(proxyAdminAddr, l1client)
					Require(t, err)
					tx, err = proxyAdmin.Upgrade(&deployAuth, managerAddr, mockImpl)
					Require(t, err)
					_, err = EnsureTxSucceeded(ctx, l1client, tx)
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
			_, err = EnsureTxSucceeded(ctx, l1client, tx)
			Require(t, err, "EnsureTxSucceeded failed for staker", stakerName, "tx")
		}
		if faultyStaker {
			conflictInfo, err := validatorUtils.FindStakerConflict(&bind.CallOpts{}, l2nodeA.DeployInfo.Rollup, l1authA.From, l1authB.From, big.NewInt(1024))
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
			TransferBalance(t, "Faucet", "Faucet", common.Big0, l1info, l1client, ctx)
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

func TestStakerSwitchDuringRollupUpgrade(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	stakerImpl, l1info, l1client, l2chainConfig, l2node, deployAuth := setupNonBoldStaker(t, ctx)
	defer l2node.StopAndWait()

	err := stakerImpl.Initialize(ctx)
	Require(t, err)
	stakerImpl.Start(ctx)
	if stakerImpl.Stopped() {
		t.Fatal("Old protocol staker not started")
	}

	rollupAddresses := deployBoldContracts(t, ctx, l1info, l1client, l2chainConfig.ChainID, deployAuth)

	bridge, err := bridgegen.NewBridge(l2node.DeployInfo.Bridge, l1client)
	Require(t, err)
	tx, err := bridge.UpdateRollupAddress(&deployAuth, rollupAddresses.Rollup)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)

	time.Sleep(time.Second)

	if !stakerImpl.Stopped() {
		t.Fatal("Old protocol staker not stopped after rollup upgrade")
	}
}

func setupNonBoldStaker(t *testing.T, ctx context.Context) (*staker.Staker, info, *ethclient.Client, *params.ChainConfig, *arbnode.Node, bind.TransactOpts) {
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs
	l2chainConfig := params.ArbitrumDevTestChainConfig()
	l2info := NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(l2chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	_, l2node, l2client, _, l1info, _, l1client, _ := createTestNodeOnL1WithConfigImpl(t, ctx, true, nil, l2chainConfig, nil, nil, l2info)

	config := arbnode.ConfigDefaultL1Test()
	config.Sequencer.Enable = false
	config.DelayedSequencer.Enable = false
	config.BatchPoster.Enable = false

	BridgeBalance(t, "Faucet", big.NewInt(1).Mul(big.NewInt(params.Ether), big.NewInt(10000)), l1info, l2info, l1client, l2client, ctx)

	deployAuth := l1info.GetDefaultTransactOpts("RollupOwner", ctx)

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	l1info.GenerateAccount("Validator")
	TransferBalance(t, "Faucet", "Validator", balance, l1info, l1client, ctx)
	l1auth := l1info.GetDefaultTransactOpts("Validator", ctx)

	rollup, err := rollupgen.NewRollupAdminLogic(l2node.DeployInfo.Rollup, l1client)
	Require(t, err)

	tx, err := rollup.SetMinimumAssertionPeriod(&deployAuth, big.NewInt(1))
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)
	valConfig := staker.DefaultL1ValidatorConfig
	valConfig.Strategy = "WatchTower"
	valConfig.EnableBold = true
	valConfig.StakerInterval = 100 * time.Millisecond

	dp, err := arbnode.StakerDataposter(rawdb.NewTable(l2node.ArbDB, storage.StakerPrefix), l2node.L1Reader, &l1auth, NewFetcherFromConfig(arbnode.ConfigDefaultL1NonSequencerTest()), nil)
	if err != nil {
		t.Fatalf("Error creating validator dataposter: %v", err)
	}
	valWallet, err := staker.NewContractValidatorWallet(dp, nil, l2node.DeployInfo.ValidatorWalletCreator, l2node.DeployInfo.Rollup, l2node.L1Reader, &l1auth, 0, func(common.Address) {}, func() uint64 { return valConfig.ExtraGas })
	Require(t, err)
	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	blockValidatorConfig := staker.TestBlockValidatorConfig

	stateless, err := staker.NewStatelessBlockValidator(
		l2node.InboxReader,
		l2node.InboxTracker,
		l2node.TxStreamer,
		l2node.Execution.Recorder,
		l2node.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	err = stateless.Start(ctx)
	Require(t, err)
	stakerImpl, err := staker.NewStaker(
		l2node.L1Reader,
		valWallet,
		bind.CallOpts{},
		valConfig,
		nil,
		stateless,
		nil,
		nil,
		l2node.DeployInfo.ValidatorUtils,
		l2node.DeployInfo.Bridge,
		nil,
	)
	Require(t, err)
	return stakerImpl, l1info, l1client, l2chainConfig, l2node, deployAuth
}

func deployBoldContracts(
	t *testing.T,
	ctx context.Context,
	l1info info,
	backend *ethclient.Client,
	chainId *big.Int,
	deployAuth bind.TransactOpts,
) *chaininfo.RollupAddresses {
	stakeToken, tx, tokenBindings, err := mocksgen_bold.DeployTestWETH9(
		&deployAuth,
		backend,
		"Weth",
		"WETH",
	)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)
	value, _ := new(big.Int).SetString("1000000", 10)
	deployAuth.Value = value
	tx, err = tokenBindings.Deposit(&deployAuth)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)
	deployAuth.Value = nil
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)

	initialBalance := new(big.Int).Lsh(big.NewInt(1), 200)
	l1info.GenerateGenesisAccount("deployer", initialBalance)
	l1info.GenerateGenesisAccount("asserter", initialBalance)
	l1info.GenerateGenesisAccount("sequencer", initialBalance)
	SendWaitTestTransactions(t, ctx, backend, []*types.Transaction{
		l1info.PrepareTx("Faucet", "RollupOwner", 30000, initialBalance, nil)})
	l1TransactionOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	locator, err := server_common.NewMachineLocator("")
	Require(t, err)

	cfg := challenge_testing.GenerateRollupConfig(
		false,
		locator.LatestWasmModuleRoot(),
		l1TransactionOpts.From,
		chainId,
		common.Address{},
		big.NewInt(1),
		stakeToken,
		rollupgen_bold.ExecutionState{
			GlobalState:   rollupgen_bold.GlobalState{},
			MachineStatus: 1,
		},
		big.NewInt(0),
		common.Address{},
	)
	config, err := json.Marshal(params.ArbitrumDevTestChainConfig())
	if err != nil {
		return nil
	}
	cfg.ChainConfig = string(config)

	addresses, err := setup.DeployFullRollupStack(
		ctx,
		backend,
		&l1TransactionOpts,
		l1info.GetAddress("sequencer"),
		cfg,
		false,
	)
	Require(t, err)

	return &chaininfo.RollupAddresses{
		Bridge:                 addresses.Bridge,
		Inbox:                  addresses.Inbox,
		SequencerInbox:         addresses.SequencerInbox,
		Rollup:                 addresses.Rollup,
		ValidatorUtils:         addresses.ValidatorUtils,
		ValidatorWalletCreator: addresses.ValidatorWalletCreator,
		DeployedAt:             addresses.DeployedAt,
	}
}
