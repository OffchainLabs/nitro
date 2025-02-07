package arbtest

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

func createCaffNode(t *testing.T, builder *NodeBuilder) (*TestClient, func()) {
	nodeConfig := builder.nodeConfig
	execConfig := builder.execConfig

	// Disable the batch poster because it requires redis if enabled on the 2nd node
	nodeConfig.BatchPoster.Enable = false
	nodeConfig.BlockValidator.Enable = false
	nodeConfig.DelayedSequencer.Enable = false
	nodeConfig.DelayedSequencer.FinalizeDistance = 1
	nodeConfig.Sequencer = true
	nodeConfig.Dangerous.NoSequencerCoordinator = true
	execConfig.Sequencer.Enable = true
	execConfig.Sequencer.EnableCaffNode = true
	execConfig.Sequencer.CaffNodeConfig.Namespace = builder.chainConfig.ChainID.Uint64()
	execConfig.Sequencer.CaffNodeConfig.StartBlock = 1
	execConfig.Sequencer.CaffNodeConfig.HotShotUrl = hotShotUrl
	builder.nodeConfig.BlockValidator.Enable = false
	nodeConfig.ParentChainReader.Enable = false
	return builder.Build2ndNode(t, &SecondNodeParams{
		nodeConfig: nodeConfig,
		execConfig: execConfig,
	})
}

func TestEspressoCaffNode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	valNodeCleanup := createValidationNode(ctx, t, true)
	defer valNodeCleanup()

	builder, cleanup := createL1AndL2Node(ctx, t, true)
	defer cleanup()

	err := waitForL1Node(ctx)
	Require(t, err)

	cleanEspresso := runEspresso()
	defer cleanEspresso()

	// wait for the builder
	err = waitForEspressoNode(ctx)
	Require(t, err)

	err = checkTransferTxOnL2(t, ctx, builder.L2, "User14", builder.L2Info)
	Require(t, err)
	err = checkTransferTxOnL2(t, ctx, builder.L2, "User15", builder.L2Info)
	Require(t, err)

	log.Info("Starting the caff node")
	// start the node
	builderCaffNode, cleanupCaffNode := createCaffNode(t, builder)
	defer cleanupCaffNode()

	rpcClient := builderCaffNode.Client.Client()
	startTime := time.Now()
	// Wait till we have two blocks created
	for {
		var lastBlock map[string]interface{}
		err = rpcClient.CallContext(ctx, &lastBlock, "eth_getBlockByNumber", "latest", false)
		Require(t, err)
		if lastBlock == nil {
			// fail
			t.Fatal("last block is nil")
		}
		log.Info("last block", "lastBlock", lastBlock)

		number, ok := lastBlock["number"].(string)
		if !ok {
			t.Fatal("number is not a string")
		}
		if number == "0x2" {
			break
		}
		if time.Since(startTime) > 10*time.Minute {
			t.Fatal("timeout waiting for node to create blocks")
		}
		time.Sleep(time.Second * 5)
	}

}
