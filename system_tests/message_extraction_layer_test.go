package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/bold/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/arbnode"
	mel "github.com/offchainlabs/nitro/arbnode/message-extraction"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/staker/bold"
	"github.com/offchainlabs/nitro/util/headerreader"
)

type mockMELStateFetcher struct {
	state *mel.State
}

func (m *mockMELStateFetcher) GetState(
	ctx context.Context, parentChainBlockHash common.Hash,
) (*mel.State, error) {
	return m.state, nil
}

type mockMELDB struct {
	savedMsgs   []*arbostypes.MessageWithMetadata
	savedStates []*mel.State
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

func TestMessageExtractionLayer(t *testing.T) {
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

	// Force a batch to be posted and ensure it is reflected in the onchain contracts.
	forceBatchPosting(t, ctx, builder, testClientB, messagesPerBatch, threshold)

	// Create an initial MEL state from the latest confirmed assertion.
	rollup, err := rollupgen.NewRollupUserLogic(builder.addresses.Rollup, builder.L1.Client)
	Require(t, err)
	confirmedHash, err := rollup.LatestConfirmed(&bind.CallOpts{})
	Require(t, err)
	latestConfirmedAssertion, err := bold.ReadBoldAssertionCreationInfo(
		ctx,
		rollup,
		builder.L1.Client,
		builder.addresses.Rollup,
		confirmedHash,
	)
	Require(t, err)
	startBlock, err := builder.L1.Client.BlockByNumber(ctx, new(big.Int).SetUint64(latestConfirmedAssertion.CreationL1Block))
	Require(t, err)
	chainId, err := builder.L1.Client.ChainID(ctx)
	Require(t, err)
	melState := &mel.State{
		Version:                      0,
		ParentChainId:                chainId.Uint64(),
		ParentChainBlockNumber:       startBlock.NumberU64(),
		ParentChainBlockHash:         startBlock.Hash(),
		ParentChainPreviousBlockHash: startBlock.ParentHash(),
		BatchPostingTargetAddress:    builder.addresses.SequencerInbox,
		MessageAccumulator:           common.Hash{},
	}

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
		savedMsgs:   make([]*arbostypes.MessageWithMetadata, 0),
		savedStates: make([]*mel.State, 0),
	}
	extractor, err := mel.NewMessageExtractor(
		l1Reader,
		builder.addresses,
		&mockMELStateFetcher{state: melState},
		mockDB,
		seqInbox,
		delayedBridge,
		nil, // TODO: Provide a da reader here.
		func() *mel.MELConfig {
			return &mel.DefaultMELConfig
		},
	)
	Require(t, err)

	_ = extractor
	err = extractor.Act(ctx)
	Require(t, err)
	err = extractor.Act(ctx)
	Require(t, err)
	err = extractor.Act(ctx)
	Require(t, err)

	for _, msg := range mockDB.savedMsgs {
		fmt.Printf("after delayed %d, %+v\n", msg.DelayedMessagesRead, msg.Message)
	}
	time.Sleep(time.Hour)
}

func forceBatchPosting(
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
