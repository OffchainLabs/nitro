package arbtest

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

var extraFeedInputPort = "54220"

func TestEspressoBatchPosterShouldNotReorg(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	extraFeedInput := fmt.Sprintf("ws://localhost:%s", extraFeedInputPort)
	builder, cleanup := createL1AndL2Node(ctx, t, false, extraFeedInput)
	defer cleanup()

	err := waitForL1Node(ctx)
	Require(t, err)

	cleanEspresso := runEspresso()
	defer cleanEspresso()

	err = waitForEspressoNode(ctx)
	Require(t, err)

	config := wsbroadcastserver.DefaultBroadcasterConfig
	config.Enable = true
	config.Port = "54220"
	feedErrChan := make(chan error, 10)
	b := broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &config }, builder.chainConfig.ChainID.Uint64(), feedErrChan, nil)
	Require(t, b.Initialize())
	Require(t, b.Start(ctx))

	l2Node := builder.L2
	err = checkTransferTxOnL2(t, ctx, l2Node, "test1", builder.L2Info)
	Require(t, err)
	err = checkTransferTxOnL2(t, ctx, l2Node, "test2", builder.L2Info)
	Require(t, err)

	// Include the initial message
	expected := arbutil.MessageIndex(3)
	err = waitForWith(ctx, 5*time.Minute, 10*time.Second, func() bool {
		msgCnt, err := l2Node.ConsensusNode.TxStreamer.GetMessageCount()
		if err != nil {
			panic(err)
		}

		validatedCnt := l2Node.ConsensusNode.BlockValidator.Validated(t)
		return msgCnt >= expected && validatedCnt >= expected
	})
	Require(t, err)
	msg1, err := builder.L2.ConsensusNode.TxStreamer.GetMessageWithMetadataAndBlockHash(t, arbutil.MessageIndex(1))
	Require(t, err)

	msg2, err := builder.L2.ConsensusNode.TxStreamer.GetMessageWithMetadataAndBlockHash(t, arbutil.MessageIndex(2))
	Require(t, err)

	if b.GetCachedMessageCount() > 0 {
		t.Fatal("should be 0 before sending any messages")
	}
	// Broadcast the message 2 with the position 1
	b.BroadcastSingle(msg2.MessageWithMeta, arbutil.MessageIndex(1), msg2.BlockHash)

	err = waitFor(ctx, func() bool {
		return b.GetCachedMessageCount() > 0
	})
	Require(t, err)

	msgNew, err := builder.L2.ConsensusNode.TxStreamer.GetMessageWithMetadataAndBlockHash(t, arbutil.MessageIndex(1))
	Require(t, err)

	if !bytes.Equal(msg1.BlockHash[:], msgNew.BlockHash[:]) {
		log.Error("block hash should not be modified", "oldMsg1", msg1.BlockHash, "newMsg1", msgNew.BlockHash, "msg2", msg2.BlockHash)
		t.Fatal("block hash should not be modified")
	}
}
