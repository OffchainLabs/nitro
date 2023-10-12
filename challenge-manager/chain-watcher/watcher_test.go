// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package watcher

import (
	"context"
	"math/big"
	"testing"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	challengetree "github.com/OffchainLabs/bold/challenge-manager/challenge-tree"
	"github.com/OffchainLabs/bold/containers/option"
	"github.com/OffchainLabs/bold/containers/threadsafe"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/bold/testing/mocks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestWatcher_processEdgeConfirmation(t *testing.T) {
	ctx := context.Background()
	mockChain := &mocks.MockProtocol{}
	mockChallengeManager := &mocks.MockSpecChallengeManager{}
	mockChain.On(
		"SpecChallengeManager",
		ctx,
	).Return(mockChallengeManager, nil)

	assertionHash := protocol.AssertionHash{Hash: common.BytesToHash([]byte("foo"))}
	edgeId := protocol.EdgeId{Hash: common.BytesToHash([]byte("bar"))}
	edge := &mocks.MockSpecEdge{}

	mockChallengeManager.On(
		"GetEdge", ctx, edgeId,
	).Return(option.Some(protocol.SpecEdge(edge)), nil)

	edge.On("ClaimId").Return(option.Some(protocol.ClaimId(assertionHash.Hash)))
	edge.On("Id").Return(edgeId)
	edge.On("GetChallengeLevel").Return(protocol.ChallengeLevel(1), nil)
	edge.On(
		"AssertionHash",
		ctx,
	).Return(assertionHash, nil)

	watcher := &Watcher{
		challenges: threadsafe.NewMap[protocol.AssertionHash, *trackedChallenge](),
		chain:      mockChain,
	}
	watcher.challenges.Put(assertionHash, &trackedChallenge{
		confirmedLevelZeroEdgeClaimIds: threadsafe.NewMap[protocol.ClaimId, protocol.EdgeId](),
	})

	err := watcher.processEdgeConfirmation(ctx, edgeId)
	require.NoError(t, err)

	chal, ok := watcher.challenges.TryGet(assertionHash)
	require.Equal(t, true, ok)
	ok = chal.confirmedLevelZeroEdgeClaimIds.Has(protocol.ClaimId(assertionHash.Hash))
	require.Equal(t, true, ok)
}

func TestWatcher_processEdgeAddedEvent(t *testing.T) {
	ctx := context.Background()
	mockChain := &mocks.MockProtocol{}
	mockChallengeManager := &mocks.MockSpecChallengeManager{}
	mockChain.On(
		"SpecChallengeManager",
		ctx,
	).Return(mockChallengeManager, nil)

	assertionHash := protocol.AssertionHash{Hash: common.BytesToHash([]byte("foo"))}
	parentAssertionHash := protocol.AssertionHash{Hash: common.BytesToHash([]byte("parent foo"))}
	edgeId := protocol.EdgeId{Hash: common.BytesToHash([]byte("bar"))}
	originId := protocol.OriginId(common.BytesToHash([]byte("origin bar")))
	edge := &mocks.MockSpecEdge{}

	mockChain.On(
		"TopLevelAssertion",
		ctx,
		edgeId,
	).Return(assertionHash, nil)

	info := &protocol.AssertionCreatedInfo{
		InboxMaxCount:       big.NewInt(1),
		ParentAssertionHash: parentAssertionHash.Hash,
	}
	mockChain.On(
		"ReadAssertionCreationInfo",
		ctx,
		assertionHash,
	).Return(info, nil)
	parentInfo := &protocol.AssertionCreatedInfo{
		InboxMaxCount: big.NewInt(1),
	}
	mockChain.On(
		"ReadAssertionCreationInfo",
		ctx,
		parentAssertionHash,
	).Return(parentInfo, nil)
	heights := protocol.OriginHeights{}
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
	).Return(option.Some(protocol.SpecEdge(edge)), nil)

	edge.On("Id").Return(edgeId)
	edge.On("OriginId").Return(originId)
	edge.On("CreatedAtBlock").Return(uint64(0), nil)
	edge.On("ClaimId").Return(option.Some(protocol.ClaimId(assertionHash.Hash)))
	edge.On("MutualId").Return(protocol.MutualId{})
	edge.On("GetChallengeLevel").Return(protocol.NewBlockChallengeLevel(), nil)
	edge.On("GetReversedChallengeLevel").Return(protocol.ChallengeLevel(2), nil)
	startCommit := common.BytesToHash([]byte("nyan"))
	endCommit := common.BytesToHash([]byte("nyan2"))
	edge.On("StartCommitment").Return(protocol.Height(0), startCommit)
	edge.On("EndCommitment").Return(protocol.Height(4), endCommit)
	edge.On(
		"AssertionHash",
		ctx,
	).Return(assertionHash, nil)

	mockStateManager := &mocks.MockStateManager{}
	mockStateManager.On(
		"AgreesWithHistoryCommitment",
		ctx,
		protocol.NewBlockChallengeLevel(),
		&l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     0,
			UpperChallengeOriginHeights: []l2stateprovider.Height{},
			FromHeight:                  0,
			UpToHeight:                  option.Some[l2stateprovider.Height](4),
		},
		l2stateprovider.History{
			Height:     uint64(0),
			MerkleRoot: startCommit,
		},
	).Return(true, nil)
	mockStateManager.On(
		"AgreesWithHistoryCommitment",
		ctx,
		protocol.NewBlockChallengeLevel(),
		&l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     0,
			UpperChallengeOriginHeights: []l2stateprovider.Height{},
			FromHeight:                  0,
			UpToHeight:                  option.Some[l2stateprovider.Height](4),
		},
		l2stateprovider.History{
			Height:     uint64(4),
			MerkleRoot: endCommit,
		},
	).Return(true, nil)

	mockManager := &mocks.MockEdgeTracker{}
	mockManager.On("TrackEdge", ctx, edge).Return(nil)

	watcher := &Watcher{
		challenges:  threadsafe.NewMap[protocol.AssertionHash, *trackedChallenge](),
		histChecker: mockStateManager,
		chain:       mockChain,
		edgeManager: mockManager,
	}
	err := watcher.processEdgeAddedEvent(ctx, &challengeV2gen.EdgeChallengeManagerEdgeAdded{
		EdgeId:   edgeId.Hash,
		OriginId: assertionHash.Hash,
	})
	require.NoError(t, err)

	chal, ok := watcher.challenges.TryGet(assertionHash)
	require.Equal(t, true, ok)

	// Expect it to exist and be unrivaled for 10 blocks if we query at block number = 10,
	// plus the number of blocks the top level assertion was unrivaled (5).
	blockNumber := uint64(10)
	resp, err := chal.honestEdgeTree.ComputeAncestorsWithTimers(ctx, edgeId, blockNumber)
	require.NoError(t, err)
	pathTimer, err := chal.honestEdgeTree.ComputeHonestPathTimer(ctx, edgeId, resp.AncestorLocalTimers, blockNumber)
	require.NoError(t, err)
	require.Equal(t, pathTimer, challengetree.PathTimer(blockNumber+assertionUnrivaledBlocks))
}

