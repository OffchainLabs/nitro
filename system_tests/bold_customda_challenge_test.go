// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build challengetest && !race

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
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
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/daclient"
	"github.com/offchainlabs/nitro/daprovider/referenceda"
	dapserver "github.com/offchainlabs/nitro/daprovider/server"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/bold"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/validator/server_arb"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func TestChallengeProtocolBOLDCustomDA(t *testing.T) {
	testChallengeProtocolBOLDCustomDA(t)
}

// createReferenceDAProviderServer creates and starts a ReferenceDA provider server with automatic port selection
func createReferenceDAProviderServer(t *testing.T, ctx context.Context) (*http.Server, string) {
	// Create ReferenceDA components
	reader := referenceda.NewReader()
	writer := referenceda.NewWriter()
	validator := referenceda.NewValidator()

	// Create server config with automatic port selection
	serverConfig := &dapserver.ServerConfig{
		Addr:               "127.0.0.1",
		Port:               0, // 0 means automatic port selection
		EnableDAWriter:     true,
		ServerTimeouts:     dapserver.DefaultServerConfig.ServerTimeouts,
		RPCServerBodyLimit: dapserver.DefaultServerConfig.RPCServerBodyLimit,
	}

	// Create the provider server
	server, err := dapserver.NewServerWithDAPProvider(ctx, serverConfig, reader, writer, validator)
	Require(t, err)

	// Extract the actual address with port
	// The server.Addr contains "http://" prefix, we need to strip it
	serverAddr := strings.TrimPrefix(server.Addr, "http://")

	// Create the full URL for client connection
	serverURL := fmt.Sprintf("http://%s", serverAddr)

	t.Logf("Started ReferenceDA provider server at %s", serverURL)

	return server, serverURL
}

