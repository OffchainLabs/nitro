// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build challengetest && !race

package arbtest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
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
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	protocol "github.com/offchainlabs/bold/chain-abstraction"
	solimpl "github.com/offchainlabs/bold/chain-abstraction/sol-implementation"
	challengemanager "github.com/offchainlabs/bold/challenge-manager"
	modes "github.com/offchainlabs/bold/challenge-manager/types"
	l2stateprovider "github.com/offchainlabs/bold/layer2-state-provider"
	"github.com/offchainlabs/bold/solgen/go/bridgegen"
	"github.com/offchainlabs/bold/solgen/go/challengeV2gen"
	"github.com/offchainlabs/bold/solgen/go/mocksgen"
	"github.com/offchainlabs/bold/solgen/go/rollupgen"
	challengetesting "github.com/offchainlabs/bold/testing"
	"github.com/offchainlabs/bold/testing/setup"
	butil "github.com/offchainlabs/bold/util"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/bold"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/validator/server_arb"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func TestChallengeProtocolBOLDReadInboxChallenge(t *testing.T) {
	testChallengeProtocolBOLD(t)
}

func TestChallengeProtocolBOLDStartStepChallenge(t *testing.T) {
	opts := []server_arb.SpawnerOption{
		server_arb.WithWrapper(func(inner server_arb.MachineInterface) server_arb.MachineInterface {
			// This wrapper is applied after the BOLD wrapper, so step 0 is the finished machine.
			// Modifying its hash results in invalid inclusion proofs for the evil validator,
			// so we start modifying hashes at step 1 (the first machine step in the running state).
			return NewIncorrectIntermediateMachine(inner, 1)
		}),
	}
	testChallengeProtocolBOLD(t, opts...)
}

