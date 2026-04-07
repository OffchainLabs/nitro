package arbtest

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbnode/mel/runner"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker/bold"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestMessageExtractionLayer_SequencerBatchMessageEquivalence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithDelayBuffer(0).
		WithTakeOwnership(false)
	builder.L2Info.GenerateAccount("User2")
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	builder.nodeConfig.MessageExtraction.Enable = false
	cleanup := builder.Build(t)
	defer cleanup()

	melState := createInitialMELState(t, ctx, builder.addresses, builder.L1.Client)

	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L1.Client)
	l1Reader, err := headerreader.New(ctx, builder.L1.Client, func() *headerreader.Config { return &headerreader.TestConfig }, arbSys)
	Require(t, err)
	l1Reader.Start(ctx)
	defer l1Reader.StopAndWait()

	// Wait for headMelState to be finalized
	for {
		latestFinalized, err := l1Reader.Client().BlockByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
		Require(t, err)
		if latestFinalized.NumberU64() >= melState.ParentChainBlockNumber {
			break
		}
		AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 5)
		time.Sleep(500 * time.Millisecond)
	}

	melDB, err := melrunner.NewDatabase(builder.L2.ConsensusNode.ConsensusDB)
	Require(t, err)
	Require(t, melDB.SaveState(melState)) // save head mel state
	mockMsgConsumer := &mockMELDB{savedMsgs: make([]*arbostypes.MessageWithMetadata, 0)}
	reorgEventChan := make(chan uint64, 1)
	extractor, err := melrunner.NewMessageExtractor(
		melrunner.DefaultMessageExtractionConfig,
		l1Reader.Client(),
		builder.chainConfig,
		builder.addresses,
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil, // TODO: SequencerInbox interface needed.
		l1Reader,
		reorgEventChan,
	)
	Require(t, err)
	Require(t, extractor.SetMessageConsumer(mockMsgConsumer))
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
	lastState, err := melDB.GetHeadMelState()
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
		WithDelayBuffer(threshold)
	builder.L2Info.GenerateAccount("User2")
	builder.nodeConfig.BatchPoster.Post4844Blobs = true
	builder.nodeConfig.BatchPoster.IgnoreBlobPrice = true
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	builder.nodeConfig.MessageExtraction.Enable = false
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

	melDB, err := melrunner.NewDatabase(builder.L2.ConsensusNode.ConsensusDB)
	Require(t, err)
	Require(t, melDB.SaveState(melState)) // save head mel state
	mockMsgConsumer := &mockMELDB{savedMsgs: make([]*arbostypes.MessageWithMetadata, 0)}
	blobReaderRegistry := daprovider.NewDAProviderRegistry()
	Require(t, blobReaderRegistry.SetupBlobReader(daprovider.NewReaderForBlobReader(builder.L1.L1BlobReader)))
	reorgEventChan := make(chan uint64, 1)
	extractor, err := melrunner.NewMessageExtractor(
		melrunner.DefaultMessageExtractionConfig,
		l1Reader.Client(),
		builder.chainConfig,
		builder.addresses,
		melDB,
		blobReaderRegistry,
		nil,
		nil,
		reorgEventChan,
	)
	Require(t, err)
	Require(t, extractor.SetMessageConsumer(mockMsgConsumer))
	extractor.StopWaiter.Start(ctx, extractor)

	// Post a blob batch with a bunch of txs
	initialBatchCount := GetBatchCount(t, builder)
	var txs types.Transactions
	for i := 0; i < 20; i++ {
		tx, _ := builder.L2.TransferBalance(t, "Faucet", "User2", big.NewInt(1e12), builder.L2Info)
		txs = append(txs, tx)
	}
	builder.nodeConfig.BatchPoster.MaxDelay = 0
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
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
	lastState, err := melDB.GetHeadMelState()
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
	if lastState.DelayedMessagesSeen != numDelayedMessages {
		t.Fatalf(
			"MEL delayed message count %d does not match inbox tracker %d",
			lastState.DelayedMessagesSeen,
			numDelayedMessages,
		)
	}

	// Start from 1 to ignore the init message.
	for i := uint64(1); i < numDelayedMessages; i++ {
		fromInboxTracker, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedMessage(ctx, i)
		Require(t, err)
		delayedMsg, err := extractor.GetDelayedMessage(i)
		Require(t, err)
		// Check the messages we extracted from MEL and the inbox tracker are the same.
		if !fromInboxTracker.Equals(delayedMsg.Message) {
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
		WithDelayBuffer(threshold)
	builder.L2Info.GenerateAccount("User2")
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	builder.nodeConfig.MessageExtraction.Enable = false
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

	melDB, err := melrunner.NewDatabase(builder.L2.ConsensusNode.ConsensusDB)
	Require(t, err)
	Require(t, melDB.SaveState(melState)) // save head mel state
	mockMsgConsumer := &mockMELDB{savedMsgs: make([]*arbostypes.MessageWithMetadata, 0)}
	reorgEventChan := make(chan uint64, 1)
	extractor, err := melrunner.NewMessageExtractor(
		melrunner.DefaultMessageExtractionConfig,
		l1Reader.Client(),
		builder.chainConfig,
		builder.addresses,
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil,
		nil,
		reorgEventChan,
	)
	Require(t, err)
	Require(t, extractor.SetMessageConsumer(mockMsgConsumer))
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

	lastState, err := melDB.GetHeadMelState()
	Require(t, err)

	// Check that MEL extracted the same number of delayed messages the inbox tracker has seen.
	if lastState.DelayedMessagesSeen != numDelayedMessages {
		t.Fatalf(
			"MEL delayed message count %d does not match inbox tracker %d",
			lastState.DelayedMessagesSeen,
			numDelayedMessages,
		)
	}

	// Start from 1 to ignore the init message.
	for i := uint64(1); i < numDelayedMessages; i++ {
		fromInboxTracker, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedMessage(ctx, i)
		Require(t, err)
		delayedMsg, err := extractor.GetDelayedMessage(i)
		Require(t, err)
		// Check the messages we extracted from MEL and the inbox tracker are the same.
		if !fromInboxTracker.Equals(delayedMsg.Message) {
			t.Fatal("Messages from MEL and inbox tracker do not match")
		}
	}

	// Small reorg of 4 mel states
	reorgToBlockNum := lastState.ParentChainBlockNumber - 4
	reorgToState, err := melDB.State(reorgToBlockNum)
	Require(t, err)
	reorgToBlockHash := reorgToState.ParentChainBlockHash
	reorgToBlock, err := builder.L1.Client.BlockByHash(ctx, reorgToBlockHash)
	Require(t, err)
	Require(t, builder.L1.L1Backend.BlockChain().ReorgToOldBlock(reorgToBlock))
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 6)

	// Before checking if reorg handling works as intended, verify that starting a new message extractor will detect a reorg too and correctly transitions to Reorging step
	newExtractor, err := melrunner.NewMessageExtractor(
		melrunner.DefaultMessageExtractionConfig,
		l1Reader.Client(),
		builder.chainConfig,
		builder.addresses,
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil,
		nil,
		reorgEventChan,
	)
	Require(t, err)
	Require(t, newExtractor.SetMessageConsumer(mockMsgConsumer))
	newExtractor.StopWaiter.Start(ctx, newExtractor)
	for {
		prevFSMState := newExtractor.CurrentFSMState()
		_, err = newExtractor.Act(ctx)
		if err != nil {
			t.Fatal(err)
		}
		newFSMState := newExtractor.CurrentFSMState()
		// After reorg rewinding is done in the SavingMessages step, break
		if prevFSMState == melrunner.Start && newFSMState == melrunner.Reorging {
			break
		} else {
			t.Fatalf("new message extractor upon start did not transition to Reorging step. prevFSMState: %s, newFSMState: %s", prevFSMState.String(), newFSMState.String())
		}
	}

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

	lastState, err = melDB.GetHeadMelState()
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
		DontParalellise().
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
	{
		timeout := time.NewTimer(time.Minute)
		defer timeout.Stop()
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		for {
			headState, err := testClientB.ConsensusNode.MessageExtractor.GetHeadState()
			Require(t, err)
			if headState.ParentChainBlockNumber >= currHead+5 {
				break
			}
			select {
			case <-tick.C:
			case <-timeout.C:
				t.Fatal("timed out waiting for MEL to rewind head state after L1 reorg")
			}
		}
	}

	// Post a batch so that mel can send up-to-date L2 messages to txStreamer
	initialBatchCount := GetBatchCount(t, builder)
	var txs types.Transactions
	for i := 0; i < 10; i++ {
		tx, _ := builder.L2.TransferBalance(t, "Faucet", "User2", big.NewInt(1e12), builder.L2Info)
		txs = append(txs, tx)
	}
	builder.nodeConfig.BatchPoster.MaxDelay = 0
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	_, err = builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
	Require(t, err)
	for _, tx := range txs {
		_, err := testClientB.EnsureTxSucceeded(tx)
		Require(t, err, "tx not found on second node")
	}
	CheckBatchCount(t, builder, initialBatchCount+1)

	// Wait for the reorg to complete: MEL and TxStreamer reorg logs, then check balance.
	var reorgLogsFound bool
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if logHandler.WasLogged("MEL detected L1 reorg") &&
			logHandler.WasLogged("TransactionStreamer: Reorg detected!") {
			reorgLogsFound = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !reorgLogsFound {
		t.Fatal("timed out waiting for reorg logs")
	}
	expectedBalance := new(big.Int).Add(oldBalance, txOpts.Value)
	bal, err := builder.L2.Client.BalanceAt(ctx, txOpts.From, nil)
	if err != nil {
		t.Fatalf("BalanceAt: %v", err)
	}
	if bal.Cmp(expectedBalance) != 0 {
		t.Fatalf("balance=%v, want %v", bal, expectedBalance)
	}
}

func TestMessageExtractionLayer_UseArbDBForStoringDelayedMessages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	threshold := uint64(0)
	messagesPerBatch := uint64(3)

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithDelayBuffer(threshold)
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	builder.nodeConfig.MessageExtraction.Enable = false
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

	melDB, err := melrunner.NewDatabase(builder.L2.ConsensusNode.ConsensusDB)
	Require(t, err)
	Require(t, melDB.SaveState(melState)) // save head mel state
	// TODO: tx streamer to be used here when ready to run the node using mel thus replacing inbox reader-tracker code
	mockMsgConsumer := &mockMELDB{savedMsgs: make([]*arbostypes.MessageWithMetadata, 0)}
	reorgEventsChan := make(chan uint64, 1)
	extractor, err := melrunner.NewMessageExtractor(
		melrunner.DefaultMessageExtractionConfig,
		l1Reader.Client(),
		builder.chainConfig,
		builder.addresses,
		melDB,
		daprovider.NewDAProviderRegistry(),
		nil,
		nil,
		reorgEventsChan,
	)
	Require(t, err)
	Require(t, extractor.SetMessageConsumer(mockMsgConsumer))
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

	lastState, err := melDB.State(headMelStateBlockNum)
	Require(t, err)

	// Check that MEL extracted the same number of delayed messages the inbox tracker has seen.
	if lastState.DelayedMessagesSeen != numDelayedMessages {
		t.Fatalf(
			"MEL delayed message count %d does not match inbox tracker %d",
			lastState.DelayedMessagesSeen,
			numDelayedMessages,
		)
	}

	newInitialState, err := melDB.GetHeadMelState()
	Require(t, err)
	if newInitialState.ParentChainBlockHash != lastState.ParentChainBlockHash {
		t.Fatalf("head mel state ParentChainBlockHash mismatch. Want: %s, Have: %s", lastState.ParentChainBlockHash, newInitialState.ParentChainBlockHash)
	}
	for i := newInitialState.DelayedMessagesRead; i < newInitialState.DelayedMessagesSeen; i++ {
		// Validates the delayed messages saved by MEL match the inbox tracker
		delayedMsgSavedByMel, err := extractor.GetDelayedMessage(i)
		Require(t, err)
		fetchedDelayedMsg, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedMessage(ctx, i)
		Require(t, err)
		if !fetchedDelayedMsg.Equals(delayedMsgSavedByMel.Message) {
			t.Fatal("Messages from MEL and inbox tracker do not match")
		}
		t.Logf("validated delayed message of index: %d", i)
	}
}