type mockHonestEdge struct {
	protocol.SpecEdge
}

func (m *mockHonestEdge) Honest() {}

func TestWatcher_AddVerifiedHonestEdge(t *testing.T) {
	ctx := context.Background()
	mockChain := &mocks.MockProtocol{}

	assertionHash := protocol.AssertionHash{Hash: common.BytesToHash([]byte("foo"))}
	edgeId := protocol.EdgeId{Hash: common.BytesToHash([]byte("bar"))}
	originId := protocol.OriginId(common.BytesToHash([]byte("origin bar")))
	edge := &mocks.MockSpecEdge{}

	edge.On(
		"AssertionHash",
		ctx,
	).Return(assertionHash, nil)
	assertionUnrivaledBlocks := uint64(1)
	mockChain.On("AssertionUnrivaledBlocks", ctx, assertionHash).Return(assertionUnrivaledBlocks, nil)

	edge.On("Id").Return(edgeId)
	edge.On("OriginId").Return(originId)
	createdAt := uint64(5)
	edge.On("CreatedAtBlock").Return(createdAt, nil)
	edge.On("ClaimId").Return(option.Some(protocol.ClaimId(assertionHash.Hash)))
	edge.On("OriginId").Return(protocol.OriginId{})
	edge.On("MutualId").Return(protocol.MutualId{})
	edge.On("GetChallengeLevel").Return(protocol.NewBlockChallengeLevel(), nil)
	edge.On("GetReversedChallengeLevel").Return(protocol.ChallengeLevel(2), nil)
	startCommit := common.BytesToHash([]byte("start"))
	endCommit := common.BytesToHash([]byte("start"))
	edge.On("StartCommitment").Return(protocol.Height(0), startCommit)
	edge.On("EndCommitment").Return(protocol.Height(32), endCommit)

	mockStateManager := &mocks.MockStateManager{}
	mockManager := &mocks.MockEdgeTracker{}
	honest := &mockHonestEdge{edge}
	mockManager.On("TrackEdge", ctx, honest).Return(nil)

	watcher := &Watcher{
		challenges:  threadsafe.NewMap[protocol.AssertionHash, *trackedChallenge](),
		histChecker: mockStateManager,
		chain:       mockChain,
		edgeManager: mockManager,
	}

	err := watcher.AddVerifiedHonestEdge(ctx, honest)
	require.NoError(t, err)
	chal, ok := watcher.challenges.TryGet(assertionHash)
	require.Equal(t, true, ok)
	blockNum := uint64(20)
	resp, err := chal.honestEdgeTree.ComputeAncestorsWithTimers(ctx, edgeId, blockNum)
	require.NoError(t, err)
	pathTimer, err := chal.honestEdgeTree.ComputeHonestPathTimer(ctx, edgeId, resp.AncestorLocalTimers, blockNum)
	require.NoError(t, err)
	require.Equal(t, blockNum-createdAt+assertionUnrivaledBlocks, uint64(pathTimer))
}
