package arbtest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/bold/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/mel"
	melrunner "github.com/offchainlabs/nitro/arbnode/mel/runner"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/staker/bold"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestMessageExtractionLayer_SequencerBatchMessageEquivalence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithBoldDeployment().
		WithDelayBuffer(0)
	builder.L2Info.GenerateAccount("User2")
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	cleanup := builder.Build(t)
	defer cleanup()

	melState := createInitialMELState(t, ctx, builder.addresses, builder.L1.Client)

	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L1.Client)
	l1Reader, err := headerreader.New(ctx, builder.L1.Client, func() *headerreader.Config { return &headerreader.TestConfig }, arbSys)
	Require(t, err)
	l1Reader.Start(ctx)
	defer l1Reader.StopAndWait()

	// Wait for headMelState to be finalized to avoid initializing delayed message backlog
	for {
		latestFinalized, err := l1Reader.Client().BlockByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
		Require(t, err)
		if latestFinalized.NumberU64() >= melState.ParentChainBlockNumber {
			break
		}
		AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 5)
		time.Sleep(500 * time.Millisecond)
	}

	melDB := melrunner.NewDatabase(builder.L2.ConsensusNode.ArbDB)
	Require(t, melDB.SaveState(ctx, melState)) // save head mel state
	mockMsgConsumer := &mockMELDB{savedMsgs: make([]*arbostypes.MessageWithMetadata, 0)}
	extractor, err := melrunner.NewMessageExtractor(
		l1Reader.Client(),
		builder.addresses,
		melDB,
		mockMsgConsumer,
		nil, // TODO: Provide da readers here.
		melState.ParentChainBlockHash,
		0,
	)
	Require(t, err)
	extractor.StopWaiter.Start(ctx, extractor)

	// Create various L2 transactions and wait for them to be included in a batch
	// as compressed messages submitted to the sequencer inbox.
	sequencerTxOpts := builder.L1Info.GetDefaultTransactOpts("Sequencer", ctx)
	numMessages := 10
	forceSequencerMessageBatchPosting(
		t,
		builder.L2.ConsensusNode,
		builder.L2Info,
		builder.L1.Client,
		&sequencerTxOpts,
		builder.addresses.SequencerInbox,
		int64(numMessages),
	)

	// Run the extractor routine until it has caught up to the latest parent chain block.
	for {
		prevFSMState := extractor.CurrentFSMState()
		_, err = extractor.Act(ctx)
		Require(t, err)
		newFSMState := extractor.CurrentFSMState()
		// If the extractor FSM has been in the ProcessingNextBlock state twice in a row, without error, it means
		// it has caught up to the latest (or configured safe/finalized) parent chain block. We can
		// exit the loop here and assert information about MEL.
		if prevFSMState == melrunner.ProcessingNextBlock && newFSMState == melrunner.ProcessingNextBlock {
			break
		}
	}

	// // Assert details about the extraction routine.
	// if len(mockDB.savedStates) == 0 {
	// 	t.Fatal("MEL did not save any states")
	// }

	inboxTracker := builder.L2.ConsensusNode.InboxTracker
	numBatches, err := inboxTracker.GetBatchCount()
	Require(t, err)
	if numBatches != 2 {
		t.Fatalf("MEL number of batches %d does not match inbox tracker %d", 2, numBatches)
	}
	batchSequenceNum := uint64(1)
	inboxTrackerMessageCount, err := inboxTracker.GetBatchMessageCount(batchSequenceNum)
	Require(t, err)
	// #nosec G115
	if uint64(inboxTrackerMessageCount) != uint64(numMessages)+1 {
		t.Fatalf(
			"MEL batch message count %d does not match inbox tracker %d",
			inboxTrackerMessageCount,
			numMessages,
		)
	}
	lastState, err := melDB.GetHeadMelState(ctx)
	Require(t, err)
	extractedNumMessages := lastState.MsgCount
	if extractedNumMessages != uint64(inboxTrackerMessageCount) {
		t.Fatalf(
			"MEL batch message count %d does not match inbox tracker %d",
			extractedNumMessages,
			inboxTrackerMessageCount,
		)
	}
	inboxStreamer := builder.L2.ConsensusNode.TxStreamer
	msgCount, err := inboxStreamer.GetMessageCount()
	Require(t, err)
	inboxTrackerMessages := make([]*arbostypes.MessageWithMetadata, 0)
	// Start from 1 to skip the init message.
	for i := uint64(1); i < uint64(msgCount); i++ {
		msg, err := inboxStreamer.GetMessage(arbutil.MessageIndex(i))
		Require(t, err)
		inboxTrackerMessages = append(inboxTrackerMessages, msg)
	}
	melMessages := mockMsgConsumer.savedMsgs
	if len(melMessages) != len(inboxTrackerMessages) {
		t.Fatalf("MEL and inbox tracker message count do not match %d != %d", len(melMessages), len(inboxTrackerMessages))
	}

	for i, msg := range melMessages {
		fromInboxTracker := inboxTrackerMessages[i]
		if !fromInboxTracker.Message.Equals(msg.Message) {
			t.Fatal("Messages from MEL and inbox tracker do not match")
		}
	}
}

