// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package chainwatcher

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/bold/chainabstraction"
	"github.com/offchainlabs/nitro/bold/challengetesting/mocks"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/containers/threadsafe"
	"github.com/offchainlabs/nitro/bold/layer2stateprovider"
	"github.com/offchainlabs/nitro/solgen/go/challengeV2gen"
)

func simpleAssertionMetadata() *layer2stateprovider.AssociatedAssertionMetadata {
	return &layer2stateprovider.AssociatedAssertionMetadata{
		WasmModuleRoot: common.Hash{},
		FromState: chainabstraction.GoGlobalState{
			Batch:      0,
			PosInBatch: 0,
		},
		BatchLimit: 1,
	}
}

func Test_challengedAssertionConfirmableBlock(t *testing.T) {
	t.Run("assertion confirm period has not yet passed", func(t *testing.T) {
		parentInfo := &chainabstraction.AssertionCreatedInfo{
			ConfirmPeriodBlocks: 50,
		}
		info := &chainabstraction.AssertionCreatedInfo{
			CreationParentBlock: 100,
			CreationL1Block:     100, // in case of l2 chain CreationL1Block is equal to CreationParentBlock
		}
		edgeConfirmationBlock := uint64(200)
		gracePeriodBlocks := uint64(10)
		want := edgeConfirmationBlock + gracePeriodBlocks
		got := challengedAssertionConfirmableBlock(parentInfo, edgeConfirmationBlock, info, gracePeriodBlocks)
		require.Equal(t, want, got)
	})
	t.Run("assertion confirm period has passed", func(t *testing.T) {
		parentInfo := &chainabstraction.AssertionCreatedInfo{
			ConfirmPeriodBlocks: 50,
		}
		info := &chainabstraction.AssertionCreatedInfo{
			CreationParentBlock: 100,
			CreationL1Block:     100, // in case of l2 chain CreationL1Block is equal to CreationParentBlock
		}
		edgeConfirmationBlock := uint64(105)
		gracePeriodBlocks := uint64(10)
		want := parentInfo.ConfirmPeriodBlocks + info.CreationL1Block
		got := challengedAssertionConfirmableBlock(parentInfo, edgeConfirmationBlock, info, gracePeriodBlocks)
		require.Equal(t, want, got)
	})
}

func TestWatcher_processEdgeConfirmation(t *testing.T) {
	ctx := context.Background()
	mockChain := &mocks.MockProtocol{}
	mockChallengeManager := &mocks.MockSpecChallengeManager{}
	mockChain.On(
		"SpecChallengeManager",
	).Return(mockChallengeManager, nil)

	assertionHash := chainabstraction.AssertionHash{Hash: common.BytesToHash([]byte("foo"))}
	mockChain.On(
		"IsChallengeComplete",
		ctx,
		assertionHash,
	).Return(false, nil)
	edgeId := chainabstraction.EdgeId{Hash: common.BytesToHash([]byte("bar"))}
	edge := &mocks.MockSpecEdge{}

	mockChallengeManager.On(
		"GetEdge", ctx, edgeId,
	).Return(option.Some(chainabstraction.SpecEdge(edge)), nil)

	edge.On("ClaimId").Return(option.Some(chainabstraction.ClaimId(assertionHash.Hash)))
	edge.On("Id").Return(edgeId)
	edge.On("GetChallengeLevel").Return(chainabstraction.ChallengeLevel(1), nil)
	edge.On(
		"AssertionHash",
		ctx,
	).Return(assertionHash, nil)

	watcher := &Watcher{
		challenges: threadsafe.NewMap[chainabstraction.AssertionHash, *trackedChallenge](),
		chain:      mockChain,
	}
	watcher.challenges.Put(assertionHash, &trackedChallenge{
		confirmedLevelZeroEdgeClaimIds: threadsafe.NewMap[chainabstraction.ClaimId, chainabstraction.EdgeId](),
	})

	err := watcher.processEdgeConfirmation(ctx, edgeId)
	require.NoError(t, err)

	chal, ok := watcher.challenges.TryGet(assertionHash)
	require.Equal(t, true, ok)
	ok = chal.confirmedLevelZeroEdgeClaimIds.Has(chainabstraction.ClaimId(assertionHash.Hash))
	require.Equal(t, true, ok)
}