// TestMELMigrationFromLegacyNode verifies that a node previously running with
// the legacy inbox reader/tracker can be seamlessly migrated to MEL.
//
// Test plan:
//
//	Phase 1 — Legacy node operation (MEL disabled):
//	  1. Build a sequencer node with MEL disabled
//	  2. Send L2 transactions and post a batch
//	  3. Send delayed messages (via L1 delayed inbox) and post a batch that consumes them
//	  4. Send additional delayed messages WITHOUT posting a batch, so that
//	     delayedSeen > delayedRead (unread delayed messages exist)
//	  5. Record the pre-migration state (delayed counts, batch count, msg count)
//	  6. Advance L1 until the finalized block is past all legacy data
//
//	Phase 2 — Restart with MEL enabled:
//	  7. Enable MEL in config and restart the node (RestartL2Node preserves the DB)
//	  8. The migration in validateAndInitializeDBForMEL should:
//	     - Read legacy batch/delayed counts from old DB schema keys
//	     - Query the on-chain bridge contract for the authoritative delayed message count
//	     - Construct and save an initial MEL state with correct counts and accumulator
//	  9. Wait for MEL to catch up to the latest parent chain block
//	 10. Verify MEL state has batch/msg counts matching pre-migration batched data
//
//	Phase 3 — Post-migration operations:
//	 11. Wait for the execution layer to fully process MEL extracted messages
//	 12. Send new L2 transactions (verifies sequencer works after migration)
//	 13. Send ETH deposits as delayed messages (verifies delayed inbox works)
//	 14. Post a batch that consumes the unread legacy delayed messages + new deposits
//	 15. Wait for MEL to process the new batch
//	 16. Verify MEL state shows increased counts for batches, messages, and delayed reads
func TestMELMigrationFromLegacyNode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Phase 1: Build node with MEL disabled (legacy mode)
	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithDelayBuffer(0)
	builder.L2Info.GenerateAccount("User2")
	builder.nodeConfig.MessageExtraction.Enable = false
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour
	cleanup := builder.Build(t)
	defer cleanup()

	// Send some L2 transactions
	for i := 0; i < 5; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}

	// Post a batch to include these L2 txs
	forceBatchPost(t, ctx, builder)

	// Send delayed messages via L1 and wait for inbox reader to process them
	delayedInboxContract, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	Require(t, err)
	sendDelayedMessagesViaL1(t, ctx, builder, delayedInboxContract, 5)

	// Post a batch to consume some of these delayed messages
	forceBatchPost(t, ctx, builder)

	// Send MORE delayed messages WITHOUT posting batches (to create delayedSeen > delayedRead)
	sendDelayedMessagesViaL1(t, ctx, builder, delayedInboxContract, 3)

	// Record pre-migration state
	preMigrationDelayedCount, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedCount()
	Require(t, err)
	preMigrationBatchCount, err := builder.L2.ConsensusNode.InboxTracker.GetBatchCount()
	Require(t, err)
	lastBatchMeta, err := builder.L2.ConsensusNode.InboxTracker.GetBatchMetadata(preMigrationBatchCount - 1)
	Require(t, err)
	preMigrationDelayedRead := lastBatchMeta.DelayedMessageCount

	t.Logf("Pre-migration state: delayedCount=%d, delayedRead=%d, batchCount=%d, batchedMsgCount=%d",
		preMigrationDelayedCount, preMigrationDelayedRead, preMigrationBatchCount, lastBatchMeta.MessageCount)

	// Verify we have unread delayed messages (delayedSeen > delayedRead)
	if preMigrationDelayedCount <= preMigrationDelayedRead {
		t.Fatalf("Expected unread delayed messages: delayedCount=%d should be > delayedRead=%d",
			preMigrationDelayedCount, preMigrationDelayedRead)
	}

	// Advance L1 until the finalized block is past the last batch's parent chain block.
	// This ensures the migration will include all legacy data.
	lastBatchBlock := lastBatchMeta.ParentChainBlock
	{
		timeout := time.NewTimer(30 * time.Second)
		defer timeout.Stop()
		for {
			finalizedHeader, err := builder.L1.Client.HeaderByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
			Require(t, err)
			if finalizedHeader.Number.Uint64() >= lastBatchBlock {
				t.Logf("Finalized block %d >= last batch block %d", finalizedHeader.Number.Uint64(), lastBatchBlock)
				break
			}
			AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 10)
			select {
			case <-timeout.C:
				t.Fatalf("timed out waiting for finalized block to catch up to last batch block %d", lastBatchBlock)
			default:
			}
		}
	}

	// Phase 2: Restart with MEL enabled
	builder.nodeConfig.MessageExtraction.Enable = true
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour
	builder.RestartL2Node(t)

	// Wait for MEL to catch up
	select {
	case <-builder.L2.ConsensusNode.MessageExtractor.CaughtUp():
		t.Log("MEL caught up after migration")
	case <-time.After(2 * time.Minute):
		t.Fatal("timed out waiting for MEL to catch up after migration")
	}

	// Verify migration state
	melExtractor := builder.L2.ConsensusNode.MessageExtractor
	headState, err := melExtractor.GetHeadState()
	Require(t, err)
	t.Logf("Post-migration MEL state: delayedSeen=%d, delayedRead=%d, batchCount=%d, msgCount=%d, parentChainBlock=%d",
		headState.DelayedMessagesSeen, headState.DelayedMessagesRead, headState.BatchCount, headState.MsgCount, headState.ParentChainBlockNumber)

	if headState.BatchCount < preMigrationBatchCount {
		t.Fatalf("MEL batch count %d is less than pre-migration batch count %d", headState.BatchCount, preMigrationBatchCount)
	}
	// Compare MEL msg count against the last batch's message count (not TxStreamer's count,
	// which includes unbatched messages from the delayed sequencer that MEL hasn't seen on L1 yet).
	preMigrationBatchedMsgCount := uint64(lastBatchMeta.MessageCount)
	if headState.MsgCount < preMigrationBatchedMsgCount {
		t.Fatalf("MEL msg count %d is less than pre-migration batched msg count %d", headState.MsgCount, preMigrationBatchedMsgCount)
	}

	// Phase 3: Post-migration operations
	// Wait for the execution layer to fully execute all messages MEL has extracted.
	// We need to wait until the node's pending nonce reflects the fully executed state.
	{
		ownerAddr := builder.L2Info.GetAddress("Owner")
		timeout := time.NewTimer(30 * time.Second)
		defer timeout.Stop()
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		var lastNonce uint64
		for {
			nonce, err := builder.L2.Client.NonceAt(ctx, ownerAddr, nil)
			if err == nil && nonce > 0 && nonce == lastNonce {
				// Nonce stabilized — execution has caught up
				break
			}
			lastNonce = nonce
			select {
			case <-tick.C:
			case <-timeout.C:
				t.Fatalf("timed out waiting for execution to catch up, last nonce: %d", lastNonce)
			}
		}
	}
	// Recalibrate L2 nonces — the restarted node's execution state may differ from
	// pre-migration because MEL only processes batched messages, not unbatched ones.
	builder.L2.RecalibrateNonce(t, builder.L2Info)
	ownerNonce := builder.L2Info.GetInfoWithPrivKey("Owner").Nonce.Load()
	t.Logf("Owner nonce after recalibration: %d", ownerNonce)

	// Send more L2 transactions
	for i := 0; i < 3; i++ {
		builder.L2.TransferBalance(t, "Owner", "User2", big.NewInt(1e12), builder.L2Info)
	}

	// Send ETH deposits as delayed messages post-migration (deposits don't need L2 nonce management)
	for i := 0; i < 2; i++ {
		depositTxOpts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
		depositTxOpts.Value = big.NewInt(1e16)
		l1tx, err := delayedInboxContract.DepositEth439370b1(&depositTxOpts)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, builder.L1.Client, l1tx)
		Require(t, err)
	}
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)

	// Post batches (which should consume the unread delayed messages + new deposits)
	forceBatchPost(t, ctx, builder)

	// Wait for MEL to process the new batch
	postBatchState, err := melExtractor.GetHeadState()
	Require(t, err)
	timeout := time.NewTimer(2 * time.Minute)
	defer timeout.Stop()
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for postBatchState.BatchCount <= headState.BatchCount {
		select {
		case <-tick.C:
			postBatchState, err = melExtractor.GetHeadState()
			Require(t, err)
		case <-timeout.C:
			t.Fatalf("timed out waiting for MEL to process new batch. current batch count: %d, expected > %d",
				postBatchState.BatchCount, headState.BatchCount)
		}
	}

	t.Logf("Final MEL state: delayedSeen=%d, delayedRead=%d, batchCount=%d, msgCount=%d",
		postBatchState.DelayedMessagesSeen, postBatchState.DelayedMessagesRead, postBatchState.BatchCount, postBatchState.MsgCount)

	// Verify counts increased
	if postBatchState.BatchCount <= headState.BatchCount {
		t.Fatalf("MEL batch count did not increase: %d", postBatchState.BatchCount)
	}
	if postBatchState.MsgCount <= headState.MsgCount {
		t.Fatalf("MEL msg count did not increase: %d", postBatchState.MsgCount)
	}
	// The new batch should have consumed the unread delayed messages from migration + new delayed messages
	if postBatchState.DelayedMessagesRead <= preMigrationDelayedRead {
		t.Fatalf("MEL delayed messages read did not increase past pre-migration: %d <= %d",
			postBatchState.DelayedMessagesRead, preMigrationDelayedRead)
	}
}

