// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build challengetest && !race

package arbtest

import (
	"bytes"
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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	solimpl "github.com/offchainlabs/nitro/bold/chain-abstraction/sol-implementation"
	challengemanager "github.com/offchainlabs/nitro/bold/challenge-manager"
	modes "github.com/offchainlabs/nitro/bold/challenge-manager/types"
	l2stateprovider "github.com/offchainlabs/nitro/bold/layer2-state-provider"
	"github.com/offchainlabs/nitro/bold/testing/setup"
	butil "github.com/offchainlabs/nitro/bold/util"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/daclient"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	"github.com/offchainlabs/nitro/daprovider/referenceda"
	dapserver "github.com/offchainlabs/nitro/daprovider/server"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/challengeV2gen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/bold"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/validator/proofenhancement"
	"github.com/offchainlabs/nitro/validator/server_arb"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
)

// Test with evil data but good certificate
// Evil validator will fail at OSP with "Invalid preimage hash"
func TestChallengeProtocolBOLDCustomDA_EvilDataGoodCert(t *testing.T) {
	testChallengeProtocolBOLDCustomDA(t, EvilDataGoodCert)
}

// Test with evil data and evil certificate
// Evil validator will fail at OSP with "WRONG_CERTIFICATE_HASH"
func TestChallengeProtocolBOLDCustomDA_EvilDataEvilCert(t *testing.T) {
	testChallengeProtocolBOLDCustomDA(t, EvilDataEvilCert)
}

// Test with certificate signed by untrusted key
// Evil validator will lie about certificate validity and will fail at OSP with "CLAIMED_VALID_BUT_INVALID"
func TestChallengeProtocolBOLDCustomDA_UntrustedSignerCert(t *testing.T) {
	testChallengeProtocolBOLDCustomDA(t, UntrustedSignerCert)
}

// Test with valid certificate but validator claims it's invalid
// Evil validator will fail at OSP with "CLAIMED_INVALID_BUT_VALID"
func TestChallengeProtocolBOLDCustomDA_ValidCertClaimedInvalid(t *testing.T) {
	testChallengeProtocolBOLDCustomDA(t, ValidCertClaimedInvalid)
}

// postBatchWithDA posts a batch through DA and returns the certificate
func postBatchWithDA(
	t *testing.T,
	l2Node *arbnode.Node,
	backend *ethclient.Client,
	sequencer *bind.TransactOpts,
	seqInbox *bridgegen.SequencerInbox,
	seqInboxAddr common.Address,
	batchData []byte,
	daWriter daprovider.Writer,
) []byte {
	ctx := context.Background()

	// Store data in DA provider
	certificate, err := daWriter.Store(batchData, 3600).Await(ctx)
	Require(t, err)

	// Certificate already contains the CustomDA header flag
	message := certificate

	// Post to L1
	receipt := postBatchToL1(t, ctx, backend, sequencer, seqInbox, message)

	// Sync to node
	syncBatchToNode(t, ctx, backend, l2Node, seqInboxAddr, receipt, "")

	return certificate
}

// createEvilDAProviderServer creates and starts a DA provider server with an evil provider that can return different data
func createEvilDAProviderServer(t *testing.T, ctx context.Context, l1Client *ethclient.Client, validatorAddr common.Address) (*http.Server, string, *EvilDAProvider) {
	// Create evil DA provider
	evilProvider := NewEvilDAProvider(l1Client, validatorAddr)

	// Create server config with automatic port selection
	serverConfig := &dapserver.ServerConfig{
		Addr:               "127.0.0.1",
		Port:               0, // automatic port selection
		EnableDAWriter:     true,
		ServerTimeouts:     dapserver.DefaultServerConfig.ServerTimeouts,
		RPCServerBodyLimit: dapserver.DefaultServerConfig.RPCServerBodyLimit,
	}

	// Use asserting writer to ensure evil provider is never used for writing.
	// In this test we call the writers directly to have more control over batch posting.
	writer := &assertingWriter{}
	headerBytes := []byte{daprovider.DACertificateMessageHeaderFlag}
	server, err := dapserver.NewServerWithDAPProvider(ctx, serverConfig, evilProvider, writer, evilProvider, headerBytes, data_streaming.PayloadCommitmentVerifier())
	Require(t, err)

	// Extract the actual address with port
	serverAddr := strings.TrimPrefix(server.Addr, "http://")
	serverURL := fmt.Sprintf("http://%s", serverAddr)

	t.Logf("Started evil DA provider server at %s", serverURL)

	return server, serverURL, evilProvider
}