func TestWatcher_processEdgeAddedEvent(t *testing.T) {
	ctx := context.Background()
	mockChain := &mocks.MockProtocol{}
	mockChallengeManager := &mocks.MockSpecChallengeManager{}
	mockChain.On(
		"SpecChallengeManager",
	).Return(mockChallengeManager, nil)

	assertionHash := chainabstraction.AssertionHash{Hash: common.BytesToHash([]byte("foo"))}
	parentAssertionHash := chainabstraction.AssertionHash{Hash: common.BytesToHash([]byte("parent foo"))}
	edgeId := chainabstraction.EdgeId{Hash: common.BytesToHash([]byte("bar"))}
	originId := chainabstraction.OriginId(common.BytesToHash([]byte("origin bar")))
	edge := &mocks.MockSpecEdge{}
	edge.On("Status", ctx).Return(chainabstraction.EdgePending, nil)
	edge.On("GetTotalChallengeLevels", ctx).Return(uint8(3), nil)
	edge.On("HasChildren", ctx).Return(false, nil)
	edge.On("MarkAsHonest").Return()
	mockHonest := &mocks.MockHonestEdge{MockSpecEdge: edge}
	edge.On("AsVerifiedHonest").Return(mockHonest, true)

	mockChain.On(
		"IsChallengeComplete",
		ctx,
		assertionHash,
	).Return(false, nil)
	mockChain.On(
		"TopLevelAssertion",
		ctx,
		edgeId,
	).Return(assertionHash, nil)

	info := &chainabstraction.AssertionCreatedInfo{
		InboxMaxCount:       big.NewInt(1),
		ParentAssertionHash: parentAssertionHash,
	}
	mockChain.On(
		"ReadAssertionCreationInfo",
		ctx,
		assertionHash,
	).Return(info, nil)
	parentInfo := &chainabstraction.AssertionCreatedInfo{
		InboxMaxCount: big.NewInt(1),
	}
	mockChain.On(
		"ReadAssertionCreationInfo",
		ctx,
		parentAssertionHash,
	).Return(parentInfo, nil)
	heights := chainabstraction.OriginHeights{}
	mockChain.On(
		"TopLevelClaimHeights",
		ctx,
		edgeId,
	).Return(heights, nil)

	assertionUnrivaledBlocks := uint64(5)
	mockChain.On(
		"AssertionUnrivaledBlocks",
		ctx,
		assertionHash,
	).Return(assertionUnrivaledBlocks, nil)

	mockChallengeManager.On(
		"GetEdge", ctx, edgeId,
	).Return(option.Some(chainabstraction.SpecEdge(edge)), nil)

	edge.On("Id").Return(edgeId)
	edge.On("OriginId").Return(originId)
	edge.On("CreatedAtBlock").Return(uint64(0), nil)
	edge.On("ClaimId").Return(option.Some(chainabstraction.ClaimId(assertionHash.Hash)))
	edge.On("MutualId").Return(chainabstraction.MutualId{})
	edge.On("GetChallengeLevel").Return(chainabstraction.NewBlockChallengeLevel(), nil)
	edge.On("GetReversedChallengeLevel").Return(chainabstraction.ChallengeLevel(2), nil)
	startCommit := common.BytesToHash([]byte("nyan"))
	endCommit := common.BytesToHash([]byte("nyan2"))
	edge.On("StartCommitment").Return(chainabstraction.Height(0), startCommit)
	edge.On("EndCommitment").Return(chainabstraction.Height(4), endCommit)
	edge.On(
		"AssertionHash",
		ctx,
	).Return(assertionHash, nil)

	mockStateManager := &mocks.MockStateManager{}
	mockStateManager.On(
		"AgreesWithHistoryCommitment",
		ctx,
		chainabstraction.NewBlockChallengeLevel(),
		&layer2stateprovider.HistoryCommitmentRequest{
			AssertionMetadata:           simpleAssertionMetadata(),
			UpperChallengeOriginHeights: []layer2stateprovider.Height{},
			UpToHeight:                  option.Some[layer2stateprovider.Height](4),
		},
		layer2stateprovider.History{
			Height:     uint64(0),
			MerkleRoot: startCommit,
		},
	).Return(true, nil)
	mockStateManager.On(
		"AgreesWithHistoryCommitment",
		ctx,
		chainabstraction.NewBlockChallengeLevel(),
		&layer2stateprovider.HistoryCommitmentRequest{
			AssertionMetadata:           simpleAssertionMetadata(),
			UpperChallengeOriginHeights: []layer2stateprovider.Height{},
			UpToHeight:                  option.Some[layer2stateprovider.Height](4),
		},
		layer2stateprovider.History{
			Height:     uint64(4),
			MerkleRoot: endCommit,
		},
	).Return(true, nil)

	mockManager := &mocks.MockEdgeTracker{}
	mockManager.On("TrackEdge", ctx, mockHonest).Return(nil)

	watcher := &Watcher{
		challenges:       threadsafe.NewMap[chainabstraction.AssertionHash, *trackedChallenge](),
		histChecker:      mockStateManager,
		chain:            mockChain,
		edgeManager:      mockManager,
		numBigStepLevels: 1,
	}
	_, err := watcher.processEdgeAddedEvent(ctx, &challengeV2gen.EdgeChallengeManagerEdgeAdded{
		EdgeId:   edgeId.Hash,
		OriginId: assertionHash.Hash,
	})
	require.NoError(t, err)

	_, ok := watcher.challenges.TryGet(assertionHash)
	require.Equal(t, true, ok)
}

