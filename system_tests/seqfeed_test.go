// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/db/schema"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/broadcastclients"
	"github.com/offchainlabs/nitro/broadcaster/backlog"
	"github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/relay"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

func newBroadcasterConfigTest() *wsbroadcastserver.BroadcasterConfig {
	config := wsbroadcastserver.DefaultTestBroadcasterConfig
	config.Enable = true
	config.Port = "0"
	return &config
}

func newBroadcastClientConfigTest(port int) *broadcastclient.Config {
	return &broadcastclient.Config{
		URL:     []string{fmt.Sprintf("ws://localhost:%d/feed", port)},
		Timeout: 200 * time.Millisecond,
		Verify: signature.VerifierConfig{
			Dangerous: signature.DangerousVerifierConfig{
				AcceptMissing: true,
			},
		},
	}
}

func TestSequencerFeed(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LvlTrace)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builderSeq := NewNodeBuilder(ctx).DefaultConfig(t, false).DontParalellise()
	builderSeq.nodeConfig.Feed.Output = *newBroadcasterConfigTest()
	cleanupSeq := builderSeq.Build(t)
	defer cleanupSeq()
	seqInfo, seqNode, seqClient := builderSeq.L2Info, builderSeq.L2.ConsensusNode, builderSeq.L2.Client

	port := testhelpers.AddrTCPPort(seqNode.BroadcastServer.ListenerAddr(), t)
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).DontParalellise()
	builder.nodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)
	builder.takeOwnership = false
	cleanup := builder.Build(t)
	defer cleanup()
	client := builder.L2.Client

	seqInfo.GenerateAccount("User2")

	tx := seqInfo.PrepareTx("Owner", "User2", seqInfo.TransferGas, big.NewInt(1e12), nil)

	err := seqClient.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = builderSeq.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	_, err = WaitForTx(ctx, client, tx.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := client.BalanceAt(ctx, seqInfo.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}

	if logHandler.WasLogged(arbnode.BlockHashMismatchLogMsg) {
		t.Fatal("BlockHashMismatchLogMsg was logged unexpectedly")
	}
}