func TestMessageExtractionLayer_SequencerBatchMessageEquivalence_Blobs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	threshold := uint64(0)
	messagesPerBatch := uint64(3)

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithBoldDeployment().
		WithDelayBuffer(threshold)
	builder.L2Info.GenerateAccount("User2")
	builder.nodeConfig.BatchPoster.Post4844Blobs = true
	builder.nodeConfig.BatchPoster.IgnoreBlobPrice = true
	builder.withBlobReader = true
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	cleanup := builder.Build(t)
	defer cleanup()

	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()

	// Force a batch to be posted as a delayed message and ensure it is reflected in the onchain contracts.
	forceDelayedBatchPosting(t, ctx, builder, testClientB, messagesPerBatch, threshold)

	// Create an initial MEL state from the latest confirmed assertion.
	melState := createInitialMELState(t, ctx, builder.addresses, builder.L1.Client)

	// Construct a new MEL service and provide with an initial MEL state
	// to begin extracting messages from the parent chain.
	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L1.Client)
	l1Reader, err := headerreader.New(ctx, builder.L1.Client, func() *headerreader.Config { return &headerreader.TestConfig }, arbSys)
	Require(t, err)
	l1Reader.Start(ctx)
	defer l1Reader.StopAndWait()

	melDB := melrunner.NewDatabase(builder.L2.ConsensusNode.ArbDB)
	Require(t, melDB.SaveState(ctx, melState)) // save head mel state
	mockMsgConsumer := &mockMELDB{savedMsgs: make([]*arbostypes.MessageWithMetadata, 0)}
	extractor, err := melrunner.NewMessageExtractor(
		l1Reader.Client(),
		builder.addresses,
		melDB,
		mockMsgConsumer,
		[]daprovider.Reader{daprovider.NewReaderForBlobReader(builder.L1.blobReader)},
		melState.ParentChainBlockHash,
		0,
	)
	Require(t, err)
	extractor.StopWaiter.Start(ctx, extractor)

	// Post a blob batch with a bunch of txs
	initialBatchCount := GetBatchCount(t, builder)
	var txs types.Transactions
	for i := 0; i < 20; i++ {
		tx, _ := builder.L2.TransferBalance(t, "Faucet", "User2", big.NewInt(1e12), builder.L2Info)
		txs = append(txs, tx)
	}
	builder.nodeConfig.BatchPoster.MaxDelay = 0
	_, err = builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
	Require(t, err)
	for _, tx := range txs {
		_, err := testClientB.EnsureTxSucceeded(tx)
		Require(t, err, "tx not found on second node")
	}
	CheckBatchCount(t, builder, initialBatchCount+1)

	for {
		prevFSMState := extractor.CurrentFSMState()
		_, err = extractor.Act(ctx)
		Require(t, err)
		newFSMState := extractor.CurrentFSMState()
		// If the extractor FSM has been in the ProcessingNextBlock state twice in a row, without error, it means
		// it has caught up to the latest (or configured safe/finalized) parent chain block. We can
		// exit the loop here and assert information about MEL.
		if prevFSMState == melrunner.ProcessingNextBlock && newFSMState == melrunner.ProcessingNextBlock {
			break
		}
	}

	numDelayedMessages, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedCount()
	Require(t, err)
	lastState, err := melDB.GetHeadMelState(ctx)
	Require(t, err)
	inboxStreamer := builder.L2.ConsensusNode.TxStreamer
	trackerMsgCount, err := inboxStreamer.GetMessageCount()
	Require(t, err)
	if lastState.MsgCount != uint64(trackerMsgCount) {
		t.Fatalf(
			"MEL batch message count %d does not match inbox tracker %d",
			lastState.MsgCount,
			trackerMsgCount,
		)
	}

	// Check that MEL extracted the same number of sequencer batch messages the inbox tracker has seen.
	// Start from 1 to skip the init message.
	inboxTrackerMessages := make([]*arbostypes.MessageWithMetadata, 0)
	for i := uint64(1); i < uint64(trackerMsgCount); i++ {
		msg, err := inboxStreamer.GetMessage(arbutil.MessageIndex(i))
		Require(t, err)
		inboxTrackerMessages = append(inboxTrackerMessages, msg)
	}
	melMessages := mockMsgConsumer.savedMsgs
	if len(melMessages) != len(inboxTrackerMessages) {
		t.Fatalf("MEL and inbox tracker message count do not match %d != %d", len(melMessages), len(inboxTrackerMessages))
	}
	for i, msg := range melMessages {
		fromInboxTracker := inboxTrackerMessages[i]
		if !fromInboxTracker.Message.Equals(msg.Message) {
			t.Fatal("Messages from MEL and inbox tracker do not match")
		}
	}

	// Check that MEL extracted the same number of delayed messages the inbox tracker has seen.
	if lastState.DelayedMessagedSeen != numDelayedMessages {
		t.Fatalf(
			"MEL delayed message count %d does not match inbox tracker %d",
			lastState.DelayedMessagedSeen,
			numDelayedMessages,
		)
	}
	// Start from 1 to ignore the init message.
	readHelperState := &mel.State{DelayedMessagedSeen: 1}
	readHelperState.SetDelayedMessageBacklog(&mel.DelayedMessageBacklog{})
	readHelperState.SetReadCountFromBacklog(numDelayedMessages) // skip checking against accumulator- not the purpose of this test
	for i := uint64(1); i < numDelayedMessages; i++ {
		fromInboxTracker, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedMessage(ctx, i)
		Require(t, err)
		Require(t, readHelperState.AccumulateDelayedMessage(&mel.DelayedInboxMessage{Message: fromInboxTracker}))
		readHelperState.DelayedMessagedSeen++
		fromMelDB, err := melDB.ReadDelayedMessage(ctx, readHelperState, i)
		Require(t, err)
		// Check the messages we extracted from MEL and the inbox tracker are the same.
		if !fromInboxTracker.Equals(fromMelDB.Message) {
			t.Fatal("Messages from MEL and inbox tracker do not match")
		}
	}

}