// createEvilDAProviderServer creates and starts a DA provider server with an evil provider that can return different data
func createEvilDAProviderServer(t *testing.T, ctx context.Context) (*http.Server, string, *EvilDAProvider) {
	// Create evil DA provider
	evilProvider := NewEvilDAProvider()

	// Create server config with automatic port selection
	serverConfig := &dapserver.ServerConfig{
		Addr:               "127.0.0.1",
		Port:               0, // automatic port selection
		EnableDAWriter:     true,
		ServerTimeouts:     dapserver.DefaultServerConfig.ServerTimeouts,
		RPCServerBodyLimit: dapserver.DefaultServerConfig.RPCServerBodyLimit,
	}

	// Note: We can use a regular writer since both nodes share the singleton storage
	writer := referenceda.NewWriter()
	server, err := dapserver.NewServerWithDAPProvider(ctx, serverConfig, evilProvider, writer, evilProvider)
	Require(t, err)

	// Extract the actual address with port
	serverAddr := strings.TrimPrefix(server.Addr, "http://")
	serverURL := fmt.Sprintf("http://%s", serverAddr)

	t.Logf("Started evil DA provider server at %s", serverURL)

	return server, serverURL, evilProvider
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

	// Create and start ReferenceDA provider server for node A
	providerServerA, providerURLNodeA := createReferenceDAProviderServer(t, ctx)
	t.Cleanup(func() {
		if err := providerServerA.Shutdown(context.Background()); err != nil {
			t.Logf("Error shutting down provider server A: %v", err)
		}
	})

	// Create and start evil DA provider server for node B
	providerServerB, providerURLNodeB, evilProvider := createEvilDAProviderServer(t, ctx)
	t.Cleanup(func() {
		if err := providerServerB.Shutdown(context.Background()); err != nil {
			t.Logf("Error shutting down provider server B: %v", err)
		}
	})

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

	// Configure external DA for node A
	nodeConfigA := arbnode.ConfigDefaultL1Test()
	nodeConfigA.DA.Mode = "external"
	nodeConfigA.DA.ExternalProvider.Enable = true
	nodeConfigA.DA.ExternalProvider.RPC.URL = providerURLNodeA

	_, l2nodeA, _, _, l1info, _, l1client, l1stack, assertionChain, stakeTokenAddr := createTestNodeOnL1ForBoldProtocol(
		t,
		ctx,
		true,
		nodeConfigA,
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

	// Configure external DA for node B
	l2nodeConfig := arbnode.ConfigDefaultL1Test()
	l2nodeConfig.DA.Mode = "external"
	l2nodeConfig.DA.ExternalProvider.Enable = true
	l2nodeConfig.DA.ExternalProvider.RPC.URL = providerURLNodeB

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

	// Create DA validators for both nodes
	daClientConfigA := func() *rpcclient.ClientConfig {
		return &rpcclient.ClientConfig{
			URL: providerURLNodeA,
		}
	}
	daClientA, err := daclient.NewClient(ctx, daClientConfigA)
	Require(t, err)

	daClientConfigB := func() *rpcclient.ClientConfig {
		return &rpcclient.ClientConfig{
			URL: providerURLNodeB,
		}
	}
	daClientB, err := daclient.NewClient(ctx, daClientConfigB)
	Require(t, err)

	// Create DA readers for validators
	dapReadersA := []daprovider.Reader{daClientA}
	dapReadersB := []daprovider.Reader{daClientB}

	statelessA, err := staker.NewStatelessBlockValidator(
		l2nodeA.InboxReader,
		l2nodeA.InboxTracker,
		l2nodeA.TxStreamer,
		l2nodeA.ExecutionRecorder,
		l2nodeA.ArbDB,
		dapReadersA,
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
		dapReadersB,
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

	// Create ProofEnhancers from DA validators
	proofEnhancerA := server_arb.NewProofEnhancementManager()
	customDAEnhancerA := server_arb.NewCustomDAProofEnhancer(daClientA, l2nodeA.InboxTracker, l2nodeA.InboxReader)
	proofEnhancerA.RegisterEnhancer(server_arb.MarkerCustomDARead, customDAEnhancerA)

	proofEnhancerB := server_arb.NewProofEnhancementManager()
	customDAEnhancerB := server_arb.NewCustomDAProofEnhancer(daClientB, l2nodeB.InboxTracker, l2nodeB.InboxReader)
	proofEnhancerB.RegisterEnhancer(server_arb.MarkerCustomDARead, customDAEnhancerB)

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
		proofEnhancerA,
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
		proofEnhancerB,
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

	// Create DA writers for both nodes
	daWriterA := referenceda.NewWriter()

	totalMessagesPosted := int64(0)
	numMessagesPerBatch := int64(5)
	divergeAt := int64(-1)

	// First batch - no divergence
	goodBatchData := createBoldBatchData(t, l2info, numMessagesPerBatch, divergeAt)

	// Post good batch through node A's DA and get certificate
	certificate := postBatchWithDA(t, l2nodeA, l1client, &sequencerTxOpts,
		honestSeqInboxBinding, honestSeqInbox, goodBatchData, daWriterA)

	// Post same certificate to node B's sequencer inbox
	// Since there's no divergence yet, evil provider will just pass through
	postBatchWithExistingCertificate(t, l2nodeB, l1client, &sequencerTxOpts,
		evilSeqInboxBinding, evilSeqInbox, certificate)

	totalMessagesPosted += numMessagesPerBatch

	// Log first batch messages (batch 0 - appears to be non-CustomDA initial batch)
	t.Logf("\n======== BATCH 0 (initial non-CustomDA batch) ========")
	// Wait a bit for nodes to process
	time.Sleep(100 * time.Millisecond)

	// Get and log batch 0 from both nodes
	msgA0, _, err := l2nodeA.InboxReader.GetSequencerMessageBytes(ctx, 0)
	if err != nil {
		t.Logf("Error getting batch 0 from node A: %v", err)
	} else {
		PrintSequencerInboxMessage(t, "Node A (Honest) - Batch 0", msgA0)
	}

	msgB0, _, err := l2nodeB.InboxReader.GetSequencerMessageBytes(ctx, 0)
	if err != nil {
		t.Logf("Error getting batch 0 from node B: %v", err)
	} else {
		PrintSequencerInboxMessage(t, "Node B (Evil) - Batch 0", msgB0)
	}

	if msgA0 != nil && msgB0 != nil {
		CompareSequencerInboxMessages(t, msgA0, msgB0)
	}

	// Next, we post another batch, this time with divergence.
	// We diverge at message index 5 within the evil node's batch.
	l2info.Accounts["Owner"].Nonce.Store(5)
	numMessagesPerBatch = int64(10)

	// Create both good and evil batch data
	goodBatchData2 := createBoldBatchData(t, l2info, numMessagesPerBatch, -1) // No divergence
	l2info.Accounts["Owner"].Nonce.Store(5)                                   // reset our tracking of owner nonce
	evilBatchData2 := createBoldBatchData(t, l2info, numMessagesPerBatch, 5)  // Diverge at index 5

	// Post good batch through node A and get certificate
	certificate2 := postBatchWithDA(t, l2nodeA, l1client, &sequencerTxOpts,
		honestSeqInboxBinding, honestSeqInbox, goodBatchData2, daWriterA)

	// Configure evil mapping BEFORE node B processes the certificate
	dataHash := common.Hash(certificate2[1:33])
	evilProvider.SetMapping(dataHash, evilBatchData2)

	// Post same certificate to node B's sequencer inbox
	// Now the evil provider will return different data
	postBatchWithExistingCertificate(t, l2nodeB, l1client, &sequencerTxOpts,
		evilSeqInboxBinding, evilSeqInbox, certificate2)

	totalMessagesPosted += numMessagesPerBatch

	// Log second batch messages (batch 1 - first CustomDA batch without divergence)
	t.Logf("\n======== BATCH 1 (first CustomDA batch - no divergence) ========")
	// Wait a bit for nodes to process
	time.Sleep(100 * time.Millisecond)

	// Get and log batch 1 from both nodes
	msgA1, _, err := l2nodeA.InboxReader.GetSequencerMessageBytes(ctx, 1)
	if err != nil {
		t.Logf("Error getting batch 1 from node A: %v", err)
	} else {
		PrintSequencerInboxMessage(t, "Node A (Honest) - Batch 1", msgA1)
	}

	msgB1, _, err := l2nodeB.InboxReader.GetSequencerMessageBytes(ctx, 1)
	if err != nil {
		t.Logf("Error getting batch 1 from node B: %v", err)
	} else {
		PrintSequencerInboxMessage(t, "Node B (Evil) - Batch 1", msgB1)
	}

	if msgA1 != nil && msgB1 != nil {
		CompareSequencerInboxMessages(t, msgA1, msgB1)
	}

	// Log third batch messages (batch 2 - second CustomDA batch with divergence)
	t.Logf("\n======== BATCH 2 (second CustomDA batch - WITH DIVERGENCE) ========")
	// Get and log batch 2 from both nodes
	msgA2, _, err := l2nodeA.InboxReader.GetSequencerMessageBytes(ctx, 2)
	if err != nil {
		t.Logf("Error getting batch 2 from node A: %v", err)
	} else {
		PrintSequencerInboxMessage(t, "Node A (Honest) - Batch 2", msgA2)
	}

	msgB2, _, err := l2nodeB.InboxReader.GetSequencerMessageBytes(ctx, 2)
	if err != nil {
		t.Logf("Error getting batch 2 from node B: %v", err)
	} else {
		PrintSequencerInboxMessage(t, "Node B (Evil) - Batch 2", msgB2)
	}

	if msgA2 != nil && msgB2 != nil {
		CompareSequencerInboxMessages(t, msgA2, msgB2)
	}

	bcA, err := l2nodeA.InboxTracker.GetBatchCount()
	Require(t, err)
	bcB, err := l2nodeB.InboxTracker.GetBatchCount()
	Require(t, err)
	msgA, err := l2nodeA.InboxTracker.GetBatchMessageCount(bcA - 1)
	Require(t, err)
	msgB, err := l2nodeB.InboxTracker.GetBatchMessageCount(bcB - 1)
	Require(t, err)

	t.Logf("\nNode A batch count %d, msgs %d", bcA, msgA)
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

	evilHistoryProvider := l2stateprovider.NewHistoryCommitmentProvider(
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
		evilHistoryProvider,
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
					time.Sleep(5 * time.Second)
					return
				}
			}
			fromBlock = toBlock
		case <-ctx.Done():
			return
		}
	}
}
