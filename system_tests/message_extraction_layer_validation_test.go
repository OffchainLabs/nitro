package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestValidationPostMEL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.L2Info.GenerateAccount("User2")
	builder.nodeConfig.BatchPoster.Post4844Blobs = true
	builder.nodeConfig.BatchPoster.IgnoreBlobPrice = true
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	builder.nodeConfig.MELValidator.Enable = true
	builder.nodeConfig.BlockValidator.Enable = true
	builder.nodeConfig.BlockValidator.EnableMEL = true
	builder.nodeConfig.BlockValidator.ForwardBlocks = 0
	cleanup := builder.Build(t)
	defer cleanup()

	// Post a blob batch with a bunch of txs
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()
	initialBatchCount := GetBatchCount(t, builder)
	var txs types.Transactions
	for range 20 {
		tx, _ := builder.L2.TransferBalance(t, "Faucet", "User2", big.NewInt(1e12), builder.L2Info)
		txs = append(txs, tx)
	}
	builder.nodeConfig.BatchPoster.MaxDelay = 0
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	_, err := builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
	Require(t, err)
	for _, tx := range txs {
		_, err := testClientB.EnsureTxSucceeded(tx)
		Require(t, err, "tx not found on second node")
	}
	CheckBatchCount(t, builder, initialBatchCount+1)

	// Post delayed messages
	forceDelayedBatchPosting(t, ctx, builder, testClientB, 10, 0)

	extractedMsgCountToValidate, err := builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 40)

	timeout := getDeadlineTimeout(t, time.Minute*10)
	if !builder.L2.ConsensusNode.BlockValidator.WaitForPos(t, ctx, extractedMsgCountToValidate-1, timeout) {
		Fatal(t, "did not validate all blocks")
	}
}