func TestRelayedSequencerFeed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builderSeq := NewNodeBuilder(ctx).DefaultConfig(t, false).DontParalellise()
	builderSeq.nodeConfig.Feed.Output = *newBroadcasterConfigTest()
	cleanupSeq := builderSeq.Build(t)
	defer cleanupSeq()
	seqInfo, seqNode, seqClient := builderSeq.L2Info, builderSeq.L2.ConsensusNode, builderSeq.L2.Client

	bigChainId, err := seqClient.ChainID(ctx)
	Require(t, err)

	config := relay.ConfigDefault
	port := testhelpers.AddrTCPPort(seqNode.BroadcastServer.ListenerAddr(), t)
	config.Node.Feed.Input = *newBroadcastClientConfigTest(port)
	config.Node.Feed.Output = *newBroadcasterConfigTest()
	config.Chain.ID = bigChainId.Uint64()

	feedErrChan := make(chan error, 10)
	currentRelay, err := relay.NewRelay(&config, feedErrChan)
	Require(t, err)
	err = currentRelay.Start(ctx)
	Require(t, err)
	defer currentRelay.StopAndWait()

	port = testhelpers.AddrTCPPort(currentRelay.GetListenerAddr(), t)
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).DontParalellise()
	builder.nodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)
	builder.takeOwnership = false
	cleanup := builder.Build(t)
	defer cleanup()
	node, client := builder.L2.ConsensusNode, builder.L2.Client
	StartWatchChanErr(t, ctx, feedErrChan, node)

	seqInfo.GenerateAccount("User2")

	tx := seqInfo.PrepareTx("Owner", "User2", seqInfo.TransferGas, big.NewInt(1e12), nil)

	err = seqClient.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = builderSeq.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	_, err = WaitForTx(ctx, client, tx.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := client.BalanceAt(ctx, seqInfo.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
}

func compareAllMsgResultsFromConsensusAndExecution(
	t *testing.T,
	ctx context.Context,
	testClient *TestClient,
	testScenario string,
) *execution.MessageResult {
	execHeadMsgIdx, err := testClient.ExecNode.HeadMessageIndex().Await(context.Background())
	Require(t, err)
	consensusHeadMsgIdx, err := testClient.ConsensusNode.TxStreamer.GetHeadMessageIndex()
	Require(t, err)
	if consensusHeadMsgIdx != execHeadMsgIdx {
		t.Fatal(
			"consensusHeadMsgIdx", consensusHeadMsgIdx, "is different than execHeadMsgIdx", execHeadMsgIdx,
			"testScenario:", testScenario,
		)
	}

	var lastResult *execution.MessageResult
	for msgIdx := arbutil.MessageIndex(0); msgIdx <= consensusHeadMsgIdx; msgIdx++ {
		resultExec, err := testClient.ExecNode.ResultAtMessageIndex(arbutil.MessageIndex(msgIdx)).Await(ctx)
		Require(t, err)

		resultConsensus, err := testClient.ConsensusNode.TxStreamer.ResultAtMessageIndex(arbutil.MessageIndex(msgIdx))
		Require(t, err)

		if !reflect.DeepEqual(resultExec, resultConsensus) {
			t.Fatal(
				"resultExec", resultExec, "is different than resultConsensus", resultConsensus,
				"msgIdx:", msgIdx,
				"testScenario:", testScenario,
			)
		}

		lastResult = resultExec
	}

	return lastResult
}

func testLyingSequencer(t *testing.T, daModeStr string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The truthful sequencer
	chainConfig, nodeConfigA, lifecycleManager, _, anyTrustSignerKey := setupConfigWithAnyTrust(t, ctx, daModeStr)
	defer lifecycleManager.StopAndWaitUntil(time.Second)

	nodeConfigA.BatchPoster.Enable = true
	nodeConfigA.Feed.Output.Enable = false
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).DontParalellise().WithTakeOwnership(false)
	builder.nodeConfig = nodeConfigA
	builder.chainConfig = chainConfig
	builder.L2Info = nil
	cleanup := builder.Build(t)
	defer cleanup()

	l2clientA := builder.L2.Client

	authorizeAnyTrustKeyset(t, ctx, anyTrustSignerKey, builder.L1Info, builder.L1.Client)

	// The lying sequencer
	nodeConfigC := arbnode.ConfigDefaultL1Test()
	nodeConfigC.BatchPoster.Enable = false
	nodeConfigC.DA.AnyTrust = nodeConfigA.DA.AnyTrust
	nodeConfigC.DA.AnyTrust.RPCAggregator.Enable = false
	nodeConfigC.Feed.Output = *newBroadcasterConfigTest()
	testClientC, cleanupC := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfigC})
	defer cleanupC()
	l2clientC, nodeC := testClientC.Client, testClientC.ConsensusNode

	port := testhelpers.AddrTCPPort(nodeC.BroadcastServer.ListenerAddr(), t)

	// The client node, connects to lying sequencer's feed
	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.Feed.Output.Enable = false
	nodeConfigB.Feed.Input = *newBroadcastClientConfigTest(port)
	nodeConfigB.DA.AnyTrust = nodeConfigA.DA.AnyTrust
	nodeConfigB.DA.AnyTrust.RPCAggregator.Enable = false
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfigB})
	defer cleanupB()
	l2clientB := testClientB.Client

	builder.L2Info.GenerateAccount("FraudUser")
	builder.L2Info.GenerateAccount("RealUser")

	fraudTx := builder.L2Info.PrepareTx("Owner", "FraudUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	builder.L2Info.GetInfoWithPrivKey("Owner").Nonce.Add(^uint64(0)) // Use same l2info object for different l2s
	realTx := builder.L2Info.PrepareTx("Owner", "RealUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)

	for i := 0; i < 10; i++ {
		err := l2clientC.SendTransaction(ctx, fraudTx)
		if err == nil {
			break
		}
		<-time.After(time.Millisecond * 10)
		if i == 9 {
			t.Fatal("error sending fraud transaction:", err)
		}
	}

	_, err := testClientC.EnsureTxSucceeded(fraudTx)
	if err != nil {
		t.Fatal("error ensuring fraud transaction succeeded:", err)
	}

	// Node B should get the transaction immediately from the sequencer feed
	_, err = WaitForTx(ctx, l2clientB, fraudTx.Hash(), time.Second*15)
	if err != nil {
		t.Fatal("error waiting for tx:", err)
	}
	l2balance, err := l2clientB.BalanceAt(ctx, builder.L2Info.GetAddress("FraudUser"), nil)
	if err != nil {
		t.Fatal("error getting balance:", err)
	}
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}

	fraudResult := compareAllMsgResultsFromConsensusAndExecution(t, ctx, testClientB, "fraud")

	// Send the real transaction to client A, will cause a reorg on nodeB
	err = l2clientA.SendTransaction(ctx, realTx)
	if err != nil {
		t.Fatal("error sending real transaction:", err)
	}

	_, err = builder.L2.EnsureTxSucceeded(realTx)
	if err != nil {
		t.Fatal("error ensuring real transaction succeeded:", err)
	}

	// Node B should get the transaction after NodeC posts a batch.
	_, err = WaitForTx(ctx, l2clientB, realTx.Hash(), time.Second*5)
	if err != nil {
		t.Fatal("error waiting for transaction to get to node b:", err)
	}
	l2balanceFraudAcct, err := l2clientB.BalanceAt(ctx, builder.L2Info.GetAddress("FraudUser"), nil)
	if err != nil {
		t.Fatal("error getting fraud balance:", err)
	}
	if l2balanceFraudAcct.Cmp(big.NewInt(0)) != 0 {
		t.Fatal("Unexpected balance (fraud acct should be empty) was:", l2balanceFraudAcct)
	}

	l2balanceRealAcct, err := l2clientB.BalanceAt(ctx, builder.L2Info.GetAddress("RealUser"), nil)
	if err != nil {
		t.Fatal("error getting real balance:", err)
	}
	if l2balanceRealAcct.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance of real account:", l2balanceRealAcct)
	}

	// Since NodeB is not a sequencer, it will produce blocks through Consensus.
	// So it is expected that Consensus.ResultAtMessageIndex will not rely on Execution to retrieve results.
	// However, since msgIdx 0 is related to genesis, and Execution is initialized through InitializeArbosInDatabase and not through Consensus,
	// first call to Consensus.ResultAtMessageIndex with msgIdx equals to 0 will fall back to Execution.
	// Not necessarily the first call to Consensus.ResultAtMessageIndex with msgIdx equals to 0 will happen through compareMsgResultFromConsensusAndExecution,
	// so we don't test this here.
	consensusHeadMsgIdx, err := testClientB.ConsensusNode.TxStreamer.GetHeadMessageIndex()
	Require(t, err)
	if consensusHeadMsgIdx != 1 {
		t.Fatal("consensusHeadMsgIdx is different than 1")
	}
	logHandler := testhelpers.InitTestLog(t, log.LvlTrace)
	_, err = testClientB.ConsensusNode.TxStreamer.ResultAtMessageIndex(arbutil.MessageIndex(1))
	Require(t, err)
	if logHandler.WasLogged(arbnode.FailedToGetMsgResultFromDB) {
		t.Fatal("Consensus relied on execution database to return the result")
	}
	// Consensus should update message result stored in its database after a reorg
	realResult := compareAllMsgResultsFromConsensusAndExecution(t, ctx, testClientB, "real")
	// Checks that results changed
	if reflect.DeepEqual(fraudResult, realResult) {
		t.Fatal("realResult and fraudResult are equal")
	}
}

