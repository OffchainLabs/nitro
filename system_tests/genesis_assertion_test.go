// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"encoding/json"
	"math/big"
	gotesting "testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbnode/dataposter/externalsignertest"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/bold/protocol"
	"github.com/offchainlabs/nitro/bold/protocol/sol"
	"github.com/offchainlabs/nitro/bold/testing"
	"github.com/offchainlabs/nitro/bold/testing/setup"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/nitro/init"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker/bold"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/validator/server_common"
)

func TestValidateGenesisAssertion(t *gotesting.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs
	l2chainConfig := chaininfo.ArbitrumDevTestChainConfig()
	l2info := NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(l2chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	ownerBal := big.NewInt(params.Ether)
	ownerBal.Mul(ownerBal, big.NewInt(1_000_000))
	l2info.GenerateGenesisAccount("Owner", ownerBal)
	sconf := setup.RollupStackConfig{
		UseBlobs:               true,
		UseMockBridge:          false,
		UseMockOneStepProver:   false,
		MinimumAssertionPeriod: 0,
	}

	_, l2nodeA, _, _, _, _, _, l1client, l1stack, _, _, _, l2blockchain, addresses := createCompleteTestNodeOnL1(
		t,
		ctx,
		true,
		nil,
		l2chainConfig,
		nil,
		sconf,
		l2info,
		false,
		false,
	)

	if l2blockchain == nil || addresses == nil {
		t.Fatal("Both l2blockchain and addresses have to be non nil")
	}
	defer requireClose(t, l1stack)
	defer l2nodeA.StopAndWait()

	// Chain assertion info contains a BeforeState and AfterState which are used to dictate if genesis
	// assertion is nil or not. When a chain is deployed, a genesis assertion is created. Such new chain
	// deployment can happen from scratch or as part of a chain upgrade. In the first case, genesis
	// chain assertion BeforeState and AfterState will be both zero since there's nothing before genesis.
	// On the other hand, for the latter case, AfterState will be non zero since the chain has been
	// created on an existing chain. In this test environment, we are simulating the first case where a
	// chain is deployed from scratch, so we expect genesis assertion to be nil/zero for both BeforeState
	// and AfterState. To that end, we'll also simulate the same behaviour as nitro-testnode where
	// initDataReader is initialized with config.Init.Empty set to true, meaning ArbosInitializationInfo
	// is initialized like below instead of using l2info.ArbInitData
	initData := statetransfer.ArbosInitializationInfo{
		NextBlockNumber: 0,
	}
	initDataReader := statetransfer.NewMemoryInitDataReader(&initData)
	if initDataReader == nil {
		t.Fatal("initDataReader can't be nil")
	}

	err := nitroinit.GetAndValidateGenesisAssertion(ctx, l2blockchain, initDataReader, addresses, l1client)
	Require(t, err)
}

func TestValidateGenesisAssertionWithBuilder(t *gotesting.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	// Chain assertion info contains a BeforeState and AfterState which are used to dictate if genesis
	// assertion is nil or not. When a chain is deployed, a genesis assertion is created. Such new chain
	// deployment can happen from scratch or as part of a chain upgrade. In the first case, genesis
	// chain assertion BeforeState and AfterState will be both zero since there's nothing before genesis.
	// On the other hand, for the latter case, AfterState will be non zero since the chain has been
	// created on an existing chain. In this test environment, we are simulating the first case where a
	// chain is deployed from scratch, so we expect genesis assertion to be nil/zero for both BeforeState
	// and AfterState. To that end, we'll also simulate the same behaviour as nitro-testnode where
	// initDataReader is initialized with config.Init.Empty set to true, meaning ArbosInitializationInfo
	// is initialized like below instead of using l2info.ArbInitData
	initData := statetransfer.ArbosInitializationInfo{
		NextBlockNumber: 0,
	}
	initDataReader := statetransfer.NewMemoryInitDataReader(&initData)
	if initDataReader == nil {
		t.Fatal("initDataReader can't be nil")
	}

	err := nitroinit.GetAndValidateGenesisAssertion(ctx, builder.L2.ExecNode.Backend.ArbInterface().BlockChain(), initDataReader, builder.addresses, builder.L1.Client)
	Require(t, err)
}

func createCompleteTestNodeOnL1(
	t *gotesting.T,
	ctx context.Context,
	isSequencer bool,
	nodeConfig *arbnode.Config,
	chainConfig *params.ChainConfig,
	_ *node.Config,
	rollupStackConf setup.RollupStackConfig,
	l2infoIn info,
	useExternalSigner bool,
	enableCustomDA bool,
) (
	l2info info, currentNode *arbnode.Node, execNode *gethexec.ExecutionNode, l2client *ethclient.Client, l2stack *node.Node,
	l1info info, l1backend *eth.Ethereum, l1client *ethclient.Client, l1stack *node.Node,
	assertionChain *sol.AssertionChain, stakeTokenAddr common.Address, asserterOpts *bind.TransactOpts, l2blockchain *core.BlockChain, addresses *chaininfo.RollupAddresses,
) {
	// First set up L1 and deploy contracts
	var signerCfg *dataposter.ExternalSignerCfg
	l1info, l1backend, l1client, l1stack, addresses, stakeTokenAddr, asserterOpts, signerCfg = setupL1WithRollupAddresses(
		t, ctx, rollupStackConf, useExternalSigner, nodeConfig, chainConfig, enableCustomDA,
	)

	// Then create L2 node
	l2info, currentNode, execNode, l2client, l2stack, assertionChain, l2blockchain = createL2NodeWithRollupAddresses(
		t, ctx, isSequencer, nodeConfig, chainConfig, l2infoIn,
		l1info, l1client, addresses,
		useExternalSigner, asserterOpts, signerCfg,
	)

	return
}

func setupL1WithRollupAddresses(
	t *gotesting.T,
	ctx context.Context,
	rollupStackConf setup.RollupStackConfig,
	useExternalSigner bool,
	nodeConfig *arbnode.Config,
	chainConfig *params.ChainConfig,
	enableCustomDA bool,
) (
	l1info info, l1backend *eth.Ethereum, l1client *ethclient.Client, l1stack *node.Node,
	addresses *chaininfo.RollupAddresses, stakeTokenAddr common.Address, asserterOpts *bind.TransactOpts,
	signerCfg *dataposter.ExternalSignerCfg,
) {
	var srv *externalsignertest.SignerServer
	if useExternalSigner {
		srv = externalsignertest.NewServer(t)
		go func() {
			if err := srv.Start(); err != nil {
				log.Error("Failed to start external signer server:", err)
				return
			}
		}()
	}

	if nodeConfig == nil {
		nodeConfig = arbnode.ConfigDefaultL1Test()
	}
	nodeConfig.ParentChainReader.OldHeaderTimeout = time.Minute * 10
	if chainConfig == nil {
		chainConfig = chaininfo.ArbitrumDevTestChainConfig()
	}
	nodeConfig.BatchPoster.DataPoster.MaxMempoolTransactions = 18
	withoutClientWrapper := false
	l1info, l1client, l1backend, l1stack, _, _ = createTestL1BlockChain(t, nil, withoutClientWrapper, testhelpers.CreateStackConfigForTest(""))

	var err error
	if useExternalSigner {
		signerCfg, err = dataposter.ExternalSignerTestCfg(srv.Address, srv.URL())
		if err != nil {
			t.Fatalf("Error getting external signer config: %v", err)
		}
		asserterOpts, err = dataposter.ExternalSignerTxOpts(ctx, signerCfg)
		Require(t, err)
	} else {
		l1info.GenerateAccount("Asserter")
		tmpOpts := l1info.GetDefaultTransactOpts("Asserter", ctx)
		asserterOpts = &tmpOpts
	}
	l1info.GenerateAccount("EvilAsserter")

	startingBal := big.NewInt(params.Ether)
	startingBal.Mul(startingBal, big.NewInt(100))

	SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
		l1info.PrepareTx("Faucet", "RollupOwner", 30000, startingBal, nil),
		l1info.PrepareTx("Faucet", "Sequencer", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTxTo("Faucet", &asserterOpts.From, 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "EvilAsserter", 30000, big.NewInt(9223372036854775807), nil),
	})

	l1TransactionOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	stakeToken, tx, tokenBindings, err := mocksgen.DeployTestWETH9(
		&l1TransactionOpts,
		l1client,
		"Weth",
		"WETH",
	)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)
	stakeTokenAddr = stakeToken
	value, ok := new(big.Int).SetString("10000", 10)
	if !ok {
		t.Fatal(t, "could not set value")
	}
	l1TransactionOpts.Value = value
	tx, err = tokenBindings.Deposit(&l1TransactionOpts)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)
	l1TransactionOpts.Value = nil

	addresses = deployContractsOnly(t, ctx, l1info, l1client, chainConfig.ChainID, rollupStackConf, stakeToken, asserterOpts, enableCustomDA)
	l1info.SetContract("Bridge", addresses.Bridge)
	l1info.SetContract("SequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("Inbox", addresses.Inbox)
	l1info.SetContract("Rollup", addresses.Rollup)
	l1info.SetContract("UpgradeExecutor", addresses.UpgradeExecutor)

	return l1info, l1backend, l1client, l1stack, addresses, stakeTokenAddr, asserterOpts, signerCfg
}