func testChallengeProtocolBOLD(t *testing.T, spawnerOpts ...server_arb.SpawnerOption) {
	goodDir, err := os.MkdirTemp("", "good_*")
	Require(t, err)
	evilDir, err := os.MkdirTemp("", "evil_*")
	Require(t, err)
	t.Cleanup(func() {
		Require(t, os.RemoveAll(goodDir))
		Require(t, os.RemoveAll(evilDir))
	})
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
		UseMockBridge:          false,
		UseMockOneStepProver:   false,
		MinimumAssertionPeriod: 0,
	}

	_, l2nodeA, _, _, l1info, _, l1client, l1stack, assertionChain, stakeTokenAddr := createTestNodeOnL1ForBoldProtocol(
		t,
		ctx,
		true,
		nil,
		l2chainConfig,
		nil,
		sconf,
		l2info,
	)
	defer requireClose(t, l1stack)
	defer l2nodeA.StopAndWait()

	// Make sure we shut down test functionality before the rest of the node
	ctx, cancelCtx = context.WithCancel(ctx)
	defer cancelCtx()

	go keepChainMoving(t, ctx, l1info, l1client)

	l2nodeConfig := arbnode.ConfigDefaultL1Test()
	_, l2nodeB, _ := create2ndNodeWithConfigForBoldProtocol(
		t,
		ctx,
		l2nodeA,
		l1stack,
		l1info,
		&l2info.ArbInitData,
		l2nodeConfig,
		nil,
		sconf,
		stakeTokenAddr,
	)
	defer l2nodeB.StopAndWait()

	genesisA, err := l2nodeA.Execution.ResultAtPos(0)
	Require(t, err)
	genesisB, err := l2nodeB.Execution.ResultAtPos(0)
	Require(t, err)
	if genesisA.BlockHash != genesisB.BlockHash {
		Fatal(t, "genesis blocks mismatch between nodes")
	}

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	TransferBalance(t, "Faucet", "Asserter", balance, l1info, l1client, ctx)
	TransferBalance(t, "Faucet", "EvilAsserter", balance, l1info, l1client, ctx)

	valCfg := valnode.TestValidationConfig
	valCfg.UseJit = false
	_, valStack := createTestValidationNode(t, ctx, &valCfg)
	blockValidatorConfig := staker.TestBlockValidatorConfig

	statelessA, err := staker.NewStatelessBlockValidator(
		l2nodeA.InboxReader,
		l2nodeA.InboxTracker,
		l2nodeA.TxStreamer,
		l2nodeA.Execution,
		l2nodeA.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	err = statelessA.Start(ctx)
	Require(t, err)
	_, valStackB := createTestValidationNode(t, ctx, &valCfg, spawnerOpts...)

	statelessB, err := staker.NewStatelessBlockValidator(
		l2nodeB.InboxReader,
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		l2nodeB.Execution,
		l2nodeB.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStackB,
	)
	Require(t, err)
	err = statelessB.Start(ctx)
	Require(t, err)

	blockValidatorA, err := staker.NewBlockValidator(
		statelessA,
		l2nodeA.InboxTracker,
		l2nodeA.TxStreamer,
		StaticFetcherFrom(t, &blockValidatorConfig),
		nil,
	)
	Require(t, err)
	Require(t, blockValidatorA.Initialize(ctx))
	Require(t, blockValidatorA.Start(ctx))

	blockValidatorB, err := staker.NewBlockValidator(
		statelessB,
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		StaticFetcherFrom(t, &blockValidatorConfig),
		nil,
	)
	Require(t, err)
	Require(t, blockValidatorB.Initialize(ctx))
	Require(t, blockValidatorB.Start(ctx))

	stateManager, err := bold.NewBOLDStateProvider(
		blockValidatorA,
		statelessA,
		l2stateprovider.Height(blockChallengeLeafHeight),
		&bold.StateProviderConfig{
			ValidatorName:          "good",
			MachineLeavesCachePath: goodDir,
			CheckBatchFinality:     false,
		},
		goodDir,
	)
	Require(t, err)

	stateManagerB, err := bold.NewBOLDStateProvider(
		blockValidatorB,
		statelessB,
		l2stateprovider.Height(blockChallengeLeafHeight),
		&bold.StateProviderConfig{
			ValidatorName:          "evil",
			MachineLeavesCachePath: evilDir,
			CheckBatchFinality:     false,
		},
		evilDir,
	)
	Require(t, err)

	Require(t, l2nodeA.Start(ctx))
	Require(t, l2nodeB.Start(ctx))

	chalManagerAddr := assertionChain.SpecChallengeManager()
	evilOpts := l1info.GetDefaultTransactOpts("EvilAsserter", ctx)
	l1ChainId, err := l1client.ChainID(ctx)
	Require(t, err)
	dp, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(l2nodeB.ArbDB, storage.StakerPrefix),
		l2nodeB.L1Reader,
		&evilOpts,
		NewFetcherFromConfig(l2nodeConfig),
		l2nodeB.SyncMonitor,
		l1ChainId,
	)
	Require(t, err)
	chainB, err := solimpl.NewAssertionChain(
		ctx,
		assertionChain.RollupAddress(),
		chalManagerAddr.Address(),
		&evilOpts,
		butil.NewBackendWrapper(l1client, rpc.LatestBlockNumber),
		bold.NewDataPosterTransactor(dp),
		solimpl.WithRpcHeadBlockNumber(rpc.LatestBlockNumber),
	)
	Require(t, err)

	l2info.GenerateAccount("Destination")
	sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)

	honestSeqInbox := l1info.GetAddress("SequencerInbox")
	evilSeqInbox := l1info.GetAddress("EvilSequencerInbox")
	honestSeqInboxBinding, err := bridgegen.NewSequencerInbox(honestSeqInbox, l1client)
	Require(t, err)
	evilSeqInboxBinding, err := bridgegen.NewSequencerInbox(evilSeqInbox, l1client)
	Require(t, err)

	// Post batches to the honest and evil sequencer inbox that are internally equal.
	// This means the honest and evil sequencer inboxes will agree with all messages in the batch.
	seqInboxABI, err := abi.JSON(strings.NewReader(bridgegen.SequencerInboxABI))
	Require(t, err)

	honestUpgradeExec, err := mocksgen.NewUpgradeExecutorMock(l1info.GetAddress("UpgradeExecutor"), l1client)
	Require(t, err)
	data, err := seqInboxABI.Pack(
		"setIsBatchPoster",
		sequencerTxOpts.From,
		true,
	)
	Require(t, err)
	honestRollupOwnerOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	_, err = honestUpgradeExec.ExecuteCall(&honestRollupOwnerOpts, honestSeqInbox, data)
	Require(t, err)

	evilUpgradeExec, err := mocksgen.NewUpgradeExecutorMock(l1info.GetAddress("EvilUpgradeExecutor"), l1client)
	Require(t, err)
	data, err = seqInboxABI.Pack(
		"setIsBatchPoster",
		sequencerTxOpts.From,
		true,
	)
	Require(t, err)
	evilRollupOwnerOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	_, err = evilUpgradeExec.ExecuteCall(&evilRollupOwnerOpts, evilSeqInbox, data)
	Require(t, err)

	totalMessagesPosted := int64(0)
	numMessagesPerBatch := int64(5)
	divergeAt := int64(-1)
	makeBoldBatch(t, l2nodeA, l2info, l1client, &sequencerTxOpts, honestSeqInboxBinding, honestSeqInbox, numMessagesPerBatch, divergeAt)
	l2info.Accounts["Owner"].Nonce.Store(0)
	makeBoldBatch(t, l2nodeB, l2info, l1client, &sequencerTxOpts, evilSeqInboxBinding, evilSeqInbox, numMessagesPerBatch, divergeAt)
	totalMessagesPosted += numMessagesPerBatch

	// Next, we post another batch, this time containing more messages.
	// We diverge at message index 5 within the evil node's batch.
	l2info.Accounts["Owner"].Nonce.Store(5)
	numMessagesPerBatch = int64(10)
	makeBoldBatch(t, l2nodeA, l2info, l1client, &sequencerTxOpts, honestSeqInboxBinding, honestSeqInbox, numMessagesPerBatch, divergeAt)
	l2info.Accounts["Owner"].Nonce.Store(5)
	divergeAt = int64(5)
	makeBoldBatch(t, l2nodeB, l2info, l1client, &sequencerTxOpts, evilSeqInboxBinding, evilSeqInbox, numMessagesPerBatch, divergeAt)
	totalMessagesPosted += numMessagesPerBatch

	bcA, err := l2nodeA.InboxTracker.GetBatchCount()
	Require(t, err)
	bcB, err := l2nodeB.InboxTracker.GetBatchCount()
	Require(t, err)
	msgA, err := l2nodeA.InboxTracker.GetBatchMessageCount(bcA - 1)
	Require(t, err)
	msgB, err := l2nodeB.InboxTracker.GetBatchMessageCount(bcB - 1)
	Require(t, err)

	t.Logf("Node A batch count %d, msgs %d", bcA, msgA)
	t.Logf("Node B batch count %d, msgs %d", bcB, msgB)

	// Wait for both nodes' chains to catch up.
	nodeAExec, ok := l2nodeA.Execution.(*gethexec.ExecutionNode)
	if !ok {
		Fatal(t, "not geth execution node")
	}
	nodeBExec, ok := l2nodeB.Execution.(*gethexec.ExecutionNode)
	if !ok {
		Fatal(t, "not geth execution node")
	}
	for {
		nodeALatest := nodeAExec.Backend.APIBackend().CurrentHeader()
		nodeBLatest := nodeBExec.Backend.APIBackend().CurrentHeader()
		isCaughtUp := nodeALatest.Number.Uint64() == uint64(totalMessagesPosted)
		areEqual := nodeALatest.Number.Uint64() == nodeBLatest.Number.Uint64()
		if isCaughtUp && areEqual {
			if nodeALatest.Hash() == nodeBLatest.Hash() {
				Fatal(t, "node A L2 hash", nodeALatest, "matches node B L2 hash", nodeBLatest)
			}
			break
		}
	}

	bridgeBinding, err := bridgegen.NewBridge(l1info.GetAddress("Bridge"), l1client)
	Require(t, err)
	totalBatchesBig, err := bridgeBinding.SequencerMessageCount(&bind.CallOpts{Context: ctx})
	Require(t, err)
	totalBatches := totalBatchesBig.Uint64()

	// Wait until the validators have validated the batches.
	for {
		lastInfo, err := blockValidatorA.ReadLastValidatedInfo()
		if lastInfo == nil || err != nil {
			continue
		}
		t.Log(lastInfo.GlobalState.Batch, totalBatches-1)
		if lastInfo.GlobalState.Batch >= totalBatches-1 {
			break
		}
		time.Sleep(time.Millisecond * 200)
	}
	for {
		lastInfo, err := blockValidatorB.ReadLastValidatedInfo()
		if lastInfo == nil || err != nil {
			continue
		}
		t.Log(lastInfo.GlobalState.Batch, totalBatches-1)
		if lastInfo.GlobalState.Batch >= totalBatches-1 {
			break
		}
		time.Sleep(time.Millisecond * 200)
	}

	provider := l2stateprovider.NewHistoryCommitmentProvider(
		stateManager,
		stateManager,
		stateManager,
		[]l2stateprovider.Height{
			l2stateprovider.Height(blockChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(smallStepChallengeLeafHeight),
		},
		stateManager,
		nil, // Api db
	)

	evilProvider := l2stateprovider.NewHistoryCommitmentProvider(
		stateManagerB,
		stateManagerB,
		stateManagerB,
		[]l2stateprovider.Height{
			l2stateprovider.Height(blockChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(smallStepChallengeLeafHeight),
		},
		stateManagerB,
		nil, // Api db
	)

	stackOpts := []challengemanager.StackOpt{
		challengemanager.StackWithName("honest"),
		challengemanager.StackWithMode(modes.MakeMode),
		challengemanager.StackWithPostingInterval(time.Second * 3),
		challengemanager.StackWithPollingInterval(time.Second),
		challengemanager.StackWithAverageBlockCreationTime(time.Second),
	}

	manager, err := challengemanager.NewChallengeStack(
		assertionChain,
		provider,
		stackOpts...,
	)
	Require(t, err)

	evilStackOpts := append(stackOpts, challengemanager.StackWithName("evil"))

	managerB, err := challengemanager.NewChallengeStack(
		chainB,
		evilProvider,
		evilStackOpts...,
	)
	Require(t, err)

	manager.Start(ctx)
	managerB.Start(ctx)

	chalManager := assertionChain.SpecChallengeManager()
	filterer, err := challengeV2gen.NewEdgeChallengeManagerFilterer(chalManager.Address(), l1client)
	Require(t, err)

	fromBlock := uint64(0)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			latestBlock, err := l1client.HeaderByNumber(ctx, nil)
			Require(t, err)
			toBlock := latestBlock.Number.Uint64()
			if fromBlock == toBlock {
				continue
			}
			filterOpts := &bind.FilterOpts{
				Start:   fromBlock,
				End:     &toBlock,
				Context: ctx,
			}
			it, err := filterer.FilterEdgeConfirmedByOneStepProof(filterOpts, nil, nil)
			Require(t, err)
			for it.Next() {
				if it.Error() != nil {
					t.Fatalf("Error in filter iterator: %v", it.Error())
				}
				t.Log("Received event of OSP confirmation!")
				tx, _, err := l1client.TransactionByHash(ctx, it.Event.Raw.TxHash)
				Require(t, err)
				signer := types.NewCancunSigner(tx.ChainId())
				address, err := signer.Sender(tx)
				Require(t, err)
				if address == l1info.GetDefaultTransactOpts("Asserter", ctx).From {
					t.Log("Honest party won OSP, impossible for evil party to win if honest party continues")
					Require(t, it.Close())
					return
				}
			}
			fromBlock = toBlock
		case <-ctx.Done():
			return
		}
	}
}