func TestLyingSequencer(t *testing.T) {
	testLyingSequencer(t, "onchain")
}

func TestLyingSequencerLocalAnyTrust(t *testing.T) {
	testLyingSequencer(t, "files")
}

func testBlockHashComparison(t *testing.T, blockHash *common.Hash, mustMismatch bool) {
	logHandler := testhelpers.InitTestLog(t, log.LvlTrace)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	backlogConfiFetcher := func() *backlog.Config {
		return &backlog.DefaultTestConfig
	}
	bklg := backlog.NewBacklog(backlogConfiFetcher)

	wsBroadcastServer := wsbroadcastserver.NewWSBroadcastServer(
		newBroadcasterConfigTest,
		bklg,
		412346,
		nil,
	)
	err := wsBroadcastServer.Initialize()
	if err != nil {
		t.Fatal("error initializing wsBroadcastServer:", err)
	}
	err = wsBroadcastServer.Start(ctx)
	if err != nil {
		t.Fatal("error starting wsBroadcastServer:", err)
	}
	defer wsBroadcastServer.StopAndWait()

	port := testhelpers.AddrTCPPort(wsBroadcastServer.ListenerAddr(), t)

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).DontParalellise().WithTakeOwnership(false)
	builder.nodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)
	// Default shuts the node down on mismatch; opt out so the tx-processing
	// assertions below still mean something.
	builder.nodeConfig.TransactionStreamer.ShutdownOnBlockhashMismatch = false
	cleanup := builder.Build(t)
	defer cleanup()
	testClient := builder.L2

	userAccount := "User2"
	builder.L2Info.GenerateAccount(userAccount)
	tx := builder.L2Info.PrepareTx("Owner", userAccount, builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	l1IncomingMsgHeader := arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		Poster:      l1pricing.BatchPosterAddress,
		BlockNumber: 29,
		Timestamp:   1715295980,
		RequestId:   nil,
		L1BaseFee:   nil,
	}
	l1IncomingMsg, err := gethexec.MessageFromTxes(&l1IncomingMsgHeader, []gethexec.TxResult{{Tx: tx}})
	Require(t, err)

	broadcastMessage := message.BroadcastMessage{
		Version: 1,
		Messages: []*message.BroadcastFeedMessage{
			{
				SequenceNumber: 1,
				Message: arbostypes.MessageWithMetadata{
					Message:             l1IncomingMsg,
					DelayedMessagesRead: 1,
				},
				BlockHash: blockHash,
			},
		},
	}
	wsBroadcastServer.Broadcast(&broadcastMessage)

	// For now, even though block hash mismatch, the transaction should still be processed
	_, err = WaitForTx(ctx, testClient.Client, tx.Hash(), time.Second*15)
	if err != nil {
		t.Fatal("error waiting for tx:", err)
	}
	l2balance, err := testClient.Client.BalanceAt(ctx, builder.L2Info.GetAddress(userAccount), nil)
	if err != nil {
		t.Fatal("error getting balance:", err)
	}
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}

	mismatched := logHandler.WasLogged(arbnode.BlockHashMismatchLogMsg)
	if mustMismatch && !mismatched {
		t.Fatal("Failed to log BlockHashMismatchLogMsg")
	} else if !mustMismatch && mismatched {
		t.Fatal("BlockHashMismatchLogMsg was logged unexpectedly")
	}
}