func TestValidationPostMELReorgHandle(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LvlInfo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.MessageExtraction.Enable = true
	builder.nodeConfig.MessageExtraction.RetryInterval = 100 * time.Millisecond
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	// Enable MEL validation
	builder.nodeConfig.MELValidator.Enable = true
	builder.nodeConfig.BlockValidator.Enable = true
	builder.nodeConfig.BlockValidator.EnableMEL = true
	builder.nodeConfig.BlockValidator.ForwardBlocks = 0
	builder.nodeConfig.BlockValidator.ClearMsgPreimagesPoll = time.Hour // Don't auto clear validated msg preimages cache
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")

	nodeConfig2 := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfig2.MessageExtraction.Enable = true
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfig2})
	defer cleanupB()
	forceDelayedBatchPosting(t, ctx, builder, testClientB, 10, 0)

	// Test plan:
	// 		* Send a delayed message in L1 by making a eth deposit
	// 		* Reorg L1 to a block before the eth deposit was made
	//      * Validate messages extracted by MEL up until now
	// 		* Advance L1 to the previous head block number at the least so that MEL detects reorg
	// 		* We verify that MEL detected reorg
	//      * Verify that MEL validator received the reorg event and reset its latestValidatedParentChainBlock
	// 		* Geth would still include the eth deposit tx as a block in the new chain
	// 		* Post a batch with L2 txs- this would include delayed message read corresponding to the index containing
	// 		  eth deposit tx- as that delayed message was sequenced
	// 		* MEL will add the right delayed message at the corresponding index and send those txs to txStreamer
	// 		* TxStreamer would detect a reorg as the previous delayed message's bytes wont match the new one's
	// 		* We verify that TxStreamer detected reorg
	// 		* Later we verify that the balance is as expected since the eth deposit tx should be successful
	//      * Verify that all the messages are validated along with their extraction

	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	Require(t, err)
	delayedBridge, err := arbnode.NewDelayedBridge(builder.L1.Client, builder.L1Info.GetAddress("Bridge"), 0)
	Require(t, err)
	lookupL2Tx := getLookupL2Tx(t, ctx, delayedBridge)

	builder.L1Info.GenerateAccount("UserX")
	builder.L1.TransferBalance(t, "Faucet", "UserX", big.NewInt(1e18), builder.L1Info)
	txOpts := builder.L1Info.GetDefaultTransactOpts("UserX", ctx)
	txOpts.Value = big.NewInt(13)
	oldBalance, err := builder.L2.Client.BalanceAt(ctx, txOpts.From, nil)
	if err != nil {
		t.Fatalf("BalanceAt(%v) unexpected error: %v", txOpts.From, err)
	}

	// Find latest L1 block, so that we can later reorg to it
	reorgToBlock, err := builder.L1.Client.BlockByNumber(ctx, nil)
	Require(t, err)

	// Verify that ethDeposit works as intended on the sequence node's side
	testDepositETH(t, ctx, builder, delayedInbox, lookupL2Tx, txOpts) // this also checks if balance increment is seen on L2

	// Validate blocks and message extraction to this point
	headState, err := builder.L2.ConsensusNode.MessageExtractor.GetHeadState()
	Require(t, err)
	extractedMsgCountToValidate := headState.MsgCount
	timeout := getDeadlineTimeout(t, time.Minute*10)
	if !builder.L2.ConsensusNode.BlockValidator.WaitForPos(t, ctx, arbutil.MessageIndex(extractedMsgCountToValidate-1), timeout) {
		Fatal(t, "did not validate all blocks")
	}

	// Check that MEL validator's LatestValidatedMELState is the state that extracted headState.MsgCount
	// and that msgPreimages and relevant MEL states for upto headState.MsgCount index are available
	want, err := builder.L2.ConsensusNode.MessageExtractor.FindMessageOriginMELState(arbutil.MessageIndex(headState.MsgCount - 1))
	Require(t, err)
	have, err := builder.L2.ConsensusNode.MELValidator.LatestValidatedMELState(ctx)
	Require(t, err)
	if have.Hash() != want.Hash() {
		t.Fatal("MELValidator LatestValidatedMELState hash mismatch")
	}
	for i := uint64(1); i < headState.MsgCount; i++ {
		// All message preimages should've been found
		_, err := builder.L2.ConsensusNode.MELValidator.FetchMsgPreimagesAndRelevantState(ctx, arbutil.MessageIndex(i))
		Require(t, err)
	}

	// Reorg L1 and advance it so that MEl can pick up the reorg
	currHead, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)
	Require(t, builder.L1.L1Backend.BlockChain().ReorgToOldBlock(reorgToBlock))
	// #nosec G115
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, int(currHead-reorgToBlock.NumberU64()+5)) // we need to advance L1 blocks up until the current head so that reorg is detected

	// Wait until mel can detect reorg and rewind head state
	time.Sleep(5 * time.Second)

	// Check that MEL validator received reorg event and updated its latestValidatedParentChainBlock
	if !logHandler.WasLogged("MEL Validator: receieved a reorg event from message extractor") {
		t.Fatal("reorg event was not forwarded to MEL validator")
	}
	newLatestValidated, err := builder.L2.ConsensusNode.MELValidator.LatestValidatedMELState(ctx)
	Require(t, err)
	if newLatestValidated.ParentChainBlockNumber != reorgToBlock.NumberU64()-1 {
		t.Fatalf("MELValidator latestValidatedParentChainBlock mismatch, have: %d, want: %d", newLatestValidated.ParentChainBlockNumber, reorgToBlock.NumberU64()-1)
	}

	// Post a batch so that mel can send up-to-date L2 messages to txStreamer
	initialBatchCount := GetBatchCount(t, builder)
	builder.nodeConfig.BatchPoster.MaxDelay = 0
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	for range 10 {
		builder.L2.TransferBalance(t, "Faucet", "User2", big.NewInt(1e12), builder.L2Info)
	}
	_, err = builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
	Require(t, err)
	time.Sleep(2 * time.Second)
	CheckBatchCount(t, builder, initialBatchCount+1)

	// Wait until mel can read the posted batch, send correct L2 messages to txStreamer and txStreamer is able to detect the Reorg and handle correct execution of L2 messages
	time.Sleep(time.Second)

	newBalance, err := builder.L2.Client.BalanceAt(ctx, txOpts.From, nil)
	if err != nil {
		t.Fatalf("BalanceAt(%v) unexpected error: %v", txOpts.From, err)
	}
	if got := new(big.Int); got.Sub(newBalance, oldBalance).Cmp(txOpts.Value) != 0 {
		t.Errorf("Got transferred: %v, want: %v", got, txOpts.Value)
	}

	// Verify that both MEL and TxStreamer detected the reorg
	if !logHandler.WasLogged("TransactionStreamer: Reorg detected!") {
		t.Fatal("reorg was not detected by TransactionStreamer")
	}
	if !logHandler.WasLogged("MEL detected L1 reorg") {
		t.Fatal("reorg was not detected by MEL")
	}

	// Check that block and MEL validators successfully validate all the blocks
	headState, err = builder.L2.ConsensusNode.MessageExtractor.GetHeadState()
	Require(t, err)
	extractedMsgCountToValidate = headState.MsgCount
	if !builder.L2.ConsensusNode.BlockValidator.WaitForPos(t, ctx, arbutil.MessageIndex(extractedMsgCountToValidate-1), timeout) {
		Fatal(t, "did not validate all blocks")
	}
}
