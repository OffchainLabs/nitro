package arbtest

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/bold/solgen/go/bridgegen"
	"github.com/offchainlabs/bold/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	mel "github.com/offchainlabs/nitro/arbnode/message-extraction"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/staker/bold"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/blobs"
	"github.com/offchainlabs/nitro/util/headerreader"
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

	mockDB := &mockMELDB{
		savedMsgs:        make([]*arbostypes.MessageWithMetadata, 0),
		savedStates:      make(map[uint64]*meltypes.State),
		savedDelayedMsgs: make([]*arbnode.DelayedInboxMessage, 0),
	}
	Require(t, mockDB.SaveState(ctx, melState))
	extractor, err := mel.NewMessageExtractor(
		l1Reader.Client(),
		builder.addresses,
		mockDB,
		mockDB,
		mockDB,
		nil, // TODO: Provide da readers here.
		melState.ParentChainBlockHash,
		0,
	)
	Require(t, err)

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
		if prevFSMState == mel.ProcessingNextBlock && newFSMState == mel.ProcessingNextBlock {
			break
		}
	}

	// Assert details about the extraction routine.
	if len(mockDB.savedStates) == 0 {
		t.Fatal("MEL did not save any states")
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
	lastState := mockDB.lastState
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
	melMessages := mockDB.savedMsgs
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
	t.Skip("Failing due to blob txs not being mined")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithBoldDeployment()
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

	mockDB := &mockMELDB{
		savedMsgs:        make([]*arbostypes.MessageWithMetadata, 0),
		savedStates:      make(map[uint64]*meltypes.State),
		savedDelayedMsgs: make([]*arbnode.DelayedInboxMessage, 0),
	}
	Require(t, mockDB.SaveState(ctx, melState))
	extractor, err := mel.NewMessageExtractor(
		l1Reader.Client(),
		builder.addresses,
		mockDB,
		mockDB,
		mockDB,
		nil, // TODO: Provide da readers here.
		melState.ParentChainBlockHash,
		0,
	)
	Require(t, err)

	// Create various L2 transactions and wait for them to be included in a batch
	// as compressed messages submitted to the sequencer inbox.
	sequencerTxOpts := builder.L1Info.GetDefaultTransactOpts("Sequencer", ctx)
	seqPrivKey := builder.L1Info.GetInfoWithPrivKey("Sequencer").PrivateKey
	numMessages := 10
	forceBlobSequencerMessageBatchPosting(
		t,
		builder.L1Info,
		builder.L2.ConsensusNode,
		builder.L2Info,
		builder.L1.Client,
		&sequencerTxOpts,
		seqPrivKey,
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
		if prevFSMState == mel.ProcessingNextBlock && newFSMState == mel.ProcessingNextBlock {
			break
		}
	}

	// Assert details about the extraction routine.
	if len(mockDB.savedStates) == 0 {
		t.Fatal("MEL did not save any states")
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

	mockDB := &mockMELDB{
		savedMsgs:        make([]*arbostypes.MessageWithMetadata, 0),
		savedStates:      make(map[uint64]*meltypes.State),
		savedDelayedMsgs: make([]*arbnode.DelayedInboxMessage, 0),
	}
	Require(t, mockDB.SaveState(ctx, melState))
	extractor, err := mel.NewMessageExtractor(
		l1Reader.Client(),
		builder.addresses,
		mockDB,
		mockDB,
		mockDB,
		nil, // TODO: Provide da readers here.
		melState.ParentChainBlockHash,
		0,
	)
	Require(t, err)

	for {
		prevFSMState := extractor.CurrentFSMState()
		_, err = extractor.Act(ctx)
		Require(t, err)
		newFSMState := extractor.CurrentFSMState()
		// If the extractor FSM has been in the ProcessingNextBlock state twice in a row, without error, it means
		// it has caught up to the latest (or configured safe/finalized) parent chain block. We can
		// exit the loop here and assert information about MEL.
		if prevFSMState == mel.ProcessingNextBlock && newFSMState == mel.ProcessingNextBlock {
			break
		}
	}

	if len(mockDB.savedStates) == 0 {
		t.Fatal("MEL did not save any states")
	}

	numDelayedMessages, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedCount()
	Require(t, err)
	// lastState := mockDB.savedStates[len(mockDB.savedStates)-1]
	lastState := mockDB.lastState

	// Check that MEL extracted the same number of delayed messages the inbox tracker has seen.
	if lastState.DelayedMessagedSeen != numDelayedMessages {
		t.Fatalf(
			"MEL delayed message count %d does not match inbox tracker %d",
			lastState.DelayedMessagedSeen,
			numDelayedMessages,
		)
	}
	delayedInInboxTracker := make([]*arbostypes.L1IncomingMessage, 0)

	// Start from 1 to ignore the init message.
	for i := uint64(1); i < numDelayedMessages; i++ {
		fetchedDelayedMsg, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedMessage(ctx, i)
		Require(t, err)
		delayedInInboxTracker = append(delayedInInboxTracker, fetchedDelayedMsg)
	}

	// Check the messages we extracted from MEL and the inbox tracker are the same.
	for i, delayedMsg := range mockDB.savedDelayedMsgs {
		fromInboxTracker := delayedInInboxTracker[i]
		if !fromInboxTracker.Equals(delayedMsg.Message) {
			t.Fatal("Messages from MEL and inbox tracker do not match")
		}
	}

	// Small reorg of 4 mel states
	reorgToBlockNum := mockDB.lastState.ParentChainBlockNumber - 4
	reorgToBlockHash := mockDB.savedStates[reorgToBlockNum].ParentChainBlockHash
	reorgToBlock, err := builder.L1.Client.BlockByHash(ctx, reorgToBlockHash)
	Require(t, err)
	Require(t, builder.L1.L1Backend.BlockChain().ReorgToOldBlock(reorgToBlock))

	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 3)
	// Check if ReorgingToOldBlock fsm state works as intended
	for {
		prevFSMState := extractor.CurrentFSMState()
		_, err = extractor.Act(ctx)
		if err != nil {
			t.Fatal(err)
		}
		newFSMState := extractor.CurrentFSMState()
		// After reorg rewinding is done in the SavingMessages step, break
		if prevFSMState == mel.SavingMessages && newFSMState == mel.ProcessingNextBlock {
			break
		}
	}

	if mockDB.lastState.ParentChainBlockNumber != reorgToBlockNum+1 {
		t.Fatalf("Unexpected number of MEL states after a parent chain reorg. Want: %d, Have: %d", reorgToBlockNum+1, mockDB.lastState.ParentChainBlockNumber)
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

	melDB := mel.NewDatabase(builder.L2.ConsensusNode.ArbDB)
	Require(t, melDB.SaveState(ctx, melState)) // save head mel state
	// TODO: tx streamer to be used here when ready to run the node using mel thus replacing inbox reader-tracker code
	mockMsgConsumer := &mockMELDB{savedMsgs: make([]*arbostypes.MessageWithMetadata, 0)}
	extractor, err := mel.NewMessageExtractor(
		l1Reader.Client(),
		builder.addresses,
		melDB,
		melDB,
		mockMsgConsumer,
		nil, // TODO: Provide da readers here.
		melState.ParentChainBlockHash,
		0,
	)
	Require(t, err)

	for {
		prevFSMState := extractor.CurrentFSMState()
		_, err = extractor.Act(ctx)
		Require(t, err)
		newFSMState := extractor.CurrentFSMState()
		// If the extractor FSM has been in the ProcessingNextBlock state twice in a row, without error, it means
		// it has caught up to the latest (or configured safe/finalized) parent chain block. We can
		// exit the loop here and assert information about MEL.
		if prevFSMState == mel.ProcessingNextBlock && newFSMState == mel.ProcessingNextBlock {
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

	newInitialState, err := melDB.FetchInitialState(ctx, lastState.ParentChainBlockHash, 0)
	Require(t, err)
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

	// // Start from 1 to ignore the init message and check the messages we extracted from MEL and the inbox tracker are the same.
	// for i := uint64(1); i < numDelayedMessages; i++ {
	// 	fetchedDelayedMsg, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedMessage(ctx, i)
	// 	Require(t, err)
	// 	delayedMsgSavedByMel, err := melDB.ReadDelayedMessage(ctx, newInitialState, i)
	// 	Require(t, err)
	// 	if !fetchedDelayedMsg.Equals(delayedMsgSavedByMel.Message) {
	// 		t.Fatal("Messages from MEL and inbox tracker do not match")
	// 	}
	// }
}

type mockMELDB struct {
	savedMsgs        []*arbostypes.MessageWithMetadata
	savedDelayedMsgs []*arbnode.DelayedInboxMessage
	savedStates      map[uint64]*meltypes.State
	lastState        *meltypes.State
}

func (m *mockMELDB) PushMessages(ctx context.Context, firstMsgIdx uint64, messages []*arbostypes.MessageWithMetadata) error {
	m.savedMsgs = append(m.savedMsgs, messages...)
	return nil
}

func (m *mockMELDB) State(
	ctx context.Context,
	parentChainBlockNumber uint64,
) (*meltypes.State, error) {
	state, ok := m.savedStates[parentChainBlockNumber]
	if !ok {
		return nil, errors.New("state not found")
	}
	return state, nil
}

func (m *mockMELDB) SaveState(
	ctx context.Context,
	state *meltypes.State,
) error {
	m.savedStates[state.ParentChainBlockNumber] = state
	m.lastState = state
	return nil
}

func (m *mockMELDB) FetchInitialState(
	ctx context.Context, parentChainBlockHash common.Hash, _ uint64,
) (*meltypes.State, error) {
	if m.lastState.ParentChainBlockHash != parentChainBlockHash {
		return nil, fmt.Errorf("parentChainBlockHash of db doesnt match the hash queried by initialStateFetcher")
	}
	return m.savedStates[m.lastState.ParentChainBlockNumber], nil
}

func (m *mockMELDB) SaveDelayedMessages(
	ctx context.Context,
	state *meltypes.State,
	delayedMessages []*arbnode.DelayedInboxMessage,
) error {
	m.savedDelayedMsgs = append(m.savedDelayedMsgs, delayedMessages...)
	return nil
}
func (m *mockMELDB) ReadDelayedMessage(
	ctx context.Context,
	_ *meltypes.State,
	index uint64,
) (*arbnode.DelayedInboxMessage, error) {
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
) *meltypes.State {
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
	return &meltypes.State{
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

func forceBlobSequencerMessageBatchPosting(
	t *testing.T,
	l1Info *BlockchainTestInfo,
	l2Node *arbnode.Node,
	l2Info *BlockchainTestInfo,
	backend *ethclient.Client,
	sequencer *bind.TransactOpts,
	seqPrivKey *ecdsa.PrivateKey,
	seqInboxAddr common.Address,
	numMessages int64,
) {
	ctx := context.Background()
	batchBuffer := bytes.NewBuffer([]byte{})
	for i := int64(0); i < numMessages; i++ {
		value := i
		err := writeTxToBatch(batchBuffer, l2Info.PrepareTx("Owner", "User2", l2Info.TransferGas, big.NewInt(value), []byte{}))
		Require(t, err)
	}
	compressed, err := arbcompress.CompressWell(batchBuffer.Bytes())
	Require(t, err)
	batchMessageData := append([]byte{0}, compressed...)
	kzgBlobs, err := blobs.EncodeBlobs(batchMessageData)
	Require(t, err)

	seqNum := new(big.Int).Lsh(common.Big1, 256)
	seqNum.Sub(seqNum, common.Big1)

	seqInboxABI, err := bridgegen.SequencerInboxMetaData.GetAbi()
	Require(t, err)
	method, ok := seqInboxABI.Methods["addSequencerL2BatchFromBlobs"]
	if !ok {
		t.Fatal("Method not found in ABI")
	}
	var args []any
	args = append(args, seqNum)
	args = append(args, new(big.Int).SetUint64(1)) // num delayed messages.
	args = append(args, common.Address{})
	args = append(args, new(big.Int).SetUint64(uint64(0))) // prev msg num.
	args = append(args, new(big.Int).SetUint64(uint64(0))) // new msg num.
	calldata, err := method.Inputs.Pack(args...)
	Require(t, err)
	fullCalldata := append([]byte{}, method.ID...)
	fullCalldata = append(fullCalldata, calldata...)

	// Prepare a blob transaction to submit to the sequencer inbox.
	commitments, blobHashes, err := blobs.ComputeCommitmentsAndHashes(kzgBlobs)
	Require(t, err)
	proofs, err := blobs.ComputeBlobProofs(kzgBlobs, commitments)
	Require(t, err)
	nonce, err := backend.NonceAt(ctx, sequencer.From, nil)
	Require(t, err)
	chainId, err := backend.ChainID(ctx)
	Require(t, err)
	inner := &types.BlobTx{
		Nonce: nonce,
		Gas:   1_000_000,
		To:    seqInboxAddr,
		Value: uint256.MustFromBig(big.NewInt(0)),
		Data:  fullCalldata,
		Sidecar: &types.BlobTxSidecar{
			Blobs:       kzgBlobs,
			Commitments: commitments,
			Proofs:      proofs,
		},
		BlobHashes: blobHashes,
		ChainID:    uint256.MustFromBig(chainId),
	}

	gas := estimateGasSimple(
		t,
		ctx,
		sequencer.From,
		seqInboxAddr,
		backend,
		fullCalldata,
		kzgBlobs,
		types.AccessList{},
	)
	inner.Gas = gas

	fullTx := l1Info.SignTxAs("Sequencer", inner)
	Require(t, backend.SendTransaction(ctx, fullTx))

	receipt, err := bind.WaitMined(ctx, backend, fullTx)
	Require(t, err)
	_ = receipt
}

type estimateGasParams struct {
	From         common.Address   `json:"from"`
	To           *common.Address  `json:"to"`
	Data         hexutil.Bytes    `json:"data"`
	MaxFeePerGas *hexutil.Big     `json:"maxFeePerGas"`
	AccessList   types.AccessList `json:"accessList"`
	BlobHashes   []common.Hash    `json:"blobVersionedHashes,omitempty"`
}

func estimateGasSimple(
	t *testing.T,
	ctx context.Context,
	from common.Address,
	to common.Address,
	rpcClient *ethclient.Client,
	realData []byte,
	realBlobs []kzg4844.Blob,
	realAccessList types.AccessList,
) uint64 {
	rawRpcClient := rpcClient.Client()
	latestHeader, err := rpcClient.HeaderByNumber(ctx, nil)
	Require(t, err)
	maxFeePerGas := arbmath.BigMulByUBips(latestHeader.BaseFee, arbmath.OneInUBips*3/2)
	_, realBlobHashes, err := blobs.ComputeCommitmentsAndHashes(realBlobs)
	Require(t, err)
	// If we're at the latest nonce, we can skip the special future tx estimate stuff
	gas, err := estimateGas(rawRpcClient, ctx, estimateGasParams{
		From:         from,
		To:           &to,
		Data:         realData,
		MaxFeePerGas: (*hexutil.Big)(maxFeePerGas),
		BlobHashes:   realBlobHashes,
		AccessList:   realAccessList,
	})
	Require(t, err)
	return gas
}

func estimateGas(client rpc.ClientInterface, ctx context.Context, params estimateGasParams) (uint64, error) {
	var gas hexutil.Uint64
	err := client.CallContext(ctx, &gas, "eth_estimateGas", params)
	return uint64(gas), err
}