type mockHonestEdge struct {
	*mocks.MockSpecEdge
}

func (m *mockHonestEdge) Honest() {}

func (m *mockHonestEdge) Bisect(
	ctx context.Context,
	prefixHistoryRoot common.Hash,
	prefixProof []byte,
) (chainabstraction.VerifiedRoyalEdge, chainabstraction.VerifiedRoyalEdge, error) {
	return m.MockSpecEdge.Bisect(ctx, prefixHistoryRoot, prefixProof)
}

func (m *mockHonestEdge) ConfirmByTimer(ctx context.Context, claimedAssertion chainabstraction.AssertionHash) (*types.Transaction, error) {
	return m.MockSpecEdge.ConfirmByTimer(ctx, claimedAssertion)
}

func TestWatcher_AddVerifiedHonestEdge(t *testing.T) {
	ctx := context.Background()
	mockChain := &mocks.MockProtocol{}

	assertionHash := chainabstraction.AssertionHash{Hash: common.BytesToHash([]byte("foo"))}
	edgeId := chainabstraction.EdgeId{Hash: common.BytesToHash([]byte("bar"))}
	originId := chainabstraction.OriginId(common.BytesToHash([]byte("origin bar")))
	edge := &mocks.MockSpecEdge{}

	edge.On(
		"AssertionHash",
		ctx,
	).Return(assertionHash, nil)
	edge.On("Status", ctx).Return(chainabstraction.EdgePending, nil)
	edge.On("GetTotalChallengeLevels", ctx).Return(uint8(3), nil)
	edge.On("HasChildren", ctx).Return(false, nil)
	assertionUnrivaledBlocks := uint64(1)
	mockChain.On("AssertionUnrivaledBlocks", ctx, assertionHash).Return(assertionUnrivaledBlocks, nil)

	edge.On("Id").Return(edgeId)
	edge.On("OriginId").Return(originId)
	createdAt := uint64(5)
	edge.On("CreatedAtBlock").Return(createdAt, nil)
	edge.On("ClaimId").Return(option.Some(chainabstraction.ClaimId(assertionHash.Hash)))
	edge.On("OriginId").Return(chainabstraction.OriginId{})
	edge.On("MutualId").Return(chainabstraction.MutualId{})
	edge.On("GetChallengeLevel").Return(chainabstraction.NewBlockChallengeLevel(), nil)
	edge.On("GetReversedChallengeLevel").Return(chainabstraction.ChallengeLevel(2), nil)
	startCommit := common.BytesToHash([]byte("start"))
	endCommit := common.BytesToHash([]byte("start"))
	edge.On("StartCommitment").Return(chainabstraction.Height(0), startCommit)
	edge.On("EndCommitment").Return(chainabstraction.Height(32), endCommit)

	mockStateManager := &mocks.MockStateManager{}
	mockManager := &mocks.MockEdgeTracker{}
	honest := &mockHonestEdge{edge}
	mockManager.On("TrackEdge", ctx, honest).Return(nil)

	watcher := &Watcher{
		challenges:       threadsafe.NewMap[chainabstraction.AssertionHash, *trackedChallenge](),
		histChecker:      mockStateManager,
		chain:            mockChain,
		edgeManager:      mockManager,
		numBigStepLevels: 1,
	}

	err := watcher.AddVerifiedHonestEdge(ctx, honest)
	require.NoError(t, err)
	_, ok := watcher.challenges.TryGet(assertionHash)
	require.Equal(t, true, ok)
}
