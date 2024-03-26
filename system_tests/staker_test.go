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
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
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
	"github.com/offchainlabs/nitro/arbnode/dataposter/externalsignertest"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/solgen/go/upgrade_executorgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/validatorwallet"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func makeBackgroundTxs(ctx context.Context, builder *NodeBuilder) error {
	for i := uint64(0); ctx.Err() == nil; i++ {
		builder.L2Info.Accounts["BackgroundUser"].Nonce = i
		tx := builder.L2Info.PrepareTx("BackgroundUser", "BackgroundUser", builder.L2Info.TransferGas, common.Big0, nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		if err != nil {
			return err
		}
		_, err = builder.L2.EnsureTxSucceeded(tx)
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
	httpSrv, srv := externalsignertest.NewServer(t)
	cp, err := externalsignertest.CertPaths()
	if err != nil {
		t.Fatalf("Error getting cert paths: %v", err)
	}
	t.Cleanup(func() {
		if err := httpSrv.Shutdown(ctx); err != nil {
			t.Fatalf("Error shutting down http server: %v", err)
		}
	})
	go func() {
		log.Debug("Server is listening on port 1234...")
		if err := httpSrv.ListenAndServeTLS(cp.ServerCert, cp.ServerKey); err != nil && err != http.ErrServerClosed {
			log.Debug("ListenAndServeTLS() failed", "error", err)
			return
		}
	}()
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.L2Info = NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(builder.chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)

	builder.nodeConfig.BatchPoster.MaxDelay = -1000 * time.Hour
	cleanupA := builder.Build(t)
	defer cleanupA()

	addNewBatchPoster(ctx, t, builder, srv.Address)

	builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
		builder.L1Info.PrepareTxTo("Faucet", &srv.Address, 30000, big.NewInt(1).Mul(big.NewInt(1e18), big.NewInt(1e18)), nil)})

	l2nodeA := builder.L2.ConsensusNode
	execNodeA := builder.L2.ExecNode

	if faultyStaker {
		builder.L2Info.GenerateGenesisAccount("FaultyAddr", common.Big1)
	}

	config := arbnode.ConfigDefaultL1Test()
	config.Sequencer = false
	config.DelayedSequencer.Enable = false
	config.BatchPoster.Enable = false
	builder.execConfig.Sequencer.Enable = false
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: config})
	defer cleanupB()

	l2nodeB := testClientB.ConsensusNode
	execNodeB := testClientB.ExecNode

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

	builder.BridgeBalance(t, "Faucet", big.NewInt(1).Mul(big.NewInt(params.Ether), big.NewInt(10000)))

	deployAuth := builder.L1Info.GetDefaultTransactOpts("RollupOwner", ctx)

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	builder.L1Info.GenerateAccount("ValidatorA")
	builder.L1.TransferBalance(t, "Faucet", "ValidatorA", balance, builder.L1Info)
	l1authA := builder.L1Info.GetDefaultTransactOpts("ValidatorA", ctx)

	builder.L1Info.GenerateAccount("ValidatorB")
	builder.L1.TransferBalance(t, "Faucet", "ValidatorB", balance, builder.L1Info)
	l1authB := builder.L1Info.GetDefaultTransactOpts("ValidatorB", ctx)

	valWalletAddrAPtr, err := validatorwallet.GetValidatorWalletContract(ctx, l2nodeA.DeployInfo.ValidatorWalletCreator, 0, &l1authA, l2nodeA.L1Reader, true)
	Require(t, err)
	valWalletAddrA := *valWalletAddrAPtr
	valWalletAddrCheck, err := validatorwallet.GetValidatorWalletContract(ctx, l2nodeA.DeployInfo.ValidatorWalletCreator, 0, &l1authA, l2nodeA.L1Reader, true)
	Require(t, err)
	if valWalletAddrA == *valWalletAddrCheck {
		Require(t, err, "didn't cache validator wallet address", valWalletAddrA.String(), "vs", valWalletAddrCheck.String())
	}

	rollup, err := rollupgen.NewRollupAdminLogic(l2nodeA.DeployInfo.Rollup, builder.L1.Client)
	Require(t, err)

	upgradeExecutor, err := upgrade_executorgen.NewUpgradeExecutor(l2nodeA.DeployInfo.UpgradeExecutor, builder.L1.Client)
	Require(t, err, "unable to bind upgrade executor")
	rollupABI, err := abi.JSON(strings.NewReader(rollupgen.RollupAdminLogicABI))
	Require(t, err, "unable to parse rollup ABI")

	setValidatorCalldata, err := rollupABI.Pack("setValidator", []common.Address{valWalletAddrA, l1authB.From, srv.Address}, []bool{true, true, true})
	Require(t, err, "unable to generate setValidator calldata")
	tx, err := upgradeExecutor.ExecuteCall(&deployAuth, l2nodeA.DeployInfo.Rollup, setValidatorCalldata)
	Require(t, err, "unable to set validators")
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	setMinAssertPeriodCalldata, err := rollupABI.Pack("setMinimumAssertionPeriod", big.NewInt(1))
	Require(t, err, "unable to generate setMinimumAssertionPeriod calldata")
	tx, err = upgradeExecutor.ExecuteCall(&deployAuth, l2nodeA.DeployInfo.Rollup, setMinAssertPeriodCalldata)
	Require(t, err, "unable to set minimum assertion period")
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	validatorUtils, err := rollupgen.NewValidatorUtils(l2nodeA.DeployInfo.ValidatorUtils, builder.L1.Client)
	Require(t, err)

	valConfig := staker.TestL1ValidatorConfig
	parentChainID, err := builder.L1.Client.ChainID(ctx)
	if err != nil {
		t.Fatalf("Failed to get parent chain id: %v", err)
	}
	dpA, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(l2nodeB.ArbDB, storage.StakerPrefix),
		l2nodeA.L1Reader,
		&l1authA, NewFetcherFromConfig(arbnode.ConfigDefaultL1NonSequencerTest()),
		nil,
		parentChainID,
	)
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
	cfg := arbnode.ConfigDefaultL1NonSequencerTest()
	signerCfg, err := externalSignerTestCfg(srv.Address)
	if err != nil {
		t.Fatalf("Error getting external signer config: %v", err)
	}
	cfg.Staker.DataPoster.ExternalSigner = *signerCfg
	dpB, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(l2nodeB.ArbDB, storage.StakerPrefix),
		l2nodeB.L1Reader,
		&l1authB, NewFetcherFromConfig(cfg),
		nil,
		parentChainID,
	)
	if err != nil {
		t.Fatalf("Error creating validator dataposter: %v", err)
	}
	valWalletB, err := validatorwallet.NewEOA(dpB, l2nodeB.DeployInfo.Rollup, l2nodeB.L1Reader.Client(), func() uint64 { return 0 })
	Require(t, err)
	valConfig.Strategy = "MakeNodes"
	statelessB, err := staker.NewStatelessBlockValidator(
		l2nodeB.InboxReader,
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		execNodeB,
		l2nodeB.ArbDB,
		nil,
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
	valWalletC := validatorwallet.NewNoOp(builder.L1.Client, l2nodeA.DeployInfo.Rollup)
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

	builder.L2Info.GenerateAccount("BackgroundUser")
	tx = builder.L2Info.PrepareTx("Faucet", "BackgroundUser", builder.L2Info.TransferGas, balance, nil)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
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
		err := makeBackgroundTxs(backgroundTxsCtx, builder)
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

					mockImpl, tx, _, err := mocksgen.DeployTimedOutChallengeManager(&deployAuth, builder.L1.Client)
					Require(t, err)
					_, err = builder.L1.EnsureTxSucceeded(tx)
					Require(t, err)

					managerAddr := valWalletA.ChallengeManagerAddress()
					// 0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103
					proxyAdminSlot := common.BigToHash(arbmath.BigSub(crypto.Keccak256Hash([]byte("eip1967.proxy.admin")).Big(), common.Big1))
					proxyAdminBytes, err := builder.L1.Client.StorageAt(ctx, managerAddr, proxyAdminSlot, nil)
					Require(t, err)
					proxyAdminAddr := common.BytesToAddress(proxyAdminBytes)
					if proxyAdminAddr == (common.Address{}) {
						Fatal(t, "failed to get challenge manager proxy admin")
					}

					proxyAdminABI, err := abi.JSON(strings.NewReader(mocksgen.ProxyAdminForBindingABI))
					Require(t, err)
					upgradeCalldata, err := proxyAdminABI.Pack("upgrade", managerAddr, mockImpl)
					Require(t, err)
					tx, err = upgradeExecutor.ExecuteCall(&deployAuth, proxyAdminAddr, upgradeCalldata)
					Require(t, err)
					_, err = builder.L1.EnsureTxSucceeded(tx)
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
			_, err = builder.L1.EnsureTxSucceeded(tx)
			Require(t, err, "EnsureTxSucceeded failed for staker", stakerName, "tx")
		}
		if faultyStaker {
			conflictInfo, err := validatorUtils.FindStakerConflict(&bind.CallOpts{}, l2nodeA.DeployInfo.Rollup, l1authA.From, srv.Address, big.NewInt(1024))
			Require(t, err)
			if staker.ConflictType(conflictInfo.Ty) == staker.CONFLICT_TYPE_FOUND {
				cancelBackgroundTxs()
			}
		}
		if faultyStaker && !sawStakerZombie {
			sawStakerZombie, err = rollup.IsZombie(&bind.CallOpts{}, srv.Address)
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
			stakerBWasStaked, err = rollup.IsStaked(&bind.CallOpts{}, srv.Address)
			Require(t, err)
		}
		for j := 0; j < 5; j++ {
			builder.L1.TransferBalance(t, "Faucet", "Faucet", common.Big0, builder.L1Info)
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
	stakerImpl, builder := setupNonBoldStaker(t, ctx)
	deployAuth := builder.L1Info.GetDefaultTransactOpts("RollupOwner", ctx)
	err := stakerImpl.Initialize(ctx)
	Require(t, err)
	stakerImpl.Start(ctx)
	if stakerImpl.Stopped() {
		t.Fatal("Old protocol staker not started")
	}

	rollupAddresses := deployBoldContracts(t, ctx, builder.L1Info, builder.L1.Client, builder.chainConfig.ChainID, deployAuth)

	upgradeExecutor, err := upgrade_executorgen.NewUpgradeExecutor(builder.L2.ConsensusNode.DeployInfo.UpgradeExecutor, builder.L1.Client)
	Require(t, err)
	bridgeABI, err := abi.JSON(strings.NewReader(bridgegen.BridgeABI))
	Require(t, err)

	updateRollupAddressCalldata, err := bridgeABI.Pack("updateRollupAddress", rollupAddresses.Rollup)
	Require(t, err)
	tx, err := upgradeExecutor.ExecuteCall(&deployAuth, builder.L2.ConsensusNode.DeployInfo.Bridge, updateRollupAddressCalldata)
	Require(t, err)
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	time.Sleep(time.Second)

	if !stakerImpl.Stopped() {
		t.Fatal("Old protocol staker not stopped after rollup upgrade")
	}
}