// assertingWriter is a Writer that panics if Store is called
type assertingWriter struct{}

func (w *assertingWriter) Store(message []byte, timeout uint64) containers.PromiseInterface[[]byte] {
	panic("assertingWriter.Store should never be called - evil provider server should not be used for writing")
}

func (w *assertingWriter) GetMaxMessageSize() containers.PromiseInterface[int] {
	panic("assertingWriter.GetMaxMessageSize should never be called - evil provider server should not be used for writing")
}

// createNodeBWithSharedContracts creates a second node that uses the same contracts as the first node
func createNodeBWithSharedContracts(
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
	l1client *ethclient.Client,
	assertionChain *solimpl.AssertionChain,
) (*ethclient.Client, *arbnode.Node) {
	fatalErrChan := make(chan error, 10)

	firstExec, ok := first.ExecutionClient.(*gethexec.ExecutionNode)
	if !ok {
		Fatal(t, "not geth execution node")
	}
	chainConfig := firstExec.ArbInterface.BlockChain().Config()

	// Use the same addresses as the first node
	addresses := first.DeployInfo

	if nodeConfig == nil {
		nodeConfig = arbnode.ConfigDefaultL1NonSequencerTest()
	}
	nodeConfig.ParentChainReader.OldHeaderTimeout = 10 * time.Minute
	nodeConfig.BatchPoster.DataPoster.MaxMempoolTransactions = 18
	if stackConfig == nil {
		stackConfig = testhelpers.CreateStackConfigForTest(t.TempDir())
		stackConfig.DBEngine = rawdb.DBPebble
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

	execConfig := ExecConfigDefaultNonSequencerTest(t, rawdb.HashScheme)
	Require(t, execConfig.Validate())
	coreCacheConfig := gethexec.DefaultCacheConfigFor(&execConfig.Caching)
	l2blockchain, err := gethexec.WriteOrTestBlockChain(l2chainDb, coreCacheConfig, initReader, chainConfig, nil, nil, initMessage, &execConfig.TxIndexer, 0)
	Require(t, err)

	execNode, err := gethexec.CreateExecutionNode(ctx, l2stack, l2chainDb, l2blockchain, l1client, NewCommonConfigFetcher(execConfig), big.NewInt(1337), 0)
	Require(t, err)
	l1ChainId, err := l1client.ChainID(ctx)
	Require(t, err)
	locator, err := server_common.NewMachineLocator("")
	Require(t, err)

	// Create node using the same addresses as the first node
	l2node, err := arbnode.CreateNodeFullExecutionClient(ctx, l2stack, execNode, execNode, execNode, execNode, l2arbDb, NewCommonConfigFetcher(nodeConfig), l2blockchain.Config(), l1client, addresses, &txOpts, &txOpts, dataSigner, fatalErrChan, l1ChainId, nil /* blob reader */, locator.LatestWasmModuleRoot())
	Require(t, err)

	l2client := ClientForStack(t, l2stack)

	StartWatchChanErr(t, ctx, fatalErrChan, l2node)

	return l2client, l2node
}

func testChallengeProtocolBOLDCustomDA(t *testing.T, evilStrategy EvilStrategy, spawnerOpts ...server_arb.SpawnerOption) {
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

	switch evilStrategy {
	case EvilDataGoodCert:
		t.Log("Testing EvilDataGoodCert strategy: Evil data with good certificate")
	case EvilDataEvilCert:
		t.Log("Testing EvilDataEvilCert strategy: Evil data with evil certificate (matching)")
	case UntrustedSignerCert:
		t.Log("Testing UntrustedSignerCert strategy: Certificate signed by untrusted key")
	}

	// First set up L1 and deploy contracts to get validator address
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

	// Configure external DA (we'll update the URL after creating providers)
	nodeConfigA := arbnode.ConfigDefaultL1Test()
	nodeConfigA.DA.ExternalProvider.Enable = true

	// Set up L1 first to get validator address
	l1info, l1backend, l1client, l1stack, addresses, stakeTokenAddr, asserterOpts, signerCfg := setupL1ForBoldProtocol(
		t, ctx, sconf, l2info, false, nodeConfigA, l2chainConfig, true, // useExternalSigner=false, enableCustomDA=true
	)
	defer requireClose(t, l1stack)

	// Now we can get the validator address and DA signer
	validatorAddr := l1info.GetAddress("ReferenceDAProofValidator")
	dataSigner := signature.DataSignerFromPrivateKey(l1info.GetInfoWithPrivKey("DASigner").PrivateKey)

	// Create and start ReferenceDA provider server for node A
	providerServerA, providerURLNodeA := createReferenceDAProviderServer(t, ctx, l1client, validatorAddr, dataSigner, 0)
	t.Cleanup(func() {
		if err := providerServerA.Shutdown(context.Background()); err != nil {
			t.Logf("Error shutting down provider server A: %v", err)
		}
	})

	// Create and start evil DA provider server for node B
	providerServerB, providerURLNodeB, evilProvider := createEvilDAProviderServer(t, ctx, l1client, validatorAddr)
	t.Cleanup(func() {
		if err := providerServerB.Shutdown(context.Background()); err != nil {
			t.Logf("Error shutting down provider server B: %v", err)
		}
	})

	// For UntrustedSignerCert, prepare untrusted signer details for later use
	var untrustedSigner signature.DataSignerFunc
	if evilStrategy == UntrustedSignerCert {
		// Create untrusted signer
		untrustedKey, err := crypto.GenerateKey()
		Require(t, err)
		untrustedSigner = signature.DataSignerFromPrivateKey(untrustedKey)

		// Get untrusted signer address for evil provider to recognize
		untrustedCert, err := referenceda.NewCertificate([]byte{1, 2, 3}, untrustedSigner)
		Require(t, err)
		untrustedSignerAddr, err := untrustedCert.RecoverSigner()
		Require(t, err)

		// Configure evil provider to lie about untrusted cert validity
		evilProvider.SetUntrustedSignerAddress(untrustedSignerAddr)

		t.Log("UntrustedSignerCert strategy: Will use untrusted signer for second certificate")
	}

	// Now update node config with provider URLs and create L2 nodes
	nodeConfigA.DA.ExternalProvider.RPC.URL = providerURLNodeA

	// Create L2 node A
	l2info, l2nodeA, _, _, assertionChain := createL2NodeForBoldProtocol(
		t, ctx, true, nodeConfigA, l2chainConfig, l2info,
		l1info, l1backend, l1client, l1stack, addresses, stakeTokenAddr,
		false, asserterOpts, signerCfg, // useExternalSigner=false
	)
	defer l2nodeA.StopAndWait()

	// Make sure we shut down test functionality before the rest of the node
	ctx, cancelCtx = context.WithCancel(ctx)
	defer cancelCtx()

	go keepChainMoving(t, ctx, l1info, l1client)

	// Configure external DA for node B
	l2nodeConfig := arbnode.ConfigDefaultL1Test()
	l2nodeConfig.DA.ExternalProvider.Enable = true
	l2nodeConfig.DA.ExternalProvider.RPC.URL = providerURLNodeB

	// Create node B using the same contracts as node A
	l2clientB, l2nodeB := createNodeBWithSharedContracts(
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
		l1client,
		assertionChain,
	)
	defer l2nodeB.StopAndWait()
	_ = l2clientB // suppress unused variable warning

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
	daClientA, err := daclient.NewClient(ctx, daclient.TestClientConfig(providerURLNodeA), data_streaming.PayloadCommiter())
	Require(t, err)

	daClientB, err := daclient.NewClient(ctx, daclient.TestClientConfig(providerURLNodeB), data_streaming.PayloadCommiter())
	Require(t, err)

	// Create DA readers for validators
	dapReadersA := daprovider.NewDAProviderRegistry()
	err = dapReadersA.SetupDACertificateReader(daClientA, daClientA)
	Require(t, err)

	dapReadersB := daprovider.NewDAProviderRegistry()
	err = dapReadersB.SetupDACertificateReader(daClientB, daClientB)
	Require(t, err)

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
	proofEnhancerA := proofenhancement.NewProofEnhancementManager()
	customDAEnhancerA := proofenhancement.NewReadPreimageProofEnhancer(dapReadersA, l2nodeA.InboxTracker, l2nodeA.InboxReader)
	proofEnhancerA.RegisterEnhancer(proofenhancement.MarkerCustomDAReadPreimage, customDAEnhancerA)
	validateCertificateEnhancerA := proofenhancement.NewValidateCertificateProofEnhancer(dapReadersA, l2nodeA.InboxTracker, l2nodeA.InboxReader)
	proofEnhancerA.RegisterEnhancer(proofenhancement.MarkerCustomDAValidateCertificate, validateCertificateEnhancerA)

	proofEnhancerB := proofenhancement.NewProofEnhancementManager()
	customDAEnhancerB := proofenhancement.NewReadPreimageProofEnhancer(dapReadersB, l2nodeB.InboxTracker, l2nodeB.InboxReader)
	validateCertificateEnhancerB := proofenhancement.NewValidateCertificateProofEnhancer(dapReadersB, l2nodeB.InboxTracker, l2nodeB.InboxReader)
	proofEnhancerB.RegisterEnhancer(proofenhancement.MarkerCustomDAValidateCertificate, validateCertificateEnhancerB)

	// For EvilDataEvilCert strategy, wrap the enhancer to inject evil certificates
	var evilEnhancer *EvilCustomDAProofEnhancer
	if evilStrategy == EvilDataEvilCert {
		evilEnhancer = NewEvilCustomDAProofEnhancer(customDAEnhancerB)
		proofEnhancerB.RegisterEnhancer(proofenhancement.MarkerCustomDAReadPreimage, evilEnhancer)
	} else {
		proofEnhancerB.RegisterEnhancer(proofenhancement.MarkerCustomDAReadPreimage, customDAEnhancerB)
	}

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
		NewCommonConfigFetcher(l2nodeConfig),
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

	seqInbox := l1info.GetAddress("SequencerInbox")
	seqInboxBinding, err := bridgegen.NewSequencerInbox(seqInbox, l1client)
	Require(t, err)

	// Post batches to the shared sequencer inbox
	seqInboxABI, err := abi.JSON(strings.NewReader(bridgegen.SequencerInboxABI))
	Require(t, err)

	upgradeExec, err := mocksgen.NewUpgradeExecutorMock(l1info.GetAddress("UpgradeExecutor"), l1client)
	Require(t, err)
	data, err := seqInboxABI.Pack(
		"setIsBatchPoster",
		sequencerTxOpts.From,
		true,
	)
	Require(t, err)
	rollupOwnerOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	_, err = upgradeExec.ExecuteCall(&rollupOwnerOpts, seqInbox, data)
	Require(t, err)

	// Create DA writers for both nodes
	daWriterA := referenceda.NewWriter(dataSigner, referenceda.DefaultConfig.MaxBatchSize)

	totalMessagesPosted := int64(0)
	numMessagesPerBatch := int64(5)
	divergeAt := int64(-1)

	// First batch - no divergence
	goodBatchData := createBoldBatchData(t, l2info, numMessagesPerBatch, divergeAt)

	// Post good batch through node A's DA and get certificate
	_ = postBatchWithDA(t, l2nodeA, l1client, &sequencerTxOpts,
		seqInboxBinding, seqInbox, goodBatchData, daWriterA)

	// Both nodes will read this certificate from the shared sequencer inbox
	// Since there's no divergence yet, both will get the same data

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
		PrintSequencerInboxMessage(t, "Node A - Batch 0", msgA0)
	}

	msgB0, _, err := l2nodeB.InboxReader.GetSequencerMessageBytes(ctx, 0)
	if err != nil {
		t.Logf("Error getting batch 0 from node B: %v", err)
	}

	// Verify messages are identical with shared inbox
	if msgA0 != nil && msgB0 != nil {
		if !bytes.Equal(msgA0, msgB0) {
			t.Errorf("Batch 0: Messages should be identical with shared inbox")
		} else {
			t.Logf("✓ Batch 0: Messages are identical (as expected with shared inbox)")
		}
	}

	// Next, we post another batch, this time with divergence.
	// We diverge at message index 5 within the evil node's batch.
	l2info.Accounts["Owner"].Nonce.Store(5)
	numMessagesPerBatch = int64(10)

	// Create both good and evil batch data
	goodBatchData2 := createBoldBatchData(t, l2info, numMessagesPerBatch, -1) // No divergence
	l2info.Accounts["Owner"].Nonce.Store(5)                                   // reset our tracking of owner nonce
	evilBatchData2 := createBoldBatchData(t, l2info, numMessagesPerBatch, 5)  // Diverge at index 5

	// Store batch in DA to get certificate
	var certificate2 []byte
	if evilStrategy == UntrustedSignerCert {
		// For UntrustedSignerCert, use a writer with untrusted signer
		daWriterUntrusted := referenceda.NewWriter(untrustedSigner, referenceda.DefaultConfig.MaxBatchSize)
		certificate2, err = daWriterUntrusted.Store(goodBatchData2, 3600).Await(ctx)
		Require(t, err)
		t.Log("Created certificate for batch 2 with untrusted signer")
	} else {
		// For other strategies, use normal writer
		certificate2, err = daWriterA.Store(goodBatchData2, 3600).Await(ctx)
		Require(t, err)
	}

	// Deserialize certificate to extract data hash
	cert2, err := referenceda.Deserialize(certificate2)
	Require(t, err)
	dataHash := common.Hash(cert2.DataHash)

	// Configure evil provider with evil data mapping BEFORE posting to L1
	if evilStrategy == EvilDataGoodCert || evilStrategy == EvilDataEvilCert {
		evilProvider.SetEvilData(dataHash, evilBatchData2)
	}

	// For EvilDataEvilCert strategy, also configure the evil enhancer
	if evilStrategy == EvilDataEvilCert && evilEnhancer != nil {
		// Create evil certificate that matches evil data
		evilCert, err := referenceda.NewCertificate(evilBatchData2, dataSigner)
		Require(t, err)

		// Configure evil enhancer to use evil certificate
		goodCertKeccak := crypto.Keccak256Hash(certificate2)
		evilEnhancer.SetMapping(goodCertKeccak, evilCert.Serialize())
	}

	// For ValidCertClaimedInvalid strategy, configure provider to lie about this specific valid cert
	if evilStrategy == ValidCertClaimedInvalid {
		certKeccak := crypto.Keccak256Hash(certificate2)
		evilProvider.SetClaimCertInvalid(certKeccak)
		t.Logf("Configured evil provider to claim certificate %s is invalid", certKeccak.Hex())
	}

	// Now post the certificate to L1 and sync to both nodes
	receipt := postBatchToL1(t, ctx, l1client, &sequencerTxOpts, seqInboxBinding, certificate2)
	syncBatchToNode(t, ctx, l1client, l2nodeA, seqInbox, receipt, "")
	syncBatchToNode(t, ctx, l1client, l2nodeB, seqInbox, receipt, "")

	// Both nodes will read the same certificate from shared sequencer inbox
	// In the case of EvilDataGoodCert and EvilDataEvilCert
	// - Node A: DA provider returns goodBatchData2
	// - Node B: Evil DA provider returns evilBatchData2
	// In the case of UntrustedSignerCert
	// - Node A: DA provider returns an error for the invalid cert
	// - Node B: DA provider returns the batch data for the invalid cert

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
		PrintSequencerInboxMessage(t, "Node A - Batch 1", msgA1)
	}

	msgB1, _, err := l2nodeB.InboxReader.GetSequencerMessageBytes(ctx, 1)
	if err != nil {
		t.Logf("Error getting batch 1 from node B: %v", err)
	}

	// Verify messages are identical with shared inbox
	if msgA1 != nil && msgB1 != nil {
		if !bytes.Equal(msgA1, msgB1) {
			t.Errorf("Batch 1: Messages should be identical with shared inbox")
		} else {
			t.Logf("✓ Batch 1: Messages are identical (as expected with shared inbox)")
		}
	}

	// Log third batch messages (batch 2 - second CustomDA batch with divergence)
	t.Logf("\n======== BATCH 2 (second CustomDA batch - WITH DIVERGENCE) ========")
	// Get and log batch 2 from both nodes
	msgA2, _, err := l2nodeA.InboxReader.GetSequencerMessageBytes(ctx, 2)
	if err != nil {
		t.Logf("Error getting batch 2 from node A: %v", err)
	} else {
		PrintSequencerInboxMessage(t, "Node A - Batch 2", msgA2)
		if evilStrategy == UntrustedSignerCert {
			t.Logf("  Note: Node A rejected this batch due to untrusted certificate signer")
		} else if evilStrategy == ValidCertClaimedInvalid {
			t.Logf("  Note: Node B rejected this batch because evil validator claimed valid cert was invalid")
		}
	}

	msgB2, _, err := l2nodeB.InboxReader.GetSequencerMessageBytes(ctx, 2)
	if err != nil {
		t.Logf("Error getting batch 2 from node B: %v", err)
	} else {
		if evilStrategy == UntrustedSignerCert {
			PrintSequencerInboxMessage(t, "Node B - Batch 2 (evil provider accepted untrusted cert)", msgB2)
		} else if evilStrategy == ValidCertClaimedInvalid {
			PrintSequencerInboxMessage(t, "Node B - Batch 2 (evil provider claimed valid cert was invalid)", msgB2)
		}
	}

	// Verify messages are identical with shared inbox
	if msgA2 != nil && msgB2 != nil {
		if !bytes.Equal(msgA2, msgB2) {
			t.Errorf("Batch 2: Messages should be identical with shared inbox")
		} else {
			t.Logf("✓ Batch 2: Messages are identical (as expected with shared inbox)")
			if evilStrategy != UntrustedSignerCert && evilStrategy != ValidCertClaimedInvalid {
				t.Logf("  Note: DA provider will return different data for same certificate!")
			}
		}
	}

	bcA, err := l2nodeA.InboxTracker.GetBatchCount()
	Require(t, err)
	bcB, err := l2nodeB.InboxTracker.GetBatchCount()
	Require(t, err)

	if bcA != bcB {
		t.Fatalf("FATAL: Expected Node A batch count %d to be equal to Node B batch count %d", bcA, bcB)
	}

	msgA, err := l2nodeA.InboxTracker.GetBatchMessageCount(bcA - 1)
	Require(t, err)
	msgB, err := l2nodeB.InboxTracker.GetBatchMessageCount(bcB - 1)
	Require(t, err)

	t.Logf("Node A batch count %d, msgs %d", bcA, msgA)
	t.Logf("Node B batch count %d, msgs %d", bcB, msgB)
	if evilStrategy == UntrustedSignerCert {
		if msgA != (msgB-10)+1 {
			// There were 10 messages in the batch with the untrusted/invalid cert.
			// Node A treats the invalid batch as if it contained a single virtual delayed message.
			t.Fatalf("FATAL: Expected Node A message count %d to be 9 less than Node B message count %d", msgA, msgB)
		}
	} else if evilStrategy == ValidCertClaimedInvalid {
		if msgB != (msgA-10)+1 {
			// There were 10 messages in the batch that the evil validator said was invalid.
			// The batch is valid for Node A, and node B treats it as if it contained a single virtual delayed message.
			t.Fatalf("FATAL: Expected Node A message count %d to be 9 more than Node B message count %d", msgA, msgB)
		}
	} else {
		if msgA != msgB {
			t.Fatalf("FATAL: Expected Node A message count %d to be equal to Node B message count %d", msgA, msgB)
		}
	}

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

		var isCaughtUp, shouldDiverge bool
		if evilStrategy == UntrustedSignerCert {
			// For UntrustedSignerCert, Node A processes fewer messages
			// Node A should have totalMessagesPosted - numMessagesPerBatch messages
			nodeAExpected := uint64(totalMessagesPosted-numMessagesPerBatch) + 1
			nodeBExpected := uint64(totalMessagesPosted)
			isCaughtUp = nodeALatest.Number.Uint64() == nodeAExpected && nodeBLatest.Number.Uint64() == nodeBExpected
			shouldDiverge = true // Nodes should always diverge in this case

		} else if evilStrategy == ValidCertClaimedInvalid {
			// For ValidCertClaimedInvalid, Node B processes fewer messages
			// Node A should have totalMessagesPosted - numMessagesPerBatch messages
			nodeAExpected := uint64(totalMessagesPosted)
			nodeBExpected := uint64(totalMessagesPosted-numMessagesPerBatch) + 1
			isCaughtUp = nodeALatest.Number.Uint64() == nodeAExpected && nodeBLatest.Number.Uint64() == nodeBExpected
			shouldDiverge = true // Nodes should always diverge in this case
		} else {
			// For other strategies, both nodes process all messages
			isCaughtUp = nodeALatest.Number.Uint64() == uint64(totalMessagesPosted)
			areEqual := nodeALatest.Number.Uint64() == nodeBLatest.Number.Uint64()
			isCaughtUp = isCaughtUp && areEqual
			shouldDiverge = true // All strategies should cause divergence
		}

		if isCaughtUp {
			t.Logf("NodeA and NodeB caught up after second batch.")
			if shouldDiverge && nodeALatest.Hash() == nodeBLatest.Hash() {
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
	// For UntrustedSignerCert and ValidCertClaimedInvalid, validator A validates one less batch
	validatorATarget := totalBatches - 1
	if evilStrategy == UntrustedSignerCert {
		validatorATarget = totalBatches - 2 // A skips the untrusted/invalid cert batch
	}

	for {
		lastInfo, err := blockValidatorA.ReadLastValidatedInfo()
		if lastInfo == nil || err != nil {
			continue
		}
		t.Log("Validator A:", lastInfo.GlobalState.Batch, "/", validatorATarget)
		if lastInfo.GlobalState.Batch >= validatorATarget {
			break
		}
		time.Sleep(time.Millisecond * 200)
	}
	for {
		lastInfo, err := blockValidatorB.ReadLastValidatedInfo()
		if lastInfo == nil || err != nil {
			continue
		}
		t.Log("Validator B:", lastInfo.GlobalState.Batch, "/", totalBatches-1)
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
					t.Fatalf("FATAL: Error in filter iterator: %v", it.Error())
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
