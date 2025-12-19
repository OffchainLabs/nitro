// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// race detection makes things slow and miss timeouts
//go:build !race

package arbtest

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbnode/dataposter/externalsignertest"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/solgen/go/bridge_legacy_gen"
	"github.com/offchainlabs/nitro/solgen/go/contractsgen"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/solgen/go/proxiesgen"
	"github.com/offchainlabs/nitro/solgen/go/rollup_legacy_gen"
	"github.com/offchainlabs/nitro/solgen/go/upgrade_executorgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/legacy"
	"github.com/offchainlabs/nitro/staker/validatorwallet"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func TestFastConfirmationWithdrawal(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	builder, stakerA, cleanupBuilder, cleanupBackgroundTx := setupFastConfirmation(ctx, t)
	defer cleanupBuilder()
	defer cleanupBackgroundTx()

	// Withdraw ETH from L2 to L1
	arbSys, err := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L2.Client)
	Require(t, err)
	authL2 := builder.L2Info.GetDefaultTransactOpts("User", ctx)
	intialL2Balance := builder.L2.GetBalance(t, authL2.From)
	withdrawAmount := big.NewInt(1000)
	authL2.Value = withdrawAmount
	builder.L1Info.GenerateAccount("Receiver")
	receiver := builder.L1Info.GetAddress("Receiver")
	tx, err := arbSys.WithdrawEth(&authL2, receiver)
	Require(t, err, "ArbSys failed")

	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	if len(receipt.Logs) == 0 {
		Fatal(t, "Tx didn't emit any logs")
	}
	gasUsedInL2 := new(big.Int).Mul(new(big.Int).SetUint64(receipt.GasUsed), receipt.EffectiveGasPrice)
	l2FundsSpent := new(big.Int).Add(withdrawAmount, gasUsedInL2)

	// Wait for staker to confirm the withdrawal
	time.Sleep(time.Second)
	tx, err = stakerA.Act(ctx)
	Require(t, err)
	if tx != nil {
		_, err = builder.L1.EnsureTxSucceeded(tx)
		Require(t, err)
	}

	arbSysAbi, err := precompilesgen.ArbSysMetaData.GetAbi()
	Require(t, err, "failed to get abi")
	withdrawTopic := arbSysAbi.Events["L2ToL1Tx"].ID
	authL1 := builder.L1Info.GetDefaultTransactOpts("User", ctx)
	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, builder.L2.Client)
	Require(t, err)
	merkleState, err := arbSys.SendMerkleTreeState(&bind.CallOpts{})
	Require(t, err, "could not get merkle root")
	bridgeBinding, err := bridge_legacy_gen.NewBridge(builder.L1Info.GetAddress("Bridge"), builder.L1.Client)
	Require(t, err)
	outboxAddress, err := bridgeBinding.AllowedOutboxList(&bind.CallOpts{}, big.NewInt(0))
	Require(t, err)
	outboxBinding, err := bridge_legacy_gen.NewOutbox(outboxAddress, builder.L1.Client)
	Require(t, err)
	ouboxAbi, err := bridge_legacy_gen.AbsOutboxMetaData.GetAbi()
	Require(t, err, "failed to get abi")
	outBoxTransactionExecutedTopic := ouboxAbi.Events["OutBoxTransactionExecuted"].ID
	// Check logs for withdraw event
	foundWithdraw := false
	for _, log := range receipt.Logs {
		if log.Topics[0] == withdrawTopic {
			foundWithdraw = true
			parsedLog, err := arbSys.ParseL2ToL1Tx(*log)
			Require(t, err, "Failed to parse log")

			// Check NodeInterface.sol produces equivalent proofs
			outboxProof, err := nodeInterface.ConstructOutboxProof(
				&bind.CallOpts{}, merkleState.Size.Uint64(), parsedLog.Position.Uint64(),
			)
			Require(t, err)
			// Execute the transaction on L1
			execTx, err := outboxBinding.ExecuteTransaction(&authL1, outboxProof.Proof, parsedLog.Position, parsedLog.Caller, parsedLog.Destination, parsedLog.ArbBlockNum, parsedLog.EthBlockNum, parsedLog.Timestamp, parsedLog.Callvalue, parsedLog.Data)
			Require(t, err)
			execReceipt, err := builder.L1.EnsureTxSucceeded(execTx)
			Require(t, err)
			if len(execReceipt.Logs) == 0 {
				Fatal(t, "Tx didn't emit any logs")
			}
			foundExec := false
			for _, execLog := range execReceipt.Logs {
				if execLog.Topics[0] == outBoxTransactionExecutedTopic {
					foundExec = true
					break
				}
			}
			if !foundExec {
				Fatal(t, "Execution event not found in logs")
			}
			break
		}
	}
	if !foundWithdraw {
		Fatal(t, "Withdraw event not found in logs")
	}
	if builder.L1.GetBalance(t, receiver).Cmp(withdrawAmount) != 0 {
		Fatal(t, "Withdrawal failed")
	}
	if builder.L2.GetBalance(t, authL2.From).Cmp(new(big.Int).Sub(intialL2Balance, l2FundsSpent)) != 0 {
		Fatal(t, "Withdrawal failed")
	}
}
func TestFastConfirmation(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	builder, stakerA, cleanupBuilder, cleanupBackgroundTx := setupFastConfirmation(ctx, t)
	defer cleanupBuilder()
	defer cleanupBackgroundTx()

	rollup, err := rollup_legacy_gen.NewRollupAdminLogic(builder.L2.ConsensusNode.DeployInfo.Rollup, builder.L1.Client)
	Require(t, err)
	latestConfirmBeforeAct, err := rollup.LatestConfirmed(&bind.CallOpts{})
	Require(t, err)
	tx, err := stakerA.Act(ctx)
	Require(t, err)
	if tx != nil {
		_, err = builder.L1.EnsureTxSucceeded(tx)
		Require(t, err)
	}
	latestConfirmAfterAct, err := rollup.LatestConfirmed(&bind.CallOpts{})
	Require(t, err)
	if latestConfirmAfterAct <= latestConfirmBeforeAct {
		Fatal(t, fmt.Sprintf("staker A didn't advance the latest confirmed node: want > %d, got: %d", latestConfirmBeforeAct, latestConfirmAfterAct))
	}
}