func setupNonBoldStaker(t *testing.T, ctx context.Context) (*staker.Staker, *NodeBuilder) {
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.L2Info = NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(builder.chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	builder.Build(t)
	l2node := builder.L2.ConsensusNode
	l1info := builder.L1Info
	l1client := builder.L1.Client

	builder.BridgeBalance(t, "Faucet", big.NewInt(1).Mul(big.NewInt(params.Ether), big.NewInt(10000)))

	deployAuth := l1info.GetDefaultTransactOpts("RollupOwner", ctx)

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	l1info.GenerateAccount("Validator")
	TransferBalance(t, "Faucet", "Validator", balance, l1info, l1client, ctx)
	l1auth := l1info.GetDefaultTransactOpts("Validator", ctx)

	upgradeExecutor, err := upgrade_executorgen.NewUpgradeExecutor(l2node.DeployInfo.UpgradeExecutor, builder.L1.Client)
	Require(t, err)
	rollupABI, err := abi.JSON(strings.NewReader(rollupgen.RollupAdminLogicABI))
	Require(t, err)

	setMinAssertPeriodCalldata, err := rollupABI.Pack("setMinimumAssertionPeriod", big.NewInt(1))
	Require(t, err)
	tx, err := upgradeExecutor.ExecuteCall(&deployAuth, l2node.DeployInfo.Rollup, setMinAssertPeriodCalldata)
	Require(t, err)
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)
	valConfig := staker.DefaultL1ValidatorConfig
	valConfig.Strategy = "WatchTower"
	valConfig.Bold = staker.DefaultBoldConfig
	valConfig.Bold.Enable = true
	valConfig.StakerInterval = 100 * time.Millisecond

	parentChainID, err := builder.L1.Client.ChainID(ctx)
	if err != nil {
		t.Fatalf("Failed to get parent chain id: %v", err)
	}
	dp, err := arbnode.StakerDataposter(ctx, rawdb.NewTable(l2node.ArbDB, storage.StakerPrefix), l2node.L1Reader, &l1auth, NewFetcherFromConfig(arbnode.ConfigDefaultL1NonSequencerTest()), nil, parentChainID)
	if err != nil {
		t.Fatalf("Error creating validator dataposter: %v", err)
	}
	valWallet, err := validatorwallet.NewContract(dp, nil, l2node.DeployInfo.ValidatorWalletCreator, l2node.DeployInfo.Rollup, l2node.L1Reader, &l1auth, 0, func(common.Address) {}, func() uint64 { return valConfig.ExtraGas })
	Require(t, err)
	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	blockValidatorConfig := staker.TestBlockValidatorConfig

	stateless, err := staker.NewStatelessBlockValidator(
		l2node.InboxReader,
		l2node.InboxTracker,
		l2node.TxStreamer,
		l2node.Execution,
		l2node.ArbDB,
		nil,
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
	return stakerImpl, builder
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