func TestBlockHashFeedMismatch(t *testing.T) {
	blockHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	testBlockHashComparison(t, &blockHash, true)
}

func TestBlockHashFeedNil(t *testing.T) {
	testBlockHashComparison(t, nil, false)
}

func TestPopulateFeedBacklog(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LvlTrace)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).DontParalellise().WithDatabase(rawdb.DBPebble)
	builder.BuildL1(t)

	userAccount := "User2"
	builder.L2Info.GenerateAccount(userAccount)

	// Guarantees that nodes will rely only on the feed to receive messages
	builder.nodeConfig.BatchPoster.Enable = false
	builder.BuildL2OnL1(t)

	dataDir := builder.l2StackConfig.DataDir

	// Sends a transaction
	tx := builder.L2Info.PrepareTx("Owner", userAccount, builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Shutdown node and starts a new one with same data dir and output feed enabled.
	// The new node will populate the feedbacklog since already has a message, related to the
	// transaction previously sent, stored in disk.
	builder.L2.cleanup()
	builder.l2StackConfig.DataDir = dataDir
	builder.nodeConfig.Feed.Output = *newBroadcasterConfigTest()
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	// Creates a sink node that will read from the output feed of the previous node.
	nodeConfigSink := builder.nodeConfig
	port := testhelpers.AddrTCPPort(builder.L2.ConsensusNode.BroadcastServer.ListenerAddr(), t)
	nodeConfigSink.Feed.Input = *newBroadcastClientConfigTest(port)
	testClientSink, cleanupSink := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfigSink})
	defer cleanupSink()

	// Waits for the transaction to be processed by the sink node.
	_, err = WaitForTx(ctx, testClientSink.Client, tx.Hash(), time.Second*5)
	if err != nil {
		t.Fatal("error waiting for transaction to get to sink:", err)
	}
	balance, err := testClientSink.Client.BalanceAt(ctx, builder.L2Info.GetAddress(userAccount), nil)
	if err != nil {
		t.Fatal("error getting fraud balance:", err)
	}
	if balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", balance)
	}

	if logHandler.WasLogged(arbnode.BlockHashMismatchLogMsg) {
		t.Fatal("BlockHashMismatchLogMsg was logged unexpectedly")
	}
}

