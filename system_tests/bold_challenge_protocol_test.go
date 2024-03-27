// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//asdasdgo:build challengetest && !race

package arbtest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"os"
	"testing"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	challengemanager "github.com/OffchainLabs/bold/challenge-manager"
	modes "github.com/OffchainLabs/bold/challenge-manager/types"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/solgen/go/bridgegen"
	"github.com/OffchainLabs/bold/solgen/go/mocksgen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
)

// One Arbitrum block had 1,849,212,947 total opcodes. The closest, higher power of two
// is 2^31. So we if we make our small step heights 2^20, we need 2048 big steps
// to cover the block. With 2^20, our small step history commitments will be approx
// 32 Mb of state roots in memory at once.
var (
	blockChallengeLeafHeight     = uint64(1 << 5) // 32
	bigStepChallengeLeafHeight   = uint64(1 << 6)
	smallStepChallengeLeafHeight = uint64(1 << 6)
)

func TestBoldProtocol(t *testing.T) {
	t.Cleanup(func() {
		Require(t, os.RemoveAll("/tmp/good"))
		Require(t, os.RemoveAll("/tmp/evil"))
	})
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs
	l2chainConfig := params.ArbitrumDevTestChainConfig()
	l2info := NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(l2chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	ownerBal := big.NewInt(params.Ether)
	ownerBal.Mul(ownerBal, big.NewInt(1_000_000))
	l2info.GenerateGenesisAccount("Owner", ownerBal)

	_, l2nodeA, _, _, l1info, _, l1client, l1stack, assertionChain, stakeTokenAddr := createTestNodeOnL1ForBoldProtocol(t, ctx, true, nil, l2chainConfig, nil, l2info)
	defer requireClose(t, l1stack)
	defer l2nodeA.StopAndWait()

	// Every 12 seconds, send an L1 transaction to keep the chain moving.
	go func() {
		delay := time.Second * 12
		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(delay)
				balance := big.NewInt(params.GWei)
				TransferBalance(t, "Faucet", "Asserter", balance, l1info, l1client, ctx)
				latestBlock, err := l1client.BlockNumber(ctx)
				Require(t, err)
				if latestBlock > 150 {
					delay = time.Second
				}
			}
		}
	}()

	l2nodeConfig := arbnode.ConfigDefaultL1Test()
	_, l2nodeB, assertionChainB := create2ndNodeWithConfigForBoldProtocol(t, ctx, l2nodeA, l1stack, l1info, &l2info.ArbInitData, l2nodeConfig, nil, stakeTokenAddr)
	defer l2nodeB.StopAndWait()

	nodeAMessage, err := l2nodeA.Execution.HeadMessageNumber()
	Require(t, err)
	nodeBMessage, err := l2nodeB.Execution.HeadMessageNumber()
	Require(t, err)
	if nodeAMessage != nodeBMessage {
		Fatal(t, "node A L2 genesis hash", nodeAMessage, "!= node B L2 genesis hash", nodeBMessage)
	}

	deployAuth := l1info.GetDefaultTransactOpts("RollupOwner", ctx)

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	TransferBalance(t, "Faucet", "Asserter", balance, l1info, l1client, ctx)
	TransferBalance(t, "Faucet", "EvilAsserter", balance, l1info, l1client, ctx)

	t.Log("Setting the minimum assertion period")
	rollup, err := rollupgen.NewRollupAdminLogicTransactor(assertionChain.RollupAddress(), l1client)
	Require(t, err)
	tx, err := rollup.SetMinimumAssertionPeriod(&deployAuth, big.NewInt(0))
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)
	rollup, err = rollupgen.NewRollupAdminLogicTransactor(assertionChainB.RollupAddress(), l1client)
	Require(t, err)
	tx, err = rollup.SetMinimumAssertionPeriod(&deployAuth, big.NewInt(0))
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)

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
		l2nodeA.BlobReader,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	err = statelessA.Start(ctx)
	Require(t, err)

	statelessB, err := staker.NewStatelessBlockValidator(
		l2nodeB.InboxReader,
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		l2nodeB.Execution,
		l2nodeB.ArbDB,
		nil,
		l2nodeB.BlobReader,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	newCtx, newCancel := context.WithCancel(context.Background())
	defer newCancel()
	err = statelessB.Start(newCtx)
	Require(t, err)

	stateManager, err := staker.NewStateManager(
		statelessA,
		"/tmp/good",
		[]l2stateprovider.Height{
			l2stateprovider.Height(blockChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(smallStepChallengeLeafHeight),
		},
		"good",
	)
	Require(t, err)

	stateManagerB, err := staker.NewStateManager(
		statelessB,
		"/tmp/evil",
		[]l2stateprovider.Height{
			l2stateprovider.Height(blockChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(smallStepChallengeLeafHeight),
		},
		"evil",
	)
	Require(t, err)

	chalManagerAddr, err := assertionChain.SpecChallengeManager(ctx)
	Require(t, err)
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
		l1client,
		solimpl.NewDataPosterTransactor(dp),
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
	totalMessagesPosted := int64(0)
	numMessagesPerBatch := int64(5)
	divergeAt := int64(-1)
	makeBoldBatch(t, l2nodeA, l2info, l1client, &sequencerTxOpts, honestSeqInboxBinding, honestSeqInbox, numMessagesPerBatch, divergeAt)
	l2info.Accounts["Owner"].Nonce = 0
	makeBoldBatch(t, l2nodeB, l2info, l1client, &sequencerTxOpts, evilSeqInboxBinding, evilSeqInbox, numMessagesPerBatch, divergeAt)
	totalMessagesPosted += numMessagesPerBatch

	// Next, we post another batch, this time containing more messages.
	// We diverge at message index 5 within the evil node's batch.
	l2info.Accounts["Owner"].Nonce = 5
	numMessagesPerBatch = int64(10)
	makeBoldBatch(t, l2nodeA, l2info, l1client, &sequencerTxOpts, honestSeqInboxBinding, honestSeqInbox, numMessagesPerBatch, divergeAt)
	l2info.Accounts["Owner"].Nonce = 5
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

	// Wait for the validator to validate the batches.
	bridgeBinding, err := bridgegen.NewBridge(l1info.GetAddress("Bridge"), l1client)
	Require(t, err)
	totalBatchesBig, err := bridgeBinding.SequencerMessageCount(&bind.CallOpts{Context: ctx})
	Require(t, err)
	totalBatches := totalBatchesBig.Uint64()
	totalMessageCount, err := l2nodeA.InboxTracker.GetBatchMessageCount(totalBatches - 1)
	Require(t, err)

	// Wait until the validator has validated the batches.
	for {
		_, err1 := l2nodeA.TxStreamer.ResultAtCount(totalMessageCount)
		nodeAHasValidated := err1 == nil

		_, err2 := l2nodeB.TxStreamer.ResultAtCount(totalMessageCount)
		nodeBHasValidated := err2 == nil

		if nodeAHasValidated && nodeBHasValidated {
			break
		}
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
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(smallStepChallengeLeafHeight),
		},
		stateManagerB,
		nil, // Api db
	)

	manager, err := challengemanager.New(
		ctx,
		assertionChain,
		provider,
		assertionChain.RollupAddress(),
		challengemanager.WithName("honest"),
		challengemanager.WithMode(modes.MakeMode),
		challengemanager.WithAddress(l1info.GetDefaultTransactOpts("Asserter", ctx).From),
		challengemanager.WithAssertionPostingInterval(time.Second*30),
		challengemanager.WithAssertionScanningInterval(time.Second),
		challengemanager.WithEdgeTrackerWakeInterval(time.Second*2),
	)
	Require(t, err)

	managerB, err := challengemanager.New(
		ctx,
		chainB,
		evilProvider,
		assertionChain.RollupAddress(),
		challengemanager.WithName("evil"),
		challengemanager.WithMode(modes.MakeMode),
		challengemanager.WithAddress(l1info.GetDefaultTransactOpts("EvilAsserter", ctx).From),
		challengemanager.WithAssertionPostingInterval(time.Second*30),
		challengemanager.WithAssertionScanningInterval(time.Second),
		challengemanager.WithEdgeTrackerWakeInterval(time.Second*2),
	)
	Require(t, err)

	manager.Start(ctx)
	managerB.Start(ctx)

	filterer, err := rollupgen.NewRollupUserLogicFilterer(assertionChain.RollupAddress(), l1client)
	Require(t, err)
	userLogic, err := rollupgen.NewRollupUserLogic(assertionChain.RollupAddress(), l1client)
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
			it, err := filterer.FilterAssertionConfirmed(filterOpts, nil)
			Require(t, err)
			for it.Next() {
				if it.Error() != nil {
					t.Fatalf("Error in filter iterator: %v", it.Error())
				}
				assertion, err := userLogic.GetAssertion(&bind.CallOpts{}, it.Event.AssertionHash)
				Require(t, err)
				isChallenged := assertion.FirstChildBlock != 0 && assertion.SecondChildBlock != 0
				if isChallenged {
					t.Logf("Assertion confirmed %#x", it.Event.AssertionHash)
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

func createTestNodeOnL1ForBoldProtocol(
	t *testing.T,
	ctx context.Context,
	isSequencer bool,
	nodeConfig *arbnode.Config,
	chainConfig *params.ChainConfig,
	stackConfig *node.Config,
	l2info_in info,
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
		chainConfig = params.ArbitrumDevTestChainConfig()
	}
	nodeConfig.BatchPoster.DataPoster.MaxMempoolTransactions = 18
	fatalErrChan := make(chan error, 10)
	l1info, l1client, l1backend, l1stack = createTestL1BlockChain(t, nil)
	var l2chainDb ethdb.Database
	var l2arbDb ethdb.Database
	var l2blockchain *core.BlockChain
	l2info = l2info_in
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

	addresses := deployContractsOnly(t, ctx, l1info, l1client, chainConfig.ChainID, stakeToken)
	rollupUser, err := rollupgen.NewRollupUserLogic(addresses.Rollup, l1client)
	Require(t, err)
	chalManagerAddr, err := rollupUser.ChallengeManager(&bind.CallOpts{})
	Require(t, err)
	l1info.SetContract("Bridge", addresses.Bridge)
	l1info.SetContract("SequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("Inbox", addresses.Inbox)

	_, l2stack, l2chainDb, l2arbDb, l2blockchain = createL2BlockChainWithStackConfig(t, l2info, "", chainConfig, getInitMessage(ctx, t, l1client, addresses), stackConfig, nil)
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

	AddDefaultValNode(t, ctx, nodeConfig, true)

	execConfig := gethexec.ConfigDefaultTest()
	Require(t, execConfig.Validate())
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

	Require(t, currentNode.Start(ctx))

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
		l1client,
		solimpl.NewDataPosterTransactor(dp),
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
	stakeToken common.Address,
) *chaininfo.RollupAddresses {
	l1TransactionOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	locator, err := server_common.NewMachineLocator("")
	Require(t, err)
	wasmModuleRoot := locator.LatestWasmModuleRoot()

	loserStakeEscrow := common.Address{}
	genesisExecutionState := rollupgen.AssertionState{
		GlobalState:    rollupgen.GlobalState{},
		MachineStatus:  1,
		EndHistoryRoot: [32]byte{},
	}
	genesisInboxCount := big.NewInt(0)
	anyTrustFastConfirmer := common.Address{}
	miniStakeValues := []*big.Int{big.NewInt(5), big.NewInt(4), big.NewInt(3), big.NewInt(2), big.NewInt(1), big.NewInt(1), big.NewInt(1)}
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
			BlockChallengeHeight:     blockChallengeLeafHeight,
			BigStepChallengeHeight:   bigStepChallengeLeafHeight,
			SmallStepChallengeHeight: smallStepChallengeLeafHeight,
		}),
		challenge_testing.WithNumBigStepLevels(uint8(5)),       // TODO: Hardcoded.
		challenge_testing.WithConfirmPeriodBlocks(uint64(120)), // TODO: Hardcoded.
	)
	config, err := json.Marshal(params.ArbitrumDevTestChainConfig())
	Require(t, err)
	cfg.ChainConfig = string(config)
	addresses, err := setup.DeployFullRollupStack(
		ctx,
		backend,
		&l1TransactionOpts,
		l1info.GetAddress("Sequencer"),
		cfg,
		false, // do not use mock bridge.
		false, // do not use a mock one step prover
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
	addresses := deployContractsOnly(t, ctx, l1info, l1client, chainConfig.ChainID, stakeTokenAddr)

	l1info.SetContract("EvilBridge", addresses.Bridge)
	l1info.SetContract("EvilSequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("EvilInbox", addresses.Inbox)

	if nodeConfig == nil {
		nodeConfig = arbnode.ConfigDefaultL1NonSequencerTest()
	}
	nodeConfig.ParentChainReader.OldHeaderTimeout = 10 * time.Minute
	nodeConfig.BatchPoster.DataPoster.MaxMempoolTransactions = 18
	if stackConfig == nil {
		stackConfig = createStackConfigForTest(t.TempDir())
	}
	l2stack, err := node.New(stackConfig)
	Require(t, err)

	l2chainDb, err := l2stack.OpenDatabase("chaindb", 0, 0, "", false)
	Require(t, err)
	l2arbDb, err := l2stack.OpenDatabase("arbdb", 0, 0, "", false)
	Require(t, err)

	AddDefaultValNode(t, ctx, nodeConfig, true)

	dataSigner := signature.DataSignerFromPrivateKey(l1info.GetInfoWithPrivKey("Sequencer").PrivateKey)
	txOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)

	initReader := statetransfer.NewMemoryInitDataReader(l2InitData)
	initMessage := getInitMessage(ctx, t, l1client, first.DeployInfo)

	execConfig := gethexec.ConfigDefaultTest()
	Require(t, execConfig.Validate())

	l2blockchain, err := gethexec.WriteOrTestBlockChain(l2chainDb, nil, initReader, chainConfig, initMessage, execConfig.TxLookupLimit, 0)
	Require(t, err)

	execConfigFetcher := func() *gethexec.Config { return execConfig }
	execNode, err := gethexec.CreateExecutionNode(ctx, l2stack, l2chainDb, l2blockchain, l1client, execConfigFetcher)
	Require(t, err)
	l1ChainId, err := l1client.ChainID(ctx)
	Require(t, err)
	l2node, err := arbnode.CreateNode(ctx, l2stack, execNode, l2arbDb, NewFetcherFromConfig(nodeConfig), l2blockchain.Config(), l1client, addresses, &txOpts, &txOpts, dataSigner, fatalErrChan, l1ChainId, nil /* blob reader */)
	Require(t, err)

	Require(t, l2node.Start(ctx))

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
		l1client,
		solimpl.NewDataPosterTransactor(dp),
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
	messagesPerBatch,
	divergeAtIndex int64,
) {
	ctx := context.Background()

	batchBuffer := bytes.NewBuffer([]byte{})
	for i := int64(0); i < messagesPerBatch; i++ {
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
	tx, err := seqInbox.AddSequencerL2BatchFromOrigin0(sequencer, seqNum, message, big.NewInt(1), common.Address{}, big.NewInt(0), big.NewInt(0))
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
	_, err = l2Node.InboxTracker.GetBatchMetadata(0)
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