func createL2NodeWithRollupAddresses(
	t *gotesting.T,
	ctx context.Context,
	isSequencer bool,
	nodeConfig *arbnode.Config,
	chainConfig *params.ChainConfig,
	l2infoIn info,
	l1info info,
	l1client *ethclient.Client,
	addresses *chaininfo.RollupAddresses,
	useExternalSigner bool,
	asserterOpts *bind.TransactOpts,
	signerCfg *dataposter.ExternalSignerCfg,
) (
	l2info info, currentNode *arbnode.Node, execNode *gethexec.ExecutionNode, l2client *ethclient.Client, l2stack *node.Node,
	assertionChain *sol.AssertionChain, l2blockchain *core.BlockChain,
) {
	if nodeConfig == nil {
		nodeConfig = arbnode.ConfigDefaultL1Test()
	}
	fatalErrChan := make(chan error, 10)

	execConfig := ExecConfigDefaultNonSequencerTest(t, rawdb.HashScheme)

	Require(t, execConfig.Validate())
	stackConfig := testhelpers.CreateStackConfigForTest("")
	stackConfig.DBEngine = rawdb.DBPebble
	initMessage, err := nitroinit.GetConsensusParsedInitMsg(ctx, true, l2infoIn.Signer.ChainID(), l1client, addresses, chainConfig)
	Require(t, err)

	var l2executionDB ethdb.Database
	var l2consensusDB ethdb.Database
	l2info, l2stack, l2executionDB, l2consensusDB, l2blockchain = createNonL1BlockChainWithStackConfig(t, l2infoIn, "", chainConfig, nil, initMessage, stackConfig, execConfig, false)
	var sequencerTxOptsPtr *bind.TransactOpts
	var dataSigner signature.DataSignerFunc
	if isSequencer {
		sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)
		sequencerTxOptsPtr = &sequencerTxOpts
		dataSigner = signature.DataSignerFromPrivateKey(l1info.GetInfoWithPrivKey("Sequencer").PrivateKey)
	}

	if !isSequencer {
		nodeConfig.BatchPoster.Enable = false
		nodeConfig.DelayedSequencer.Enable = false
	}

	AddValNodeIfNeeded(t, ctx, nodeConfig, true, "", "")

	parentChainId, err := l1client.ChainID(ctx)
	Require(t, err)
	execNode, err = gethexec.CreateExecutionNode(ctx, l2stack, l2executionDB, l2blockchain, l1client, NewCommonConfigFetcher(execConfig), parentChainId, 0)
	Require(t, err)

	Require(t, err)
	locator, err := server_common.NewMachineLocator("")
	Require(t, err)
	currentNode, err = arbnode.CreateConsensusNode(
		ctx, l2stack, execNode, l2consensusDB, NewCommonConfigFetcher(nodeConfig), l2blockchain.Config(), l1client,
		addresses, sequencerTxOptsPtr, sequencerTxOptsPtr, dataSigner, fatalErrChan, parentChainId,
		nil, // Blob reader.
		locator.LatestWasmModuleRoot(),
	)
	Require(t, err)

	l2client = ClientForStack(t, l2stack)

	StartWatchChanErr(t, ctx, fatalErrChan, currentNode)

	// Get challenge manager address from rollup contract
	rollupUser, err := rollupgen.NewRollupUserLogic(addresses.Rollup, l1client)
	Require(t, err)
	chalManagerAddr, err := rollupUser.ChallengeManager(&bind.CallOpts{})
	Require(t, err)

	var dpOpts *bind.TransactOpts
	if useExternalSigner {
		nodeConfig.Staker.DataPoster.ExternalSigner = *signerCfg
	} else {
		dpOpts = asserterOpts
	}
	dp, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(l2consensusDB, storage.StakerPrefix),
		currentNode.L1Reader,
		dpOpts,
		NewCommonConfigFetcher(nodeConfig),
		currentNode.SyncMonitor,
		parentChainId,
	)
	Require(t, err)
	assertionChainBindings, err := sol.NewAssertionChain(
		ctx,
		addresses.Rollup,
		chalManagerAddr,
		dp.Auth(),
		l1client,
		bold.NewDataPosterTransactor(dp),
		sol.WithRpcHeadBlockNumber(rpc.LatestBlockNumber),
	)
	Require(t, err)
	assertionChain = assertionChainBindings

	return l2info, currentNode, execNode, l2client, l2stack, assertionChain, l2blockchain
}