func TestRegressionInPopulateFeedBacklog(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.BuildL1(t)

	userAccount := "User2"
	builder.L2Info.GenerateAccount(userAccount)

	// Guarantees that nodes will rely only on the feed to receive messages
	builder.nodeConfig.BatchPoster.Enable = false
	builder.BuildL2OnL1(t)

	// Sends a transaction
	tx := builder.L2Info.PrepareTx("Owner", userAccount, builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Create dummy batch posting report data
	data, err := createDummyBatchPostingReportTransaction()
	Require(t, err)

	// sub in correct batch hash
	batchData, _, err := builder.L2.ConsensusNode.GetParentChainDataSource().GetSequencerMessageBytes(ctx, 0)
	Require(t, err)
	expectedBatchHash := crypto.Keccak256Hash(batchData)
	copy(data[52:52+32], expectedBatchHash[:])

	dummyMessage := arbostypes.MessageWithMetadata{
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind:        arbostypes.L1MessageType_BatchPostingReport,
				Poster:      l1pricing.BatchPosterAddress,
				BlockNumber: 0,
				Timestamp:   0,
			},
			L2msg: data,
		},
		DelayedMessagesRead: 0,
	}

	// Override last index to be a batch posting report
	messageCount, err := builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
	if err != nil {
		panic(fmt.Sprintf("error getting tx streamer message count: %v", err))
	}
	key := dbKey(schema.MessagePrefix, uint64(messageCount-1))
	msgBytes, err := rlp.EncodeToBytes(dummyMessage)
	if err != nil {
		panic(fmt.Sprintf("error encoding dummy message: %v", err))
	}
	batch := builder.L2.ConsensusNode.ConsensusDB.NewBatch()
	if err := batch.Put(key, msgBytes); err != nil {
		panic(fmt.Sprintf("error putting dummy message to db: %v", err))
	}
	err = batch.Write()
	if err != nil {
		panic(fmt.Sprintf("error writing batch to db: %v", err))
	}

	// Shutdown node and starts a new one with same data dir and output feed enabled.
	// The new node will populate the feedbacklog since already has a message, related to the
	// transaction previously sent, stored in disk.
	builder.L2.cleanup()
	dataDir := builder.l2StackConfig.DataDir
	builder.l2StackConfig.DataDir = dataDir
	builder.nodeConfig.Feed.Output = *newBroadcasterConfigTest()
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()
}

func createDummyBatchPostingReportTransaction() ([]byte, error) {
	batchTimestamp := new(big.Int)
	batchTimestamp.SetUint64(0)
	batchPosterAddr := common.Address{}
	batchNum := uint64(0)
	batchGas := uint64(0)
	l1BaseFee := new(big.Int)
	l1BaseFee.SetUint64(0)

	return util.PackInternalTxDataBatchPostingReport(
		batchTimestamp, batchPosterAddr, batchNum, batchGas, l1BaseFee,
	)
}

// restBackfillArchive is a minimal stand-in for arb-relay's REST backlog API, serving the chunks it
// is given.
type restBackfillArchive struct {
	server    *httptest.Server
	chainId   uint64
	chunkSize uint64

	mu     sync.Mutex
	chunks map[uint64][]*message.BroadcastFeedMessage
}

func newRestBackfillArchive(t *testing.T, chainId, chunkSize uint64) *restBackfillArchive {
	t.Helper()
	a := &restBackfillArchive{
		chainId:   chainId,
		chunkSize: chunkSize,
		chunks:    make(map[uint64][]*message.BroadcastFeedMessage),
	}
	a.server = httptest.NewServer(http.HandlerFunc(a.serve))
	t.Cleanup(a.server.Close)
	return a
}

func (a *restBackfillArchive) publish(start uint64, messages []*message.BroadcastFeedMessage) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.chunks[start] = messages
}

func (a *restBackfillArchive) serve(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/feed/v1/info" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"chainId":%d,"feedVersion":1,"chunkSize":%d}`, a.chainId, a.chunkSize)
		return
	}
	rawStart, ok := strings.CutPrefix(r.URL.Path, "/feed/v1/chunk/")
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	start, err := strconv.ParseUint(rawStart, 10, 64)
	if err != nil || start%a.chunkSize != 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	a.mu.Lock()
	messages, published := a.chunks[start]
	a.mu.Unlock()
	if !published {
		// Not settled yet, as far as this client can tell.
		w.Header().Set("Cache-Control", "public, max-age=1")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	body, err := json.Marshal(message.BroadcastMessage{Version: 1, Messages: messages})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Encoding", "gzip")
	gz := gzip.NewWriter(w)
	defer gz.Close()
	_, _ = gz.Write(body)
}