func TestMessageExtractionLayer_DelayedMessageEquivalence_Simple(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	threshold := uint64(0)
	messagesPerBatch := uint64(3)

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithBoldDeployment().
		WithDelayBuffer(threshold)
	builder.L2Info.GenerateAccount("User2")
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	cleanup := builder.Build(t)
	defer cleanup()

	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()

	// Force a batch to be posted as a delayed message and ensure it is reflected in the onchain contracts.
	forceDelayedBatchPosting(t, ctx, builder, testClientB, messagesPerBatch, threshold)

	// Create an initial MEL state from the latest confirmed assertion.
	melState := createInitialMELState(t, ctx, builder.addresses, builder.L1.Client)

	// Construct a new MEL service and provide with an initial MEL state
	// to begin extracting messages from the parent chain.
	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L1.Client)
	l1Reader, err := headerreader.New(ctx, builder.L1.Client, func() *headerreader.Config { return &headerreader.TestConfig }, arbSys)
	Require(t, err)
	l1Reader.Start(ctx)
	defer l1Reader.StopAndWait()

	melDB := melrunner.NewDatabase(builder.L2.ConsensusNode.ArbDB)
	Require(t, melDB.SaveState(ctx, melState)) // save head mel state
	mockMsgConsumer := &mockMELDB{savedMsgs: make([]*arbostypes.MessageWithMetadata, 0)}
	extractor, err := melrunner.NewMessageExtractor(
		l1Reader.Client(),
		builder.addresses,
		melDB,
		mockMsgConsumer,
		nil, // TODO: Provide da readers here.
		melState.ParentChainBlockHash,
		0,
	)
	Require(t, err)
	extractor.StopWaiter.Start(ctx, extractor)

	for {
		prevFSMState := extractor.CurrentFSMState()
		_, err = extractor.Act(ctx)
		Require(t, err)
		newFSMState := extractor.CurrentFSMState()
		// If the extractor FSM has been in the ProcessingNextBlock state twice in a row, without error, it means
		// it has caught up to the latest (or configured safe/finalized) parent chain block. We can
		// exit the loop here and assert information about MEL.
		if prevFSMState == melrunner.ProcessingNextBlock && newFSMState == melrunner.ProcessingNextBlock {
			break
		}
	}

	numDelayedMessages, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedCount()
	Require(t, err)

	lastState, err := melDB.GetHeadMelState(ctx)
	Require(t, err)

	// Check that MEL extracted the same number of delayed messages the inbox tracker has seen.
	if lastState.DelayedMessagedSeen != numDelayedMessages {
		t.Fatalf(
			"MEL delayed message count %d does not match inbox tracker %d",
			lastState.DelayedMessagedSeen,
			numDelayedMessages,
		)
	}

	// Start from 1 to ignore the init message.
	readHelperState := &mel.State{DelayedMessagedSeen: 1}
	readHelperState.SetDelayedMessageBacklog(&mel.DelayedMessageBacklog{})
	readHelperState.SetReadCountFromBacklog(numDelayedMessages) // skip checking against accumulator- not the purpose of this test
	for i := uint64(1); i < numDelayedMessages; i++ {
		fromInboxTracker, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedMessage(ctx, i)
		Require(t, err)
		Require(t, readHelperState.AccumulateDelayedMessage(&mel.DelayedInboxMessage{Message: fromInboxTracker}))
		readHelperState.DelayedMessagedSeen++
		fromMelDB, err := melDB.ReadDelayedMessage(ctx, readHelperState, i)
		Require(t, err)
		// Check the messages we extracted from MEL and the inbox tracker are the same.
		if !fromInboxTracker.Equals(fromMelDB.Message) {
			t.Fatal("Messages from MEL and inbox tracker do not match")
		}
	}

	// Small reorg of 4 mel states
	reorgToBlockNum := lastState.ParentChainBlockNumber - 4
	reorgToState, err := melDB.State(ctx, reorgToBlockNum)
	Require(t, err)
	reorgToBlockHash := reorgToState.ParentChainBlockHash
	reorgToBlock, err := builder.L1.Client.BlockByHash(ctx, reorgToBlockHash)
	Require(t, err)
	Require(t, builder.L1.L1Backend.BlockChain().ReorgToOldBlock(reorgToBlock))

	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 6)
	// Check if ReorgingToOldBlock fsm state works as intended
	for {
		prevFSMState := extractor.CurrentFSMState()
		_, err = extractor.Act(ctx)
		if err != nil {
			t.Fatal(err)
		}
		newFSMState := extractor.CurrentFSMState()
		// After reorg rewinding is done in the SavingMessages step, break
		if prevFSMState == melrunner.SavingMessages && newFSMState == melrunner.ProcessingNextBlock {
			break
		}
	}

	lastState, err = melDB.GetHeadMelState(ctx)
	Require(t, err)
	if lastState.ParentChainBlockNumber != reorgToBlockNum+1 {
		t.Fatalf("Unexpected number of MEL states after a parent chain reorg. Want: %d, Have: %d", reorgToBlockNum+1, lastState.ParentChainBlockNumber)
	}
}