// Every 3 seconds, send an L1 transaction to keep the chain moving.
func keepChainMoving(t *testing.T, ctx context.Context, l1Info *BlockchainTestInfo, l1Client *ethclient.Client) {
	delay := time.Second * 3
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(delay)
			if ctx.Err() != nil {
				break
			}
			TransferBalance(t, "Faucet", "Faucet", common.Big0, l1Info, l1Client, ctx)
			latestBlock, err := l1Client.BlockNumber(ctx)
			if ctx.Err() != nil {
				break
			}
			Require(t, err)
			if latestBlock > 150 {
				delay = time.Second
			}
		}
	}
}

func createTestNodeOnL1ForBoldProtocol(
	t *testing.T,
	ctx context.Context,
	isSequencer bool,
	nodeConfig *arbnode.Config,
	chainConfig *params.ChainConfig,
	_ *node.Config,
	rollupStackConf setup.RollupStackConfig,
	l2infoIn info,
) (
	l2info info, currentNode *arbnode.Node, l2client *ethclient.Client, l2stack *node.Node,
	l1info info, l1backend *eth.Ethereum, l1client *ethclient.Client, l1stack *node.Node,
	assertionChain *solimpl.AssertionChain, stakeTokenAddr common.Address,
) {
	if nodeConfig == nil {
		nodeConfig = arbnode.ConfigDefaultL1Test()
	}
	nodeConfig.ParentChainReader.OldHeaderTimeout = time.Minute * 10
	if chainConfig == nil {
		chainConfig = chaininfo.ArbitrumDevTestChainConfig()
	}
	nodeConfig.BatchPoster.DataPoster.MaxMempoolTransactions = 18
	fatalErrChan := make(chan error, 10)
	l1info, l1client, l1backend, l1stack = createTestL1BlockChain(t, nil)
	var l2chainDb ethdb.Database
	var l2arbDb ethdb.Database
	var l2blockchain *core.BlockChain
	l2info = l2infoIn
	if l2info == nil {
		l2info = NewArbTestInfo(t, chainConfig.ChainID)
	}

	l1info.GenerateAccount("RollupOwner")
	l1info.GenerateAccount("Sequencer")
	l1info.GenerateAccount("User")
	l1info.GenerateAccount("Asserter")
	l1info.GenerateAccount("EvilAsserter")

	SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
		l1info.PrepareTx("Faucet", "RollupOwner", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "Sequencer", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "Asserter", 30000, big.NewInt(9223372036854775807), nil),
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

	addresses := deployContractsOnly(t, ctx, l1info, l1client, chainConfig.ChainID, rollupStackConf, stakeToken)
	rollupUser, err := rollupgen.NewRollupUserLogic(addresses.Rollup, l1client)
	Require(t, err)
	chalManagerAddr, err := rollupUser.ChallengeManager(&bind.CallOpts{})
	Require(t, err)
	l1info.SetContract("Bridge", addresses.Bridge)
	l1info.SetContract("SequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("Inbox", addresses.Inbox)
	l1info.SetContract("Rollup", addresses.Rollup)
	l1info.SetContract("UpgradeExecutor", addresses.UpgradeExecutor)

	execConfig := ExecConfigDefaultNonSequencerTest(t)
	Require(t, execConfig.Validate())
	execConfig.Caching.StateScheme = rawdb.HashScheme
	useWasmCache := uint32(1)
	initMessage := getInitMessage(ctx, t, l1client, addresses)
	_, l2stack, l2chainDb, l2arbDb, l2blockchain = createNonL1BlockChainWithStackConfig(t, l2info, "", chainConfig, initMessage, nil, execConfig, useWasmCache)
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

	execConfigFetcher := func() *gethexec.Config { return execConfig }
	execNode, err := gethexec.CreateExecutionNode(ctx, l2stack, l2chainDb, l2blockchain, l1client, execConfigFetcher)
	Require(t, err)

	parentChainId, err := l1client.ChainID(ctx)
	Require(t, err)
	currentNode, err = arbnode.CreateNode(
		ctx, l2stack, execNode, l2arbDb, NewFetcherFromConfig(nodeConfig), l2blockchain.Config(), l1client,
		addresses, sequencerTxOptsPtr, sequencerTxOptsPtr, dataSigner, fatalErrChan, parentChainId,
		nil, // Blob reader.
	)
	Require(t, err)

	l2client = ClientForStack(t, l2stack)

	StartWatchChanErr(t, ctx, fatalErrChan, currentNode)

	opts := l1info.GetDefaultTransactOpts("Asserter", ctx)
	dp, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(l2arbDb, storage.StakerPrefix),
		currentNode.L1Reader,
		&opts,
		NewFetcherFromConfig(nodeConfig),
		currentNode.SyncMonitor,
		parentChainId,
	)
	Require(t, err)
	assertionChainBindings, err := solimpl.NewAssertionChain(
		ctx,
		addresses.Rollup,
		chalManagerAddr,
		&opts,
		butil.NewBackendWrapper(l1client, rpc.LatestBlockNumber),
		bold.NewDataPosterTransactor(dp),
		solimpl.WithRpcHeadBlockNumber(rpc.LatestBlockNumber),
	)
	Require(t, err)
	assertionChain = assertionChainBindings

	return
}

