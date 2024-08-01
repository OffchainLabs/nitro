// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package arbtest

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter/externalsignertest"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/solgen/go/contractsgen"
	"github.com/offchainlabs/nitro/solgen/go/proxiesgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/solgen/go/upgrade_executorgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/validatorwallet"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func TestFastConfirmation(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	srv := externalsignertest.NewServer(t)
	go func() {
		if err := srv.Start(); err != nil {
			log.Error("Failed to start external signer server:", err)
			return
		}
	}()
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithProdConfirmPeriodBlocks()
	builder.L2Info = NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(builder.chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)

	builder.nodeConfig.BatchPoster.MaxDelay = -1000 * time.Hour
	cleanup := builder.Build(t)
	defer cleanup()

	addNewBatchPoster(ctx, t, builder, srv.Address)

	builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
		builder.L1Info.PrepareTxTo("Faucet", &srv.Address, 30000, big.NewInt(1).Mul(big.NewInt(1e18), big.NewInt(1e18)), nil)})

	l2node := builder.L2.ConsensusNode
	execNode := builder.L2.ExecNode

	config := arbnode.ConfigDefaultL1Test()
	config.Sequencer = false
	config.DelayedSequencer.Enable = false
	config.BatchPoster.Enable = false
	builder.execConfig.Sequencer.Enable = false

	builder.BridgeBalance(t, "Faucet", big.NewInt(1).Mul(big.NewInt(params.Ether), big.NewInt(10000)))

	deployAuth := builder.L1Info.GetDefaultTransactOpts("RollupOwner", ctx)

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	builder.L1.TransferBalance(t, "Faucet", "Validator", balance, builder.L1Info)
	l1auth := builder.L1Info.GetDefaultTransactOpts("Validator", ctx)

	valWalletAddrPtr, err := validatorwallet.GetValidatorWalletContract(ctx, l2node.DeployInfo.ValidatorWalletCreator, 0, &l1auth, l2node.L1Reader, true)
	Require(t, err)
	valWalletAddr := *valWalletAddrPtr
	valWalletAddrCheck, err := validatorwallet.GetValidatorWalletContract(ctx, l2node.DeployInfo.ValidatorWalletCreator, 0, &l1auth, l2node.L1Reader, true)
	Require(t, err)
	if valWalletAddr == *valWalletAddrCheck {
		Require(t, err, "didn't cache validator wallet address", valWalletAddr.String(), "vs", valWalletAddrCheck.String())
	}

	rollup, err := rollupgen.NewRollupAdminLogic(l2node.DeployInfo.Rollup, builder.L1.Client)
	Require(t, err)

	upgradeExecutor, err := upgrade_executorgen.NewUpgradeExecutor(l2node.DeployInfo.UpgradeExecutor, builder.L1.Client)
	Require(t, err, "unable to bind upgrade executor")
	rollupABI, err := abi.JSON(strings.NewReader(rollupgen.RollupAdminLogicABI))
	Require(t, err, "unable to parse rollup ABI")

	setValidatorCalldata, err := rollupABI.Pack("setValidator", []common.Address{valWalletAddr, srv.Address}, []bool{true, true})
	Require(t, err, "unable to generate setValidator calldata")
	tx, err := upgradeExecutor.ExecuteCall(&deployAuth, l2node.DeployInfo.Rollup, setValidatorCalldata)
	Require(t, err, "unable to set validators")
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	setMinAssertPeriodCalldata, err := rollupABI.Pack("setMinimumAssertionPeriod", big.NewInt(1))
	Require(t, err, "unable to generate setMinimumAssertionPeriod calldata")
	tx, err = upgradeExecutor.ExecuteCall(&deployAuth, l2node.DeployInfo.Rollup, setMinAssertPeriodCalldata)
	Require(t, err, "unable to set minimum assertion period")
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	setAnyTrustFastConfirmerCalldata, err := rollupABI.Pack("setAnyTrustFastConfirmer", valWalletAddr)
	Require(t, err, "unable to generate setAnyTrustFastConfirmer calldata")
	tx, err = upgradeExecutor.ExecuteCall(&deployAuth, l2node.DeployInfo.Rollup, setAnyTrustFastConfirmerCalldata)
	Require(t, err, "unable to set anytrust fast confirmer")
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	valConfig := staker.TestL1ValidatorConfig
	parentChainID, err := builder.L1.Client.ChainID(ctx)
	if err != nil {
		t.Fatalf("Failed to get parent chain id: %v", err)
	}
	dp, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(l2node.ArbDB, storage.StakerPrefix),
		l2node.L1Reader,
		&l1auth, NewFetcherFromConfig(arbnode.ConfigDefaultL1NonSequencerTest()),
		nil,
		parentChainID,
	)
	if err != nil {
		t.Fatalf("Error creating validator dataposter: %v", err)
	}
	valWallet, err := validatorwallet.NewContract(dp, nil, l2node.DeployInfo.ValidatorWalletCreator, l2node.DeployInfo.Rollup, l2node.L1Reader, &l1auth, 0, func(common.Address) {}, func() uint64 { return valConfig.ExtraGas })
	Require(t, err)
	valConfig.Strategy = "MakeNodes"

	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	blockValidatorConfig := staker.TestBlockValidatorConfig

	stateless, err := staker.NewStatelessBlockValidator(
		l2node.InboxReader,
		l2node.InboxTracker,
		l2node.TxStreamer,
		execNode,
		l2node.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	err = stateless.Start(ctx)
	Require(t, err)
	stakerA, err := staker.NewStaker(
		l2node.L1Reader,
		valWallet,
		bind.CallOpts{},
		valConfig,
		nil,
		stateless,
		nil,
		nil,
		l2node.DeployInfo.ValidatorUtils,
		nil,
	)
	Require(t, err)
	err = stakerA.Initialize(ctx)
	if stakerA.Strategy() != staker.WatchtowerStrategy {
		err = valWallet.Initialize(ctx)
		Require(t, err)
	}
	Require(t, err)
	cfg := arbnode.ConfigDefaultL1NonSequencerTest()
	signerCfg, err := externalSignerTestCfg(srv.Address, srv.URL())
	if err != nil {
		t.Fatalf("Error getting external signer config: %v", err)
	}
	cfg.Staker.DataPoster.ExternalSigner = *signerCfg

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

	latestConfirmBeforeAct, err := rollup.LatestConfirmed(&bind.CallOpts{})
	Require(t, err)
	tx, err = stakerA.Act(ctx)
	Require(t, err)
	if tx != nil {
		_, err = builder.L1.EnsureTxSucceeded(tx)
		Require(t, err)
	}
	latestConfirmAfterAct, err := rollup.LatestConfirmed(&bind.CallOpts{})
	Require(t, err)
	if latestConfirmAfterAct <= latestConfirmBeforeAct {
		Fatal(t, "staker A didn't advance the latest confirmed node")
	}
}