func TestMessageExtractionLayer_RunningNode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	threshold := uint64(0)
	messagesPerBatch := uint64(3)

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithBoldDeployment().
		WithDelayBuffer(threshold)

	builder.nodeConfig.MessageExtraction.Enable = true
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")

	nodeConfig2 := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfig2.MessageExtraction.Enable = true
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfig2})
	defer cleanupB()

	// Force a batch to be posted as a delayed message and ensure it is reflected in the onchain contracts.
	forceDelayedBatchPosting(t, ctx, builder, testClientB, messagesPerBatch, threshold)

	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	Require(t, err)
	delayedBridge, err := arbnode.NewDelayedBridge(builder.L1.Client, builder.L1Info.GetAddress("Bridge"), 0)
	Require(t, err)
	lookupL2Tx := getLookupL2Tx(t, ctx, delayedBridge)

	// Test eth deposit
	builder.L1Info.GenerateAccount("UserX")
	builder.L1.TransferBalance(t, "Faucet", "UserX", big.NewInt(1e18), builder.L1Info)

	txOpts := builder.L1Info.GetDefaultTransactOpts("UserX", ctx)
	txOpts.Value = big.NewInt(13)
	oldBalanceClientB, err := testClientB.Client.BalanceAt(ctx, txOpts.From, nil)
	if err != nil {
		t.Fatalf("BalanceAt(%v) unexpected error: %v", txOpts.From, err)
	}

	// Verify that ethDeposit works as intended on the sequence node's side
	l2Receipt := testDepositETH(t, ctx, builder, delayedInbox, lookupL2Tx, txOpts)
	forceDelayedBatchPosting(t, ctx, builder, testClientB, messagesPerBatch, threshold) // We need to post a batch so that clientB can pick up the deposit tx, since it itself cannot execute a delayed message!

	// Wait for deposit to be seen at clientB
	l2ReceiptClientB, err := WaitForTx(ctx, testClientB.Client, l2Receipt.TxHash, time.Second*5)
	Require(t, err)

	newBalance, err := testClientB.Client.BalanceAt(ctx, txOpts.From, l2ReceiptClientB.BlockNumber)
	if err != nil {
		t.Fatalf("BalanceAt(%v) unexpected error: %v", txOpts.From, err)
	}
	if got := new(big.Int); got.Sub(newBalance, oldBalanceClientB).Cmp(txOpts.Value) != 0 {
		t.Errorf("Got transferred: %v, want: %v", got, txOpts.Value)
	}
}