type mockMELDB struct {
	savedMsgs []*arbostypes.MessageWithMetadata
}

func (m *mockMELDB) PushMessages(ctx context.Context, firstMsgIdx uint64, messages []*arbostypes.MessageWithMetadata) error {
	m.savedMsgs = append(m.savedMsgs, messages...)
	return nil
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
		DelayedMessagesSeen:                1,
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
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	posted, err := builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
	Require(t, err)
	if !posted {
		t.Fatal("forceDelayedBatchPosting: sequencer batch was not posted")
	}
	for _, tx := range txs {
		_, err := testClientB.EnsureTxSucceeded(tx)
		Require(t, err, "tx not found on second node")
	}

	CheckBatchCount(t, builder, initialBatchCount+1)
	// Reset the max delay.
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
}

// sendDelayedMessagesViaL1 sends numMsgs delayed messages through the L1 delayed inbox,
// advances L1, and waits for the inbox reader to process them.
func sendDelayedMessagesViaL1(
	t *testing.T,
	ctx context.Context,
	builder *NodeBuilder,
	delayedInbox *bridgegen.Inbox,
	numMsgs int,
) {
	t.Helper()
	countBefore, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedCount()
	Require(t, err)
	for i := 0; i < numMsgs; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(int64(i+1)*1e6), nil)
		txBytes, err := tx.MarshalBinary()
		Require(t, err)
		txWrapped := append([]byte{arbos.L2MessageKind_SignedTx}, txBytes...)
		usertxopts := builder.L1Info.GetDefaultTransactOpts("User", ctx)
		l1tx, err := delayedInbox.SendL2Message(&usertxopts, txWrapped)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, builder.L1.Client, l1tx)
		Require(t, err)
	}
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)
	waitForDelayedCount(t, ctx, builder, countBefore+uint64(numMsgs)) // #nosec G115
}

// waitForDelayedCount polls the inbox tracker until the delayed message count reaches the expected value.
func waitForDelayedCount(t *testing.T, ctx context.Context, builder *NodeBuilder, expected uint64) {
	t.Helper()
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		count, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedCount()
		Require(t, err)
		if count >= expected {
			return
		}
		select {
		case <-tick.C:
		case <-timeout.C:
			t.Fatalf("timed out waiting for delayed count: got %d, want %d", count, expected)
		}
	}
}

// forceBatchPost triggers a batch post and resets MaxDelay back to high.
func forceBatchPost(t *testing.T, ctx context.Context, builder *NodeBuilder) {
	t.Helper()
	builder.nodeConfig.BatchPoster.MaxDelay = 0
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	posted, err := builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
	Require(t, err)
	if !posted {
		t.Fatal("sequencer batch was not posted")
	}
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
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
