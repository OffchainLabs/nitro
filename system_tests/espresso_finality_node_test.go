package arbtest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

func createEspressoFinalityNode(t *testing.T, builder *NodeBuilder) (*TestClient, func()) {
	nodeConfig := builder.nodeConfig
	execConfig := builder.execConfig
	// Disable the batch poster because it requires redis if enabled on the 2nd node
	nodeConfig.BatchPoster.Enable = false

	nodeConfig.BlockValidator.Enable = true
	nodeConfig.BlockValidator.ValidationPoll = 2 * time.Second
	nodeConfig.BlockValidator.ValidationServer.URL = fmt.Sprintf("ws://127.0.0.1:%d", 54327)
	nodeConfig.BlockValidator.LightClientAddress = lightClientAddress
	nodeConfig.BlockValidator.Espresso = true
	nodeConfig.DelayedSequencer.Enable = true
	nodeConfig.DelayedSequencer.FinalizeDistance = 1
	nodeConfig.Sequencer = true
	nodeConfig.Dangerous.NoSequencerCoordinator = true
	execConfig.Sequencer.Enable = true
	execConfig.Sequencer.EnableEspressoFinalityNode = true
	execConfig.Sequencer.EspressoFinalityNodeConfig.Namespace = builder.chainConfig.ChainID.Uint64()
	execConfig.Sequencer.EspressoFinalityNodeConfig.StartBlock = 1
	execConfig.Sequencer.EspressoFinalityNodeConfig.HotShotUrl = hotShotUrl

	builder.nodeConfig.TransactionStreamer.SovereignSequencerEnabled = false

	return builder.Build2ndNode(t, &SecondNodeParams{
		nodeConfig: nodeConfig,
		execConfig: execConfig,
	})
}

func TestEspressoFinalityNode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	valNodeCleanup := createValidationNode(ctx, t, true)
	defer valNodeCleanup()

	builder, cleanup := createL1AndL2Node(ctx, t)
	defer cleanup()

	err := waitForL1Node(t, ctx)
	Require(t, err)

	cleanEspresso := runEspresso(t, ctx)
	defer cleanEspresso()

	// wait for the builder
	err = waitForEspressoNode(t, ctx)
	Require(t, err)

	err = checkTransferTxOnL2(t, ctx, builder.L2, "User14", builder.L2Info)
	Require(t, err)

	msgCnt, err := builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)

	err = waitForWith(t, ctx, 6*time.Minute, 5*time.Second, func() bool {
		validatedCnt := builder.L2.ConsensusNode.BlockValidator.Validated(t)
		log.Info("L2 validated count", "validatedCnt", validatedCnt, "msgCnt", msgCnt)
		return validatedCnt == msgCnt
	})
	Require(t, err)

	// start the finality node
	builderEspressoFinalityNode, cleanupEspressoFinalityNode := createEspressoFinalityNode(t, builder)
	defer cleanupEspressoFinalityNode()

	err = waitForWith(t, ctx, 6*time.Minute, 5*time.Second, func() bool {
		msgCntFinalityNode, err := builderEspressoFinalityNode.ConsensusNode.TxStreamer.GetMessageCount()
		log.Info("Finality node validated count", "msgCntFinalityNode", msgCntFinalityNode, "msgCnt", msgCnt)
		Require(t, err)
		return msgCntFinalityNode == msgCnt
	})
	Require(t, err)
}