func TestMessageExtractionLayer_TxStreamerHandleReorg(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LvlInfo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	threshold := uint64(0)
	messagesPerBatch := uint64(3)

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithBoldDeployment().
		WithDelayBuffer(threshold)

	builder.nodeConfig.MessageExtraction.Enable = true
	builder.nodeConfig.MessageExtraction.RetryInterval = 100 * time.Millisecond
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")

	nodeConfig2 := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfig2.MessageExtraction.Enable = true
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfig2})
	defer cleanupB()
	forceDelayedBatchPosting(t, ctx, builder, testClientB, messagesPerBatch, threshold)

	// Test plan:
	// 		* Send a delayed message in L1 by making a eth deposit
	// 		* Reorg L1 to a block before the eth deposit was made
	// 		* Advance L1 to the previous head block number at the least so that MEL detects reorg
	// 		* We verify that MEL detected reorg
	// 		* Geth would still include the eth deposit tx as a block in the new chain
	// 		* Post a batch with L2 txs- this would include delayed message read corresponding to the index containing
	// 		  eth deposit tx- as that delayed message was sequenced
	// 		* MEL will add the right delayed message at the corresponding index and send those txs to txStreamer
	// 		* TxStreamer would detect a reorg as the previous delayed message's bytes wont match the new one's
	// 		* We verify that TxStreamer detected reorg
	// 		* Later we verify that the balance is as expected since the eth deposit tx should be successful

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

	// Reorg L1 and advance it so that MEl can pick up the reorg
	currHead, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)
	Require(t, builder.L1.L1Backend.BlockChain().ReorgToOldBlock(reorgToBlock))
	// #nosec G115
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, int(currHead-reorgToBlock.NumberU64()+5)) // we need to advance L1 blocks up until the current head so that reorg is detected

	// Wait until mel can detect reorg and rewind head state
	time.Sleep(5 * time.Second)

	// Post a batch so that mel can send up-to-date L2 messages to txStreamer
	initialBatchCount := GetBatchCount(t, builder)
	var txs types.Transactions
	for i := 0; i < 10; i++ {
		tx, _ := builder.L2.TransferBalance(t, "Faucet", "User2", big.NewInt(1e12), builder.L2Info)
		txs = append(txs, tx)
	}
	builder.nodeConfig.BatchPoster.MaxDelay = 0
	_, err = builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
	Require(t, err)
	for _, tx := range txs {
		_, err := testClientB.EnsureTxSucceeded(tx)
		Require(t, err, "tx not found on second node")
	}
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
}

