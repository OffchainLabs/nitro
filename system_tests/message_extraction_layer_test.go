package arbtest

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	mel "github.com/offchainlabs/nitro/arbnode/message-extraction"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

type mockMELStateFetcher struct {
	state *mel.State
}

func (m *mockMELStateFetcher) GetState(
	ctx context.Context, parentChainBlockHash common.Hash,
) (*mel.State, error) {
	return m.state, nil
}

type mockMELDB struct{}

func (m *mockMELDB) SaveState(
	ctx context.Context,
	state *mel.State,
	messages []*arbostypes.MessageWithMetadata,
) error {
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
	latestBlock, err := builder.L1.Client.BlockByNumber(ctx, nil)
	Require(t, err)
	chainId, err := builder.L1.Client.ChainID(ctx)
	Require(t, err)
	melState := &mel.State{
		Version:                0,
		ParentChainId:          chainId.Uint64(),
		ParentChainBlockNumber: 0,
		// ParentChainBlockHash:         builder.L1Info.BlockHash,
		// ParentChainPreviousBlockHash: builder.L1Info.BlockHash,
		BatchPostingTargetAddress: builder.addresses.SequencerInbox,
		MessageAccumulator:        common.Hash{},
	}

	// Construct a new MEL service and provide with an initial MEL state
	// to begin extracting messages from the parent chain.
	melService, err := mel.NewMessageExtractionLayer(
		builder.L1.ConsensusNode.L1Reader,
		builder.addresses,
		&mockMELStateFetcher{state: melState},
		&mockMELDB{},
		nil, // TODO: Provide a da reader here.
		func() *mel.MELConfig {
			return &mel.DefaultMELConfig
		},
	)
	Require(t, err)

	postState, err := melService.WalkForwards(
		ctx,
		melState,
		builder.L1.Client,
		latestBlock.NumberU64(),
	)
	Require(t, err)
	_ = postState
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