// TestSequencerFeedRestBackfill covers the case the feature exists for: the feed serves only a
// short catchup window, and the node fills everything below it from the REST backlog.
func TestSequencerFeedRestBackfill(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LvlTrace)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const chainId = uint64(412346)
	const chunkSize = 4

	backlogConfigFetcher := func() *backlog.Config { return &backlog.DefaultTestConfig }
	wsBroadcastServer := wsbroadcastserver.NewWSBroadcastServer(
		newBroadcasterConfigTest,
		backlog.NewBacklog(backlogConfigFetcher),
		chainId,
		nil,
	)
	Require(t, wsBroadcastServer.Initialize())
	Require(t, wsBroadcastServer.Start(ctx))
	defer wsBroadcastServer.StopAndWait()
	port := testhelpers.AddrTCPPort(wsBroadcastServer.ListenerAddr(), t)

	archive := newRestBackfillArchive(t, chainId, chunkSize)

	// No parent chain: this node only ever follows the feed, which also starts its feed clients
	// immediately rather than waiting to catch up from L1.
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).DontParalellise().WithTakeOwnership(false)
	builder.nodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)
	builder.nodeConfig.Feed.Input.Rest = broadcastclient.RestConfig{
		Enable:  true,
		URL:     archive.server.URL,
		Timeout: 5 * time.Second,
	}
	// A backfilled message below carries a wrong block hash; the node must log it, not stop.
	builder.nodeConfig.TransactionStreamer.ShutdownOnBlockhashMismatch = false
	cleanup := builder.Build(t)
	defer cleanup()
	testClient := builder.L2

	userAccount := "User2"
	builder.L2Info.GenerateAccount(userAccount)

	// One transfer per feed message, indexed by sequence number; index 0 is the init message the
	// node already has.
	const transfer = int64(1e12)
	const lastMsgIdx = 2 * chunkSize
	txs := make([]*types.Transaction, lastMsgIdx+1)
	feedMessages := make([]*message.BroadcastFeedMessage, lastMsgIdx+1)
	for msgIdx := 1; msgIdx <= lastMsgIdx; msgIdx++ {
		tx := builder.L2Info.PrepareTx("Owner", userAccount, builder.L2Info.TransferGas, big.NewInt(transfer), nil)
		header := arbostypes.L1IncomingMessageHeader{
			Kind:        arbostypes.L1MessageType_L2Message,
			Poster:      l1pricing.BatchPosterAddress,
			BlockNumber: 29,
			Timestamp:   1715295980,
		}
		l1IncomingMsg, err := gethexec.MessageFromTxes(&header, []gethexec.TxResult{{Tx: tx}})
		Require(t, err)
		txs[msgIdx] = tx
		feedMessages[msgIdx] = &message.BroadcastFeedMessage{
			SequenceNumber: arbutil.MessageIndex(msgIdx), // #nosec G115
			Message: arbostypes.MessageWithMetadata{
				Message:             l1IncomingMsg,
				DelayedMessagesRead: 1,
			},
		}
	}

	// The rest of the first chunk arrives over the feed as usual, leaving the node's head at the
	// start of the second.
	for _, msg := range feedMessages[1:chunkSize] {
		wsBroadcastServer.Broadcast(&message.BroadcastMessage{
			Version:  1,
			Messages: []*message.BroadcastFeedMessage{msg},
		})
	}
	_, err := WaitForTx(ctx, testClient.Client, txs[chunkSize-1].Hash(), time.Second*15)
	Require(t, err)

	// The second chunk is only in the REST backlog, as it would be once the feed has moved on. Its
	// first message carries a wrong block hash, so backfilled messages must reach the same block
	// hash check as live ones.
	wrongHash := common.Hash{1}
	feedMessages[chunkSize].BlockHash = &wrongHash
	archive.publish(chunkSize, feedMessages[chunkSize:2*chunkSize])

	// The message after it over the feed opens the gap the backfill has to close.
	wsBroadcastServer.Broadcast(&message.BroadcastMessage{
		Version:  1,
		Messages: []*message.BroadcastFeedMessage{feedMessages[lastMsgIdx]},
	})

	// The last message can only execute once everything below it has been backfilled.
	_, err = WaitForTx(ctx, testClient.Client, txs[lastMsgIdx].Hash(), time.Second*30)
	Require(t, err)

	balance, err := testClient.Client.BalanceAt(ctx, builder.L2Info.GetAddress(userAccount), nil)
	Require(t, err)
	if expected := big.NewInt(transfer * lastMsgIdx); balance.Cmp(expected) != 0 {
		t.Fatal("Unexpected balance:", balance, "expected:", expected)
	}

	if !logHandler.WasLogged(broadcastclients.FeedGapLogMsg) {
		t.Fatal("the feed gap was not detected")
	}
	if !logHandler.WasLogged(arbnode.BlockHashMismatchLogMsg) {
		t.Fatal("the wrong block hash on a backfilled message was not checked")
	}
}
