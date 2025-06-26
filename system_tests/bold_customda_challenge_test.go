// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build challengetest && !race

package arbtest

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	solimpl "github.com/offchainlabs/bold/chain-abstraction/sol-implementation"
	challengemanager "github.com/offchainlabs/bold/challenge-manager"
	modes "github.com/offchainlabs/bold/challenge-manager/types"
	l2stateprovider "github.com/offchainlabs/bold/layer2-state-provider"
	"github.com/offchainlabs/bold/solgen/go/bridgegen"
	"github.com/offchainlabs/bold/solgen/go/challengeV2gen"
	"github.com/offchainlabs/bold/solgen/go/mocksgen"
	"github.com/offchainlabs/bold/testing/setup"
	butil "github.com/offchainlabs/bold/util"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/bold"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/validator/server_arb"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func TestChallengeProtocolBOLDCustomDASetup(t *testing.T) {
	// Simple test to verify ReferenceDAProofValidator deployment
	testChallengeProtocolBOLDCustomDA(t)
}

func testChallengeProtocolBOLDCustomDA(t *testing.T, spawnerOpts ...server_arb.SpawnerOption) {
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
		UseBlobs:               true,
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
		true,
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
		true,
	)
	defer l2nodeB.StopAndWait()

	genesisA, err := l2nodeA.ExecutionClient.ResultAtMessageIndex(0).Await(ctx)
	Require(t, err)
	genesisB, err := l2nodeB.ExecutionClient.ResultAtMessageIndex(0).Await(ctx)
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

	locator, err := server_common.NewMachineLocator(valCfg.Wasm.RootPath)
	Require(t, err)
	statelessA, err := staker.NewStatelessBlockValidator(
		l2nodeA.InboxReader,
		l2nodeA.InboxTracker,
		l2nodeA.TxStreamer,
		l2nodeA.ExecutionRecorder,
		l2nodeA.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
		locator.LatestWasmModuleRoot(),
	)
	Require(t, err)
	err = statelessA.Start(ctx)
	Require(t, err)
	_, valStackB := createTestValidationNode(t, ctx, &valCfg, spawnerOpts...)

	statelessB, err := staker.NewStatelessBlockValidator(
		l2nodeB.InboxReader,
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		l2nodeB.ExecutionRecorder,
		l2nodeB.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStackB,
		locator.LatestWasmModuleRoot(),
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
		l2nodeA.InboxTracker,
		l2nodeA.TxStreamer,
		l2nodeA.InboxReader,
		nil,
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
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		l2nodeB.InboxReader,
		nil,
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
	nodeAExec, ok := l2nodeA.ExecutionClient.(*gethexec.ExecutionNode)
	if !ok {
		Fatal(t, "not geth execution node")
	}
	nodeBExec, ok := l2nodeB.ExecutionClient.(*gethexec.ExecutionNode)
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
		challengemanager.StackWithMinimumGapToParentAssertion(0),
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