func deployContractsOnly(
	t *gotesting.T,
	ctx context.Context,
	l1info info,
	backend *ethclient.Client,
	chainId *big.Int,
	rollupStackConf setup.RollupStackConfig,
	stakeToken common.Address,
	asserterOpts *bind.TransactOpts,
	enableCustomDA bool,
) *chaininfo.RollupAddresses {
	l1TransactionOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	locator, err := server_common.NewMachineLocator("")
	Require(t, err)
	wasmModuleRoot := locator.LatestWasmModuleRoot()

	loserStakeEscrow := l1TransactionOpts.From
	genesisExecutionState := rollupgen.AssertionState{
		GlobalState:    rollupgen.GlobalState{},
		MachineStatus:  1,
		EndHistoryRoot: [32]byte{},
	}
	genesisInboxCount := big.NewInt(0)
	anyTrustFastConfirmer := common.Address{}
	miniStakeValues := []*big.Int{big.NewInt(5), big.NewInt(4), big.NewInt(3), big.NewInt(2), big.NewInt(1)}
	cfg := challenge_testing.GenerateRollupConfig(
		false,
		wasmModuleRoot,
		l1TransactionOpts.From,
		chainId,
		loserStakeEscrow,
		miniStakeValues,
		stakeToken,
		genesisExecutionState,
		genesisInboxCount,
		anyTrustFastConfirmer,
		challenge_testing.WithLayerZeroHeights(&protocol.LayerZeroHeights{
			BlockChallengeHeight:     protocol.Height(blockChallengeLeafHeight),
			BigStepChallengeHeight:   protocol.Height(bigStepChallengeLeafHeight),
			SmallStepChallengeHeight: protocol.Height(smallStepChallengeLeafHeight),
		}),
		challenge_testing.WithNumBigStepLevels(uint8(3)),       // TODO: Hardcoded.
		challenge_testing.WithConfirmPeriodBlocks(uint64(120)), // TODO: Hardcoded.
	)
	config, err := json.Marshal(chaininfo.ArbitrumDevTestChainConfig())
	Require(t, err)
	cfg.ChainConfig = string(config)

	var addresses *setup.RollupAddresses

	if enableCustomDA {
		t.Log("Deploying ReferenceDAProofValidator and custom OSP for custom DA")

		// Deploy ReferenceDAProofValidator with trusted signers
		// Create a dedicated DA signer account
		l1info.GenerateAccount("DASigner")
		trustedSigners := []common.Address{l1info.GetAddress("DASigner")}
		refDAValidatorAddr, tx, _, err := localgen.DeployReferenceDAProofValidator(&l1TransactionOpts, backend, trustedSigners)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, backend, tx)
		Require(t, err)
		t.Logf("Deployed ReferenceDAProofValidator at %s", refDAValidatorAddr.Hex())
		// Store the validator address so it can be accessed by tests
		l1info.SetContract("ReferenceDAProofValidator", refDAValidatorAddr)

		// Deploy custom OSP contracts
		customOspAddr := deployCustomDAOSP(t, ctx, backend, &l1TransactionOpts, refDAValidatorAddr)
		t.Logf("Deployed custom OneStepProofEntry at %s", customOspAddr.Hex())

		// Deploy using the custom OSP
		rollupStackConf.CustomDAOsp = customOspAddr
		addresses, err = setup.DeployFullRollupStack(
			ctx,
			backend,
			&l1TransactionOpts,
			l1info.GetAddress("Sequencer"),
			cfg,
			rollupStackConf,
		)
		Require(t, err)

		t.Log("Successfully deployed with custom OneStepProofEntry for custom DA support")
	} else {
		addresses, err = setup.DeployFullRollupStack(
			ctx,
			backend,
			&l1TransactionOpts,
			l1info.GetAddress("Sequencer"),
			cfg,
			rollupStackConf,
		)
		Require(t, err)
	}

	evilAsserter := l1info.GetDefaultTransactOpts("EvilAsserter", ctx)
	userLogic, err := rollupgen.NewRollupUserLogic(addresses.Rollup, backend)
	Require(t, err)
	chalManagerAddr, err := userLogic.ChallengeManager(&bind.CallOpts{})
	Require(t, err)
	seed, ok := new(big.Int).SetString("1000", 10)
	if !ok {
		t.Fatal("not ok")
	}
	value, ok := new(big.Int).SetString("10000", 10)
	if !ok {
		t.Fatal(t, "could not set value")
	}
	tokenBindings, err := mocksgen.NewTestWETH9(stakeToken, backend)
	Require(t, err)
	tx, err := tokenBindings.TestWETH9Transactor.Transfer(&l1TransactionOpts, asserterOpts.From, seed)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)
	tx, err = tokenBindings.TestWETH9Transactor.Approve(asserterOpts, addresses.Rollup, value)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)
	tx, err = tokenBindings.TestWETH9Transactor.Approve(asserterOpts, chalManagerAddr, value)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)

	tx, err = tokenBindings.TestWETH9Transactor.Transfer(&l1TransactionOpts, evilAsserter.From, seed)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)
	tx, err = tokenBindings.TestWETH9Transactor.Approve(&evilAsserter, addresses.Rollup, value)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)
	tx, err = tokenBindings.TestWETH9Transactor.Approve(&evilAsserter, chalManagerAddr, value)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)

	return &chaininfo.RollupAddresses{
		Bridge:                 addresses.Bridge,
		Inbox:                  addresses.Inbox,
		SequencerInbox:         addresses.SequencerInbox,
		Rollup:                 addresses.Rollup,
		ValidatorUtils:         addresses.ValidatorUtils,
		ValidatorWalletCreator: addresses.ValidatorWalletCreator,
		DeployedAt:             addresses.DeployedAt,
		UpgradeExecutor:        addresses.UpgradeExecutor,
	}
}