func TestFastConfirmationWithSafe(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	srv := externalsignertest.NewServer(t)
	go func() {
		if err := srv.Start(); err != nil {
			log.Error("Failed to start external signer server:", err)
			return
		}
	}()
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs

	// Create a node with a large confirm period to ensure that the staker can't confirm without the fast confirmer.
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithProdConfirmPeriodBlocks()
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
	if nodeAGenesis != nodeBGenesis {
		Fatal(t, "node A L2 genesis hash", nodeAGenesis, "!= node B L2 genesis hash", nodeBGenesis)
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

	safeAddress := deploySafe(t, builder.L1, builder.L1.Client, deployAuth, []common.Address{valWalletAddrA, srv.Address})
	setValidatorCalldata, err := rollupABI.Pack("setValidator", []common.Address{valWalletAddrA, l1authB.From, srv.Address, safeAddress}, []bool{true, true, true, true})
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

	setAnyTrustFastConfirmerCalldata, err := rollupABI.Pack("setAnyTrustFastConfirmer", safeAddress)
	Require(t, err, "unable to generate setAnyTrustFastConfirmer calldata")
	tx, err = upgradeExecutor.ExecuteCall(&deployAuth, l2nodeA.DeployInfo.Rollup, setAnyTrustFastConfirmerCalldata)
	Require(t, err, "unable to set anytrust fast confirmer")
	_, err = builder.L1.EnsureTxSucceeded(tx)
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
	valConfig.Strategy = "MakeNodes"

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
		nil,
	)
	Require(t, err)
	err = stakerA.Initialize(ctx)
	Require(t, err)
	err = valWalletA.Initialize(ctx)
	Require(t, err)
	cfg := arbnode.ConfigDefaultL1NonSequencerTest()
	signerCfg, err := externalSignerTestCfg(srv.Address, srv.URL())
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
	valConfig.Strategy = "watchtower"
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
		nil,
	)
	Require(t, err)
	err = stakerB.Initialize(ctx)
	Require(t, err)
	err = valWalletB.Initialize(ctx)
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

	latestConfirmBeforeAct, err := rollup.LatestConfirmed(&bind.CallOpts{})
	Require(t, err)
	tx, err = stakerA.Act(ctx)
	Require(t, err)
	if tx != nil {
		_, err = builder.L1.EnsureTxSucceeded(tx)
		Require(t, err)
	}
	latestConfirmAfterStakerAAct, err := rollup.LatestConfirmed(&bind.CallOpts{})
	Require(t, err)
	if latestConfirmAfterStakerAAct != latestConfirmBeforeAct {
		Fatal(t, "staker A alone advanced the latest confirmed node", latestConfirmAfterStakerAAct, "when it shouldn't have")
	}
	for j := 0; j < 5; j++ {
		builder.L1.TransferBalance(t, "Faucet", "Faucet", common.Big0, builder.L1Info)
	}
	tx, err = stakerB.Act(ctx)
	Require(t, err)
	if tx != nil {
		_, err = builder.L1.EnsureTxSucceeded(tx)
		Require(t, err)
	}
	latestConfirmAfterStakerBAct, err := rollup.LatestConfirmed(&bind.CallOpts{})
	Require(t, err)
	if latestConfirmAfterStakerBAct <= latestConfirmBeforeAct {
		Fatal(t, "staker A and B together didn't advance the latest confirmed node")
	}
}

func deploySafe(t *testing.T, l1 *TestClient, backend bind.ContractBackend, deployAuth bind.TransactOpts, owners []common.Address) common.Address {
	safeAddress, tx, _, err := contractsgen.DeploySafeL2(&deployAuth, backend)
	Require(t, err)
	_, err = l1.EnsureTxSucceeded(tx)
	Require(t, err)
	safeProxyAddress, tx, _, err := proxiesgen.DeploySafeProxy(&deployAuth, backend, safeAddress)
	Require(t, err)
	_, err = l1.EnsureTxSucceeded(tx)
	Require(t, err)
	var safe *contractsgen.Safe
	safe, err = contractsgen.NewSafe(safeProxyAddress, backend)
	Require(t, err)
	_, err = l1.EnsureTxSucceeded(tx)
	Require(t, err)
	tx, err = safe.Setup(
		&deployAuth,
		owners,
		big.NewInt(2),
		common.Address{},
		nil,
		common.Address{},
		common.Address{},
		big.NewInt(0),
		common.Address{},
	)
	Require(t, err)
	_, err = l1.EnsureTxSucceeded(tx)
	Require(t, err)
	return safeProxyAddress
}