func deployContractsOnly(
	t *testing.T,
	ctx context.Context,
	l1info info,
	backend *ethclient.Client,
	chainId *big.Int,
	rollupStackConf setup.RollupStackConfig,
	stakeToken common.Address,
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
	cfg := challengetesting.GenerateRollupConfig(
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
		challengetesting.WithLayerZeroHeights(&protocol.LayerZeroHeights{
			BlockChallengeHeight:     protocol.Height(blockChallengeLeafHeight),
			BigStepChallengeHeight:   protocol.Height(bigStepChallengeLeafHeight),
			SmallStepChallengeHeight: protocol.Height(smallStepChallengeLeafHeight),
		}),
		challengetesting.WithNumBigStepLevels(uint8(3)),       // TODO: Hardcoded.
		challengetesting.WithConfirmPeriodBlocks(uint64(120)), // TODO: Hardcoded.
	)
	config, err := json.Marshal(chaininfo.ArbitrumDevTestChainConfig())
	Require(t, err)
	cfg.ChainConfig = string(config)
	addresses, err := setup.DeployFullRollupStack(
		ctx,
		butil.NewBackendWrapper(backend, rpc.LatestBlockNumber),
		&l1TransactionOpts,
		l1info.GetAddress("Sequencer"),
		cfg,
		rollupStackConf,
	)
	Require(t, err)

	asserter := l1info.GetDefaultTransactOpts("Asserter", ctx)
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
	tx, err := tokenBindings.TestWETH9Transactor.Transfer(&l1TransactionOpts, asserter.From, seed)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)
	tx, err = tokenBindings.TestWETH9Transactor.Approve(&asserter, addresses.Rollup, value)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)
	tx, err = tokenBindings.TestWETH9Transactor.Approve(&asserter, chalManagerAddr, value)
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