func setupFastConfirmation(ctx context.Context, t *testing.T) (*NodeBuilder, *legacystaker.Staker, func(), func()) {
	srv := externalsignertest.NewServer(t)
	go func() {
		if err := srv.Start(); err != nil {
			log.Error("Failed to start external signer server:", err)
			return
		}
	}()
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithPreBoldDeployment().WithProdConfirmPeriodBlocks().DontParalellise()
	builder.L2Info = NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(builder.chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	builder.L2Info.GenerateGenesisAccount("User", new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(9)))

	builder.nodeConfig.BatchPoster.MaxDelay = -1000 * time.Hour
	cleanupBuilder := builder.Build(t)

	addNewBatchPoster(ctx, t, builder, srv.Address)

	builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
		builder.L1Info.PrepareTxTo("Faucet", &srv.Address, 30000, big.NewInt(1).Mul(big.NewInt(1e18), big.NewInt(1e18)), nil)})

	l2node := builder.L2.ConsensusNode
	execNode := builder.L2.ExecNode

	builder.execConfig.Sequencer.Enable = false

	builder.BridgeBalance(t, "Faucet", big.NewInt(1).Mul(big.NewInt(params.Ether), big.NewInt(10000)))

	deployAuth := builder.L1Info.GetDefaultTransactOpts("RollupOwner", ctx)

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	builder.L1.TransferBalance(t, "Faucet", "Validator", balance, builder.L1Info)
	l1auth := builder.L1Info.GetDefaultTransactOpts("Validator", ctx)

	upgradeExecutor, err := upgrade_executorgen.NewUpgradeExecutor(l2node.DeployInfo.UpgradeExecutor, builder.L1.Client)
	Require(t, err, "unable to bind upgrade executor")
	rollupABI, err := abi.JSON(strings.NewReader(rollup_legacy_gen.RollupAdminLogicABI))
	Require(t, err, "unable to parse rollup ABI")

	setMinAssertPeriodCalldata, err := rollupABI.Pack("setMinimumAssertionPeriod", big.NewInt(1))
	Require(t, err, "unable to generate setMinimumAssertionPeriod calldata")
	tx, err := upgradeExecutor.ExecuteCall(&deployAuth, l2node.DeployInfo.Rollup, setMinAssertPeriodCalldata)
	Require(t, err, "unable to set minimum assertion period")
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	valConfig := legacystaker.TestL1ValidatorConfig
	valConfig.EnableFastConfirmation = true
	parentChainID, err := builder.L1.Client.ChainID(ctx)
	if err != nil {
		t.Fatalf("Failed to get parent chain id: %v", err)
	}
	dp, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(l2node.ArbDB, storage.StakerPrefix),
		l2node.L1Reader,
		&l1auth, NewCommonConfigFetcher(arbnode.ConfigDefaultL1NonSequencerTest()),
		nil,
		parentChainID,
	)
	if err != nil {
		t.Fatalf("Error creating validator dataposter: %v", err)
	}
	valWallet, err := validatorwallet.NewContract(dp, nil, l2node.DeployInfo.ValidatorWalletCreator, l2node.L1Reader, &l1auth, 0, func(common.Address) {}, func() uint64 { return valConfig.ExtraGas })
	Require(t, err)
	valConfig.Strategy = "MakeNodes"

	valWalletAddrPtr, err := validatorwallet.GetValidatorWalletContract(ctx, l2node.DeployInfo.ValidatorWalletCreator, 0, l2node.L1Reader, true, valWallet.DataPoster(), valWallet.GetExtraGas())
	Require(t, err)
	valWalletAddr := *valWalletAddrPtr
	valWalletAddrCheck, err := validatorwallet.GetValidatorWalletContract(ctx, l2node.DeployInfo.ValidatorWalletCreator, 0, l2node.L1Reader, true, valWallet.DataPoster(), valWallet.GetExtraGas())
	Require(t, err)
	if valWalletAddr == *valWalletAddrCheck {
		Require(t, err, "didn't cache validator wallet address", valWalletAddr.String(), "vs", valWalletAddrCheck.String())
	}

	setValidatorCalldata, err := rollupABI.Pack("setValidator", []common.Address{valWalletAddr, srv.Address}, []bool{true, true})
	Require(t, err, "unable to generate setValidator calldata")
	tx, err = upgradeExecutor.ExecuteCall(&deployAuth, l2node.DeployInfo.Rollup, setValidatorCalldata)
	Require(t, err, "unable to set validators")
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	setAnyTrustFastConfirmerCalldata, err := rollupABI.Pack("setAnyTrustFastConfirmer", valWalletAddr)
	Require(t, err, "unable to generate setAnyTrustFastConfirmer calldata")
	tx, err = upgradeExecutor.ExecuteCall(&deployAuth, l2node.DeployInfo.Rollup, setAnyTrustFastConfirmerCalldata)
	Require(t, err, "unable to set anytrust fast confirmer")
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	blockValidatorConfig := staker.TestBlockValidatorConfig

	locator, err := server_common.NewMachineLocator(valnode.TestValidationConfig.Wasm.RootPath)
	Require(t, err)
	stateless, err := staker.NewStatelessBlockValidator(
		l2node.InboxReader,
		l2node.InboxTracker,
		l2node.TxStreamer,
		execNode,
		l2node.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
		locator.LatestWasmModuleRoot(),
	)
	Require(t, err)
	err = stateless.Start(ctx)
	Require(t, err)
	err = valWallet.Initialize(ctx)
	Require(t, err)
	stakerA, err := legacystaker.NewStaker(
		l2node.L1Reader,
		valWallet,
		bind.CallOpts{},
		func() *legacystaker.L1ValidatorConfig { return &valConfig },
		nil,
		stateless,
		nil,
		nil,
		l2node.DeployInfo.ValidatorUtils,
		l2node.DeployInfo.Rollup,
		l2node.InboxTracker,
		l2node.TxStreamer,
		l2node.InboxReader,
		nil,
	)
	Require(t, err)
	err = stakerA.Initialize(ctx)
	Require(t, err)
	cfg := arbnode.ConfigDefaultL1NonSequencerTest()
	signerCfg, err := dataposter.ExternalSignerTestCfg(srv.Address, srv.URL())
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
	cleanupBackgroundTx := func() {
		cancelBackgroundTxs()
		<-backgroundTxsShutdownChan
	}
	go (func() {
		defer close(backgroundTxsShutdownChan)
		err := makeBackgroundTxs(backgroundTxsCtx, builder)
		if !errors.Is(err, context.Canceled) {
			log.Warn("error making background txs", "err", err)
		}
	})()
	return builder, stakerA, cleanupBuilder, cleanupBackgroundTx
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
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithPreBoldDeployment().WithProdConfirmPeriodBlocks()
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

	rollup, err := rollup_legacy_gen.NewRollupAdminLogic(l2nodeA.DeployInfo.Rollup, builder.L1.Client)
	Require(t, err)

	upgradeExecutor, err := upgrade_executorgen.NewUpgradeExecutor(l2nodeA.DeployInfo.UpgradeExecutor, builder.L1.Client)
	Require(t, err, "unable to bind upgrade executor")
	rollupABI, err := abi.JSON(strings.NewReader(rollup_legacy_gen.RollupAdminLogicABI))
	Require(t, err, "unable to parse rollup ABI")

	setMinAssertPeriodCalldata, err := rollupABI.Pack("setMinimumAssertionPeriod", big.NewInt(1))
	Require(t, err, "unable to generate setMinimumAssertionPeriod calldata")
	tx, err := upgradeExecutor.ExecuteCall(&deployAuth, l2nodeA.DeployInfo.Rollup, setMinAssertPeriodCalldata)
	Require(t, err, "unable to set minimum assertion period")
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	valConfigA := legacystaker.TestL1ValidatorConfig
	valConfigA.EnableFastConfirmation = true

	parentChainID, err := builder.L1.Client.ChainID(ctx)
	if err != nil {
		t.Fatalf("Failed to get parent chain id: %v", err)
	}
	dpA, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(l2nodeB.ArbDB, storage.StakerPrefix),
		l2nodeA.L1Reader,
		&l1authA, NewCommonConfigFetcher(arbnode.ConfigDefaultL1NonSequencerTest()),
		nil,
		parentChainID,
	)
	if err != nil {
		t.Fatalf("Error creating validator dataposter: %v", err)
	}
	valWalletA, err := validatorwallet.NewContract(dpA, nil, l2nodeA.DeployInfo.ValidatorWalletCreator, l2nodeA.L1Reader, &l1authA, 0, func(common.Address) {}, func() uint64 { return valConfigA.ExtraGas })
	Require(t, err)
	valConfigA.Strategy = "MakeNodes"

	valWalletAddrAPtr, err := validatorwallet.GetValidatorWalletContract(ctx, l2nodeA.DeployInfo.ValidatorWalletCreator, 0, l2nodeA.L1Reader, true, valWalletA.DataPoster(), valWalletA.GetExtraGas())
	Require(t, err)
	valWalletAddrA := *valWalletAddrAPtr
	valWalletAddrCheck, err := validatorwallet.GetValidatorWalletContract(ctx, l2nodeA.DeployInfo.ValidatorWalletCreator, 0, l2nodeA.L1Reader, true, valWalletA.DataPoster(), valWalletA.GetExtraGas())
	Require(t, err)
	if valWalletAddrA == *valWalletAddrCheck {
		Require(t, err, "didn't cache validator wallet address", valWalletAddrA.String(), "vs", valWalletAddrCheck.String())
	}

	safeAddress := deploySafe(t, builder.L1, builder.L1.Client, deployAuth, []common.Address{valWalletAddrA, srv.Address})
	setValidatorCalldata, err := rollupABI.Pack("setValidator", []common.Address{valWalletAddrA, l1authB.From, srv.Address, safeAddress}, []bool{true, true, true, true})
	Require(t, err, "unable to generate setValidator calldata")
	tx, err = upgradeExecutor.ExecuteCall(&deployAuth, l2nodeA.DeployInfo.Rollup, setValidatorCalldata)
	Require(t, err, "unable to set validators")
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	setAnyTrustFastConfirmerCalldata, err := rollupABI.Pack("setAnyTrustFastConfirmer", safeAddress)
	Require(t, err, "unable to generate setAnyTrustFastConfirmer calldata")
	tx, err = upgradeExecutor.ExecuteCall(&deployAuth, l2nodeA.DeployInfo.Rollup, setAnyTrustFastConfirmerCalldata)
	Require(t, err, "unable to set anytrust fast confirmer")
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	blockValidatorConfig := staker.TestBlockValidatorConfig

	locator, err := server_common.NewMachineLocator(valnode.TestValidationConfig.Wasm.RootPath)
	Require(t, err)
	statelessA, err := staker.NewStatelessBlockValidator(
		l2nodeA.InboxReader,
		l2nodeA.InboxTracker,
		l2nodeA.TxStreamer,
		execNodeA,
		l2nodeA.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
		locator.LatestWasmModuleRoot(),
	)
	Require(t, err)
	err = statelessA.Start(ctx)
	Require(t, err)
	err = valWalletA.Initialize(ctx)
	Require(t, err)
	stakerA, err := legacystaker.NewStaker(
		l2nodeA.L1Reader,
		valWalletA,
		bind.CallOpts{},
		func() *legacystaker.L1ValidatorConfig { return &valConfigA },
		nil,
		statelessA,
		nil,
		nil,
		l2nodeA.DeployInfo.ValidatorUtils,
		l2nodeA.DeployInfo.Rollup,
		l2nodeA.InboxTracker,
		l2nodeA.TxStreamer,
		l2nodeA.InboxReader,
		nil,
	)
	Require(t, err)
	err = stakerA.Initialize(ctx)
	Require(t, err)
	cfg := arbnode.ConfigDefaultL1NonSequencerTest()
	signerCfg, err := dataposter.ExternalSignerTestCfg(srv.Address, srv.URL())
	if err != nil {
		t.Fatalf("Error getting external signer config: %v", err)
	}
	cfg.Staker.DataPoster.ExternalSigner = *signerCfg
	dpB, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(l2nodeB.ArbDB, storage.StakerPrefix),
		l2nodeB.L1Reader,
		&l1authB, NewCommonConfigFetcher(cfg),
		nil,
		parentChainID,
	)
	if err != nil {
		t.Fatalf("Error creating validator dataposter: %v", err)
	}
	valWalletB, err := validatorwallet.NewEOA(dpB, l2nodeB.L1Reader.Client(), func() uint64 { return 0 })
	Require(t, err)
	valConfigB := legacystaker.TestL1ValidatorConfig
	valConfigB.EnableFastConfirmation = true
	valConfigB.Strategy = "watchtower"
	statelessB, err := staker.NewStatelessBlockValidator(
		l2nodeB.InboxReader,
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		execNodeB,
		l2nodeB.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
		locator.LatestWasmModuleRoot(),
	)
	Require(t, err)
	err = statelessB.Start(ctx)
	Require(t, err)
	err = valWalletB.Initialize(ctx)
	Require(t, err)
	stakerB, err := legacystaker.NewStaker(
		l2nodeB.L1Reader,
		valWalletB,
		bind.CallOpts{},
		func() *legacystaker.L1ValidatorConfig { return &valConfigB },
		nil,
		statelessB,
		nil,
		nil,
		l2nodeB.DeployInfo.ValidatorUtils,
		l2nodeB.DeployInfo.Rollup,
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		l2nodeB.InboxReader,
		nil,
	)
	Require(t, err)
	err = stakerB.Initialize(ctx)
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
