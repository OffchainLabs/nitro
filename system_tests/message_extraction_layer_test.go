package arbtest

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/bold/solgen/go/bridgegen"
	"github.com/offchainlabs/bold/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	mel "github.com/offchainlabs/nitro/arbnode/message-extraction"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/staker/bold"
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

	melState := createInitialMELState(t, ctx, builder.addresses.Rollup, builder.L1.Client)

	seqInbox, err := arbnode.NewSequencerInbox(builder.L1.Client, builder.addresses.SequencerInbox, 0)
	Require(t, err)
	delayedBridge, err := arbnode.NewDelayedBridge(builder.L1.Client, builder.addresses.Bridge, 0)
	Require(t, err)

	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L1.Client)
	l1Reader, err := headerreader.New(ctx, builder.L1.Client, func() *headerreader.Config { return &headerreader.TestConfig }, arbSys)
	Require(t, err)
	l1Reader.Start(ctx)
	defer l1Reader.StopAndWait()

	mockDB := &mockMELDB{
		savedMsgs:        make([]*arbostypes.MessageWithMetadata, 0),
		savedStates:      make([]*mel.State, 0),
		savedDelayedMsgs: make([]*arbnode.DelayedInboxMessage, 0),
	}
	extractor, err := mel.NewMessageExtractor(
		l1Reader,
		builder.addresses,
		&mockMELStateFetcher{state: melState},
		mockDB,
		seqInbox,
		delayedBridge,
		nil, // TODO: Provide da readers here.
		func() *mel.MELConfig {
			return &mel.DefaultMELConfig
		},
	)
	Require(t, err)
	_ = extractor

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
		Require(t, extractor.Act(ctx))
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
	melState := createInitialMELState(t, ctx, builder.addresses.Rollup, builder.L1.Client)

	// Construct a new MEL service and provide with an initial MEL state
	// to begin extracting messages from the parent chain.
	seqInbox, err := arbnode.NewSequencerInbox(builder.L1.Client, builder.addresses.SequencerInbox, 0)
	Require(t, err)
	delayedBridge, err := arbnode.NewDelayedBridge(builder.L1.Client, builder.addresses.Bridge, 0)
	Require(t, err)

	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L1.Client)
	l1Reader, err := headerreader.New(ctx, builder.L1.Client, func() *headerreader.Config { return &headerreader.TestConfig }, arbSys)
	Require(t, err)
	l1Reader.Start(ctx)
	defer l1Reader.StopAndWait()

	mockDB := &mockMELDB{
		savedMsgs:        make([]*arbostypes.MessageWithMetadata, 0),
		savedStates:      make([]*mel.State, 0),
		savedDelayedMsgs: make([]*arbnode.DelayedInboxMessage, 0),
	}
	extractor, err := mel.NewMessageExtractor(
		l1Reader,
		builder.addresses,
		&mockMELStateFetcher{state: melState},
		mockDB,
		seqInbox,
		delayedBridge,
		nil, // TODO: Provide da readers here.
		func() *mel.MELConfig {
			return &mel.DefaultMELConfig
		},
	)
	Require(t, err)

	for {
		prevFSMState := extractor.CurrentFSMState()
		Require(t, extractor.Act(ctx))
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
	lastState := mockDB.savedStates[len(mockDB.savedStates)-1]

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
	for i := 1; i < int(numDelayedMessages); i++ {
		fetchedDelayedMsg, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedMessage(ctx, uint64(i))
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
}

type mockMELStateFetcher struct {
	state *mel.State
}

func (m *mockMELStateFetcher) GetState(
	ctx context.Context, parentChainBlockHash common.Hash,
) (*mel.State, error) {
	return m.state, nil
}

type mockMELDB struct {
	savedMsgs        []*arbostypes.MessageWithMetadata
	savedDelayedMsgs []*arbnode.DelayedInboxMessage
	savedStates      []*mel.State
}

func (m *mockMELDB) SaveState(
	ctx context.Context,
	state *mel.State,
	messages []*arbostypes.MessageWithMetadata,
) error {
	m.savedStates = append(m.savedStates, state)
	m.savedMsgs = append(m.savedMsgs, messages...)
	return nil
}

func (m *mockMELDB) SaveDelayedMessages(
	ctx context.Context,
	state *mel.State,
	delayedMessages []*arbnode.DelayedInboxMessage,
) error {
	m.savedDelayedMsgs = append(m.savedDelayedMsgs, delayedMessages...)
	return nil
}
func (m *mockMELDB) ReadDelayedMessage(
	ctx context.Context,
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
	rollupAddr common.Address,
	client *ethclient.Client,
) *mel.State {
	// Create an initial MEL state from the latest confirmed assertion.
	rollup, err := rollupgen.NewRollupUserLogic(rollupAddr, client)
	Require(t, err)
	confirmedHash, err := rollup.LatestConfirmed(&bind.CallOpts{})
	Require(t, err)
	latestConfirmedAssertion, err := bold.ReadBoldAssertionCreationInfo(
		ctx,
		rollup,
		client,
		rollupAddr,
		confirmedHash,
	)
	Require(t, err)
	startBlock, err := client.BlockByNumber(ctx, new(big.Int).SetUint64(latestConfirmedAssertion.CreationL1Block))
	Require(t, err)
	chainId, err := client.ChainID(ctx)
	Require(t, err)

	// TODO: Construct the correct MEL state from the latest confirmed assertion.
	return &mel.State{
		Version:                      0,
		ParentChainId:                chainId.Uint64(),
		ParentChainBlockNumber:       startBlock.NumberU64(),
		ParentChainBlockHash:         startBlock.Hash(),
		ParentChainPreviousBlockHash: startBlock.ParentHash(),
		MessageAccumulator:           common.Hash{},
		DelayedMessagedSeen:          1,
		DelayedMessagesRead:          1, // Assumes we have read the init message.
		MsgCount:                     1,
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