func TestMessageExtractionLayer_UseArbDBForStoringDelayedMessages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	threshold := uint64(0)
	messagesPerBatch := uint64(3)

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithBoldDeployment().
		WithDelayBuffer(threshold)
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")

	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()

	// Force a batch to be posted as a delayed message and ensure it is reflected in the onchain contracts.
	forceDelayedBatchPosting(t, ctx, builder, testClientB, messagesPerBatch, threshold)

	// Create an initial MEL state from the latest confirmed assertion.
	melState := createInitialMELState(t, ctx, builder.addresses, builder.L1.Client)

	// Construct a new MEL service and provide with an initial MEL state
	// to begin extracting messages from the parent chain.
	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L1.Client)
	l1Reader, err := headerreader.New(ctx, builder.L1.Client, func() *headerreader.Config { return &headerreader.TestConfig }, arbSys)
	Require(t, err)
	l1Reader.Start(ctx)
	defer l1Reader.StopAndWait()

	melDB := melrunner.NewDatabase(builder.L2.ConsensusNode.ArbDB)
	Require(t, melDB.SaveState(ctx, melState)) // save head mel state
	// TODO: tx streamer to be used here when ready to run the node using mel thus replacing inbox reader-tracker code
	mockMsgConsumer := &mockMELDB{savedMsgs: make([]*arbostypes.MessageWithMetadata, 0)}
	extractor, err := melrunner.NewMessageExtractor(
		l1Reader.Client(),
		builder.addresses,
		melDB,
		mockMsgConsumer,
		nil, // TODO: Provide da readers here.
		melState.ParentChainBlockHash,
		0,
	)
	Require(t, err)
	extractor.StopWaiter.Start(ctx, extractor)

	for {
		prevFSMState := extractor.CurrentFSMState()
		_, err = extractor.Act(ctx)
		Require(t, err)
		newFSMState := extractor.CurrentFSMState()
		// If the extractor FSM has been in the ProcessingNextBlock state twice in a row, without error, it means
		// it has caught up to the latest (or configured safe/finalized) parent chain block. We can
		// exit the loop here and assert information about MEL.
		if prevFSMState == melrunner.ProcessingNextBlock && newFSMState == melrunner.ProcessingNextBlock {
			break
		}
	}

	headMelStateBlockNum, err := melDB.GetHeadMelStateBlockNum()
	Require(t, err)
	if headMelStateBlockNum == melState.ParentChainBlockNumber {
		t.Fatal("MEL did not save any states")
	}

	numDelayedMessages, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedCount()
	Require(t, err)

	lastState, err := melDB.State(ctx, headMelStateBlockNum)
	Require(t, err)

	// Check that MEL extracted the same number of delayed messages the inbox tracker has seen.
	if lastState.DelayedMessagedSeen != numDelayedMessages {
		t.Fatalf(
			"MEL delayed message count %d does not match inbox tracker %d",
			lastState.DelayedMessagedSeen,
			numDelayedMessages,
		)
	}

	newInitialState, err := melDB.FetchInitialState(ctx, lastState.ParentChainBlockHash)
	Require(t, err)
	delayedMessageBacklog := mel.NewDelayedMessageBacklog(ctx, 100, extractor.GetFinalizedDelayedMessagesRead)
	err = melrunner.InitializeDelayedMessageBacklog(ctx, delayedMessageBacklog, melDB, newInitialState, extractor.GetFinalizedDelayedMessagesRead)
	Require(t, err)
	newInitialState.SetDelayedMessageBacklog(delayedMessageBacklog)
	newInitialState.SetReadCountFromBacklog(newInitialState.DelayedMessagedSeen) // skip checking against accumulator- not the purpose of this test
	for i := newInitialState.DelayedMessagesRead; i < newInitialState.DelayedMessagedSeen; i++ {
		// Validates the pending unread delayed messages via accumulator
		delayedMsgSavedByMel, err := melDB.ReadDelayedMessage(ctx, newInitialState, newInitialState.DelayedMessagesRead)
		Require(t, err)
		fetchedDelayedMsg, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedMessage(ctx, i)
		Require(t, err)
		if !fetchedDelayedMsg.Equals(delayedMsgSavedByMel.Message) {
			t.Fatal("Messages from MEL and inbox tracker do not match")
		}
		t.Logf("validated delayed message of index: %d", i)
	}
}

