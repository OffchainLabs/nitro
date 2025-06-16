// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster/backlog"
	"github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

const L2MessageKindBatch = 3

// generateL2Message creates an L1IncomingMessage containing a batch of L2 transactions.
func generateL2Message(t *testing.T, builder *NodeBuilder, txCount int, blockNum uint64) *arbostypes.L1IncomingMessage {
	var txs types.Transactions
	for i := 0; i < txCount; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User1", builder.L2Info.TransferGas, nil, nil)
		txs = append(txs, tx)
	}

	encodedTxs, err := rlp.EncodeToBytes(txs)
	Require(t, err)

	l2Msg := append([]byte{L2MessageKindBatch}, encodedTxs...)

	l1IncomingMsg := &arbostypes.L1IncomingMessage{
		Header: &arbostypes.L1IncomingMessageHeader{
			Kind:   arbostypes.L1MessageType_L2Message,
			Poster: l1pricing.BatchPosterAddress,
			// #nosec G115
			Timestamp:   uint64(time.Now().Unix()),
			BlockNumber: blockNum,
		},
		L2msg: l2Msg,
	}
	return l1IncomingMsg
}

// TestMaliciousSequencerFeed verifies that a node's TransactionStreamer
// correctly rejects oversized messages from a malicious feed.
func TestMaliciousSequencerFeed(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LvlInfo)
	_ = logHandler

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wsCfg := newBroadcasterConfigTest()
	backlogCfg := func() *backlog.Config { return &backlog.DefaultTestConfig }
	bklg := backlog.NewBacklog(backlogCfg)

	maliciousFeed := wsbroadcastserver.NewWSBroadcastServer(func() *wsbroadcastserver.BroadcasterConfig { return wsCfg }, bklg, 412346, nil)
	if err := maliciousFeed.Initialize(); err != nil {
		t.Fatal("error initializing malicious feed:", err)
	}
	if err := maliciousFeed.Start(ctx); err != nil {
		t.Fatal("error starting malicious feed:", err)
	}
	defer maliciousFeed.StopAndWait()

	port := testhelpers.AddrTCPPort(maliciousFeed.ListenerAddr(), t)
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	clientConfig := newBroadcastClientConfigTest(port)

	builder.nodeConfig.Feed.Input = *clientConfig
	builder.nodeConfig.BatchPoster.Enable = false
	builder.takeOwnership = false
	cleanup := builder.Build(t)
	defer cleanup()

	time.Sleep(400 * time.Millisecond)

	txStreamer := builder.L2.ConsensusNode.TxStreamer
	initialMsgCount, err := txStreamer.GetMessageCount()
	Require(t, err)

	builder.L2Info.GenerateAccount("User1")

	validL1IncomingMsg := generateL2Message(t, builder, 1, 1)
	validMsg := &message.BroadcastFeedMessage{
		SequenceNumber: arbutil.MessageIndex(initialMsgCount),
		Message: arbostypes.MessageWithMetadata{
			DelayedMessagesRead: 1,
			Message:             validL1IncomingMsg,
		},
	}
	maliciousFeed.Broadcast(&message.BroadcastMessage{Messages: []*message.BroadcastFeedMessage{validMsg}})

	time.Sleep(400 * time.Millisecond)

	msgCountAfterValid, err := txStreamer.GetMessageCount()
	Require(t, err)
	if msgCountAfterValid <= initialMsgCount {
		t.Fatalf("Valid message was rejected. Message count %d did not increase", msgCountAfterValid)
	}

	oversizedL1IncomingMsg := generateL2Message(t, builder, 3500, 2)
	oversizedMsg := &message.BroadcastFeedMessage{
		SequenceNumber: arbutil.MessageIndex(msgCountAfterValid),
		Message: arbostypes.MessageWithMetadata{
			DelayedMessagesRead: 1,
			Message:             oversizedL1IncomingMsg,
		},
	}

	if len(oversizedL1IncomingMsg.L2msg) <= arbostypes.MaxL2MessageSize {
		t.Fatalf("test logic error: generated message is not oversized, it has size %d", len(oversizedL1IncomingMsg.L2msg))
	}

	maliciousFeed.Broadcast(&message.BroadcastMessage{Messages: []*message.BroadcastFeedMessage{oversizedMsg}})

	// Give time to process and log the error (L2 message is too large)
	time.Sleep(1 * time.Second)

	if !logHandler.WasLogged("L2 message is too large") {
		t.Fatalf("Oversized message was rejected, but not with the expected 'L2 message is too large' error")
	}

	msgCountAfterOversized, err := txStreamer.GetMessageCount()
	Require(t, err)
	if msgCountAfterOversized > msgCountAfterValid {
		t.Fatalf("Oversized message was incorrectly accepted. Message count increased from %d to %d", msgCountAfterValid, msgCountAfterOversized)
	}
}
