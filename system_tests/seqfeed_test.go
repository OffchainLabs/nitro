// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
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

	builderSeq := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builderSeq.nodeConfig.Feed.Output = *newBroadcasterConfigTest()
	cleanupSeq := builderSeq.Build(t)
	defer cleanupSeq()
	seqInfo, seqNode, seqClient := builderSeq.L2Info, builderSeq.L2.ConsensusNode, builderSeq.L2.Client

	port := seqNode.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
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
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builderSeq := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builderSeq.nodeConfig.Feed.Output = *newBroadcasterConfigTest()
	cleanupSeq := builderSeq.Build(t)
	defer cleanupSeq()
	seqInfo, seqNode, seqClient := builderSeq.L2Info, builderSeq.L2.ConsensusNode, builderSeq.L2.Client

	bigChainId, err := seqClient.ChainID(ctx)
	Require(t, err)

	config := relay.ConfigDefault
	port := seqNode.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	config.Node.Feed.Input = *newBroadcastClientConfigTest(port)
	config.Node.Feed.Output = *newBroadcasterConfigTest()
	config.Chain.ID = bigChainId.Uint64()

	feedErrChan := make(chan error, 10)
	currentRelay, err := relay.NewRelay(&config, feedErrChan)
	Require(t, err)
	err = currentRelay.Start(ctx)
	Require(t, err)
	defer currentRelay.StopAndWait()

	port = currentRelay.GetListenerAddr().(*net.TCPAddr).Port
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
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
	testClient *TestClient,
	testScenario string,
) *execution.MessageResult {
	execHeadMsgNum, err := testClient.ExecNode.HeadMessageNumber()
	Require(t, err)
	consensusMsgCount, err := testClient.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)
	if consensusMsgCount != execHeadMsgNum+1 {
		t.Fatal(
			"consensusMsgCount", consensusMsgCount, "is different than (execHeadMsgNum + 1)", execHeadMsgNum,
			"testScenario:", testScenario,
		)
	}

	var lastResult *execution.MessageResult
	for msgCount := 1; arbutil.MessageIndex(msgCount) <= consensusMsgCount; msgCount++ {
		pos := msgCount - 1
		resultExec, err := testClient.ExecNode.ResultAtPos(arbutil.MessageIndex(pos))
		Require(t, err)

		resultConsensus, err := testClient.ConsensusNode.TxStreamer.ResultAtCount(arbutil.MessageIndex(msgCount))
		Require(t, err)

		if !reflect.DeepEqual(resultExec, resultConsensus) {
			t.Fatal(
				"resultExec", resultExec, "is different than resultConsensus", resultConsensus,
				"pos:", pos,
				"testScenario:", testScenario,
			)
		}

		lastResult = resultExec
	}

	return lastResult
}

func testLyingSequencer(t *testing.T, dasModeStr string) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The truthful sequencer
	chainConfig, nodeConfigA, lifecycleManager, _, dasSignerKey := setupConfigWithDAS(t, ctx, dasModeStr)
	defer lifecycleManager.StopAndWaitUntil(time.Second)

	nodeConfigA.BatchPoster.Enable = true
	nodeConfigA.Feed.Output.Enable = false
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig = nodeConfigA
	builder.chainConfig = chainConfig
	builder.L2Info = nil
	cleanup := builder.Build(t)
	defer cleanup()

	l2clientA := builder.L2.Client

	authorizeDASKeyset(t, ctx, dasSignerKey, builder.L1Info, builder.L1.Client)

	// The lying sequencer
	nodeConfigC := arbnode.ConfigDefaultL1Test()
	nodeConfigC.BatchPoster.Enable = false
	nodeConfigC.DataAvailability = nodeConfigA.DataAvailability
	nodeConfigC.DataAvailability.RPCAggregator.Enable = false
	nodeConfigC.Feed.Output = *newBroadcasterConfigTest()
	testClientC, cleanupC := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfigC})
	defer cleanupC()
	l2clientC, nodeC := testClientC.Client, testClientC.ConsensusNode

	port := nodeC.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port

	// The client node, connects to lying sequencer's feed
	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.Feed.Output.Enable = false
	nodeConfigB.Feed.Input = *newBroadcastClientConfigTest(port)
	nodeConfigB.DataAvailability = nodeConfigA.DataAvailability
	nodeConfigB.DataAvailability.RPCAggregator.Enable = false
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

	fraudResult := compareAllMsgResultsFromConsensusAndExecution(t, testClientB, "fraud")

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

	// Consensus should update message result stored in its database after a reorg
	realResult := compareAllMsgResultsFromConsensusAndExecution(t, testClientB, "real")
	// Checks that results changed
	if reflect.DeepEqual(fraudResult, realResult) {
		t.Fatal("realResult and fraudResult are equal")
	}
}

func TestLyingSequencer(t *testing.T) {
	testLyingSequencer(t, "onchain")
}

func TestLyingSequencerLocalDAS(t *testing.T) {
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

	port := wsBroadcastServer.ListenerAddr().(*net.TCPAddr).Port

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)
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
	l1IncomingMsg, err := gethexec.MessageFromTxes(
		&l1IncomingMsgHeader,
		types.Transactions{tx},
		[]error{nil},
	)
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