func create2ndNodeWithConfigForBoldProtocol(
	t *testing.T,
	ctx context.Context,
	first *arbnode.Node,
	l1stack *node.Node,
	l1info *BlockchainTestInfo,
	l2InitData *statetransfer.ArbosInitializationInfo,
	nodeConfig *arbnode.Config,
	stackConfig *node.Config,
	rollupStackConf setup.RollupStackConfig,
	stakeTokenAddr common.Address,
) (*ethclient.Client, *arbnode.Node, *solimpl.AssertionChain) {
	fatalErrChan := make(chan error, 10)
	l1rpcClient := l1stack.Attach()
	l1client := ethclient.NewClient(l1rpcClient)
	firstExec, ok := first.Execution.(*gethexec.ExecutionNode)
	if !ok {
		Fatal(t, "not geth execution node")
	}
	chainConfig := firstExec.ArbInterface.BlockChain().Config()
	addresses := deployContractsOnly(t, ctx, l1info, l1client, chainConfig.ChainID, rollupStackConf, stakeTokenAddr)

	l1info.SetContract("EvilBridge", addresses.Bridge)
	l1info.SetContract("EvilSequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("EvilInbox", addresses.Inbox)
	l1info.SetContract("EvilRollup", addresses.Rollup)
	l1info.SetContract("EvilUpgradeExecutor", addresses.UpgradeExecutor)

	if nodeConfig == nil {
		nodeConfig = arbnode.ConfigDefaultL1NonSequencerTest()
	}
	nodeConfig.ParentChainReader.OldHeaderTimeout = 10 * time.Minute
	nodeConfig.BatchPoster.DataPoster.MaxMempoolTransactions = 18
	if stackConfig == nil {
		stackConfig = testhelpers.CreateStackConfigForTest(t.TempDir())
	}
	l2stack, err := node.New(stackConfig)
	Require(t, err)

	l2chainDb, err := l2stack.OpenDatabase("chaindb", 0, 0, "", false)
	Require(t, err)
	l2arbDb, err := l2stack.OpenDatabase("arbdb", 0, 0, "", false)
	Require(t, err)

	AddValNodeIfNeeded(t, ctx, nodeConfig, true, "", "")

	dataSigner := signature.DataSignerFromPrivateKey(l1info.GetInfoWithPrivKey("Sequencer").PrivateKey)
	txOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)

	initReader := statetransfer.NewMemoryInitDataReader(l2InitData)
	initMessage := getInitMessage(ctx, t, l1client, first.DeployInfo)

	execConfig := ExecConfigDefaultNonSequencerTest(t)
	Require(t, execConfig.Validate())
	execConfig.Caching.StateScheme = rawdb.HashScheme
	coreCacheConfig := gethexec.DefaultCacheConfigFor(l2stack, &execConfig.Caching)
	l2blockchain, err := gethexec.WriteOrTestBlockChain(l2chainDb, coreCacheConfig, initReader, chainConfig, initMessage, execConfig.TxLookupLimit, 0)
	Require(t, err)

	execConfigFetcher := func() *gethexec.Config { return execConfig }
	execNode, err := gethexec.CreateExecutionNode(ctx, l2stack, l2chainDb, l2blockchain, l1client, execConfigFetcher)
	Require(t, err)
	l1ChainId, err := l1client.ChainID(ctx)
	Require(t, err)
	l2node, err := arbnode.CreateNode(ctx, l2stack, execNode, l2arbDb, NewFetcherFromConfig(nodeConfig), l2blockchain.Config(), l1client, addresses, &txOpts, &txOpts, dataSigner, fatalErrChan, l1ChainId, nil /* blob reader */)
	Require(t, err)

	l2client := ClientForStack(t, l2stack)

	StartWatchChanErr(t, ctx, fatalErrChan, l2node)

	rollupUserLogic, err := rollupgen.NewRollupUserLogic(addresses.Rollup, l1client)
	Require(t, err)
	chalManagerAddr, err := rollupUserLogic.ChallengeManager(&bind.CallOpts{})
	Require(t, err)
	evilOpts := l1info.GetDefaultTransactOpts("EvilAsserter", ctx)
	dp, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(l2arbDb, storage.StakerPrefix),
		l2node.L1Reader,
		&evilOpts,
		NewFetcherFromConfig(nodeConfig),
		l2node.SyncMonitor,
		l1ChainId,
	)
	Require(t, err)
	assertionChain, err := solimpl.NewAssertionChain(
		ctx,
		addresses.Rollup,
		chalManagerAddr,
		&evilOpts,
		butil.NewBackendWrapper(l1client, rpc.LatestBlockNumber),
		bold.NewDataPosterTransactor(dp),
	)
	Require(t, err)

	return l2client, l2node, assertionChain
}