type mockMELDB struct {
	savedMsgs        []*arbostypes.MessageWithMetadata
	savedDelayedMsgs []*mel.DelayedInboxMessage
	savedStates      map[uint64]*mel.State
	lastState        *mel.State
}

func (m *mockMELDB) PushMessages(ctx context.Context, firstMsgIdx uint64, messages []*arbostypes.MessageWithMetadata) error {
	m.savedMsgs = append(m.savedMsgs, messages...)
	return nil
}

func (m *mockMELDB) State(
	ctx context.Context,
	parentChainBlockNumber uint64,
) (*mel.State, error) {
	state, ok := m.savedStates[parentChainBlockNumber]
	if !ok {
		return nil, errors.New("state not found")
	}
	return state, nil
}

func (m *mockMELDB) SaveState(
	ctx context.Context,
	state *mel.State,
) error {
	m.savedStates[state.ParentChainBlockNumber] = state
	m.lastState = state
	return nil
}

func (m *mockMELDB) FetchInitialState(
	ctx context.Context, parentChainBlockHash common.Hash, _ uint64,
) (*mel.State, error) {
	if m.lastState.ParentChainBlockHash != parentChainBlockHash {
		return nil, fmt.Errorf("parentChainBlockHash of db doesnt match the hash queried by initialStateFetcher")
	}
	return m.savedStates[m.lastState.ParentChainBlockNumber], nil
}

func (m *mockMELDB) SaveDelayedMessages(
	ctx context.Context,
	state *mel.State,
	delayedMessages []*mel.DelayedInboxMessage,
) error {
	m.savedDelayedMsgs = append(m.savedDelayedMsgs, delayedMessages...)
	return nil
}
func (m *mockMELDB) ReadDelayedMessage(
	ctx context.Context,
	_ *mel.State,
	index uint64,
) (*mel.DelayedInboxMessage, error) {
	if index == 0 {
		return nil, errors.New("index cannot be 0")
	}
	// Ignore the init message, as we do not store it in this mock DB.
	index = index - 1
	if index >= uint64(len(m.savedDelayedMsgs)) {
		return nil, errors.New("index out of bounds")
	}
	return m.savedDelayedMsgs[index], nil
}

