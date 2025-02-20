package arbtest

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

func createCaffNode(ctx context.Context, t *testing.T, existing *NodeBuilder) (*TestClient, func()) {
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
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
	execConfig.Sequencer.CaffNodeConfig.NextHotshotBlock = 1
	execConfig.Sequencer.CaffNodeConfig.ParentChainNodeUrl = "http://0.0.0.0:8545"
	execConfig.Sequencer.CaffNodeConfig.EspressoTEEVerifierAddr = existing.L1Info.GetAddress("EspressoTEEVerifierMock")
	execConfig.Sequencer.CaffNodeConfig.SequencerUrl = fmt.Sprintf("http://localhost:%d", existing.l2StackConfig.HTTPPort)
	execConfig.Sequencer.CaffNodeConfig.ParentChainReader.Enable = true
	execConfig.Sequencer.CaffNodeConfig.ParentChainReader.UseFinalityData = true
	// for testing, we can use the same hotshot url for both
	execConfig.Sequencer.CaffNodeConfig.HotShotUrls = []string{hotShotUrl, hotShotUrl, hotShotUrl, hotShotUrl}
	execConfig.Sequencer.CaffNodeConfig.RetryTime = time.Second * 1
	execConfig.Sequencer.CaffNodeConfig.HotshotPollingInterval = time.Millisecond * 100
	nodeConfig.ParentChainReader.Enable = false

	cleanup := builder.BuildEspressoCaffNode(t)
	return builder.L2, cleanup
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

	newAccount := "User16"
	l2Info := builder.L2Info
	l2Info.GenerateAccount(newAccount)
	addr := l2Info.GetAddress(newAccount)

	// Transfer via the delayed inbox
	delayedTx := l2Info.PrepareTx("Owner", newAccount, 3e7, transferAmount, nil)
	builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
		WrapL2ForDelayed(t, delayedTx, builder.L1Info, "Faucet", 100000),
	})

	err = waitForWith(ctx, 240*time.Second, 10*time.Second, func() bool {
		balance := builder.L2.GetBalance(t, addr)
		log.Info("waiting for balance", "account", newAccount, "addr", addr, "balance", balance)
		return balance.Cmp(transferAmount) >= 0
	})
	Require(t, err)

	log.Info("Starting the caff node")
	// start the node
	builderCaffNode, cleanupCaffNode := createCaffNode(ctx, t, builder)
	defer cleanupCaffNode()

	err = waitForWith(ctx, 10*time.Minute, 10*time.Second, func() bool {
		balance1 := builderCaffNode.GetBalance(t, builder.L2Info.GetAddress("User14"))
		balance2 := builderCaffNode.GetBalance(t, builder.L2Info.GetAddress("User15"))
		return balance1.Cmp(transferAmount) > 0 && balance2.Cmp(transferAmount) > 0
	})
	Require(t, err)

	err = waitForWith(ctx, 240*time.Second, 10*time.Second, func() bool {
		balance := builderCaffNode.GetBalance(t, addr)
		log.Info("waiting for balance", "account", newAccount, "addr", addr, "balance", balance)
		return balance.Cmp(transferAmount) >= 0
	})
	Require(t, err)

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
		numberString, ok := lastBlock["number"].(string)
		if !ok {
			t.Fatal("number is not a string")
		}
		// convert number to uint
		number, err := strconv.ParseInt(numberString, 0, 64)
		Require(t, err)
		if number >= 3 {
			break
		}
		if time.Since(startTime) > 10*time.Minute {
			t.Fatal("timeout waiting for node to create blocks")
		}
		time.Sleep(time.Second * 5)
	}

	// Send transaction to CaffNode and it should works later
	err = checkTransferTxOnL2(t, ctx, builderCaffNode, "User17", builder.L2Info)
	Require(t, err)
}