func makeBoldBatch(
	t *testing.T,
	l2Node *arbnode.Node,
	l2Info *BlockchainTestInfo,
	backend *ethclient.Client,
	sequencer *bind.TransactOpts,
	seqInbox *bridgegen.SequencerInbox,
	seqInboxAddr common.Address,
	numMessages,
	divergeAtIndex int64,
) {
	ctx := context.Background()

	batchBuffer := bytes.NewBuffer([]byte{})
	for i := int64(0); i < numMessages; i++ {
		value := i
		if i == divergeAtIndex {
			value++
		}
		err := writeTxToBatchBold(batchBuffer, l2Info.PrepareTx("Owner", "Destination", 1000000, big.NewInt(value), []byte{}))
		Require(t, err)
	}
	compressed, err := arbcompress.CompressWell(batchBuffer.Bytes())
	Require(t, err)
	message := append([]byte{0}, compressed...)

	seqNum := new(big.Int).Lsh(common.Big1, 256)
	seqNum.Sub(seqNum, common.Big1)
	tx, err := seqInbox.AddSequencerL2BatchFromOrigin8f111f3c(sequencer, seqNum, message, big.NewInt(1), common.Address{}, big.NewInt(0), big.NewInt(0))
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)

	nodeSeqInbox, err := arbnode.NewSequencerInbox(backend, seqInboxAddr, 0)
	Require(t, err)
	batches, err := nodeSeqInbox.LookupBatchesInRange(ctx, receipt.BlockNumber, receipt.BlockNumber)
	Require(t, err)
	if len(batches) == 0 {
		Fatal(t, "batch not found after AddSequencerL2BatchFromOrigin")
	}
	err = l2Node.InboxTracker.AddSequencerBatches(ctx, backend, batches)
	Require(t, err)
	batchMetaData, err := l2Node.InboxTracker.GetBatchMetadata(batches[0].SequenceNumber)
	log.Info("Batch metadata", "md", batchMetaData)
	Require(t, err, "failed to get batch metadata after adding batch:")
}

func writeTxToBatchBold(writer io.Writer, tx *types.Transaction) error {
	txData, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	var segment []byte
	segment = append(segment, arbstate.BatchSegmentKindL2Message)
	segment = append(segment, arbos.L2MessageKind_SignedTx)
	segment = append(segment, txData...)
	err = rlp.Encode(writer, segment)
	return err
}