func createInitialMELState(
	t *testing.T,
	ctx context.Context,
	addrs *chaininfo.RollupAddresses,
	client *ethclient.Client,
) *mel.State {
	// Create an initial MEL state from the latest confirmed assertion.
	rollup, err := rollupgen.NewRollupUserLogic(addrs.Rollup, client)
	Require(t, err)
	confirmedHash, err := rollup.LatestConfirmed(&bind.CallOpts{})
	Require(t, err)
	latestConfirmedAssertion, err := bold.ReadBoldAssertionCreationInfo(
		ctx,
		rollup,
		client,
		addrs.Rollup,
		confirmedHash,
	)
	Require(t, err)
	startBlock, err := client.BlockByNumber(ctx, new(big.Int).SetUint64(latestConfirmedAssertion.CreationL1Block))
	Require(t, err)
	chainId, err := client.ChainID(ctx)
	Require(t, err)

	// TODO: Construct the correct MEL state from the latest confirmed assertion.
	return &mel.State{
		Version:                            0,
		BatchPostingTargetAddress:          addrs.SequencerInbox,
		DelayedMessagePostingTargetAddress: addrs.Bridge,
		ParentChainId:                      chainId.Uint64(),
		ParentChainBlockNumber:             startBlock.NumberU64(),
		ParentChainBlockHash:               startBlock.Hash(),
		ParentChainPreviousBlockHash:       startBlock.ParentHash(),
		MessageAccumulator:                 common.Hash{},
		DelayedMessagedSeen:                1,
		DelayedMessagesRead:                1, // Assumes we have read the init message.
		MsgCount:                           1,
	}

}

func forceDelayedBatchPosting(
	t *testing.T,
	ctx context.Context,
	builder *NodeBuilder,
	testClientB *TestClient,
	messagesPerBatch uint64,
	threshold uint64,
) {
	// Advance L1 to force a batch given the delay buffer threshold
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, int(threshold)) // #nosec G115

	initialBatchCount := GetBatchCount(t, builder)
	txs := make(types.Transactions, messagesPerBatch)
	for i := range txs {
		txs[i] = builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, common.Big1, nil)
	}

	// Send txs to the L1 inbox.
	SendSignedTxesInBatchViaL1(t, ctx, builder.L1Info, builder.L1.Client, builder.L2.Client, txs)

	// Advance L1 to force a batch given the delay buffer threshold
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, int(threshold)) // #nosec G115

	builder.nodeConfig.BatchPoster.MaxDelay = 0
	_, err := builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
	Require(t, err)
	for _, tx := range txs {
		_, err := testClientB.EnsureTxSucceeded(tx)
		Require(t, err, "tx not found on second node")
	}

	CheckBatchCount(t, builder, initialBatchCount+1)
	// Reset the max delay.
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour
}

func forceSequencerMessageBatchPosting(
	t *testing.T,
	l2Node *arbnode.Node,
	l2Info *BlockchainTestInfo,
	backend *ethclient.Client,
	sequencer *bind.TransactOpts,
	seqInboxAddr common.Address,
	numMessages int64,
) {
	ctx := context.Background()
	seqInbox, err := bridgegen.NewSequencerInbox(seqInboxAddr, backend)
	Require(t, err)
	batchBuffer := bytes.NewBuffer([]byte{})
	for i := int64(0); i < numMessages; i++ {
		value := i
		err := writeTxToBatch(batchBuffer, l2Info.PrepareTx("Owner", "User2", l2Info.TransferGas, big.NewInt(value), []byte{}))
		Require(t, err)
	}
	compressed, err := arbcompress.CompressWell(batchBuffer.Bytes())
	Require(t, err)
	message := append([]byte{0}, compressed...)

	seqNum := new(big.Int).Lsh(common.Big1, 256)
	seqNum.Sub(seqNum, common.Big1)
	tx, err := seqInbox.AddSequencerL2BatchFromOrigin8f111f3c(sequencer, seqNum, message, big.NewInt(1), common.Address{}, big.NewInt(0), big.NewInt(0))
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)

	nodeSeqInbox, err := arbnode.NewSequencerInbox(backend, seqInboxAddr, 0)
	Require(t, err)
	batches, err := nodeSeqInbox.LookupBatchesInRange(ctx, receipt.BlockNumber, receipt.BlockNumber)
	Require(t, err)
	if len(batches) == 0 {
		Fatal(t, "batch not found after AddSequencerL2BatchFromOrigin")
	}
	err = l2Node.InboxTracker.AddSequencerBatches(ctx, backend, batches)
	Require(t, err)
}
