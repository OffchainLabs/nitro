// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package challengetree

import (
	"context"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/chainabstraction"
	"github.com/offchainlabs/nitro/bold/challengemanager/challengetree/mock"
	"github.com/offchainlabs/nitro/bold/challengetesting/mocks"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/containers/threadsafe"
	"github.com/offchainlabs/nitro/bold/layer2stateprovider"
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

func TestAddEdge(t *testing.T) {
	ht := &RoyalChallengeTree{
		edges:                 threadsafe.NewMap[chainabstraction.EdgeId, chainabstraction.SpecEdge](),
		edgeCreationTimes:     threadsafe.NewMap[OriginPlusMutualId, *threadsafe.Map[chainabstraction.EdgeId, creationTime]](),
		royalRootEdgesByLevel: threadsafe.NewMap[chainabstraction.ChallengeLevel, *threadsafe.Slice[chainabstraction.SpecEdge]](),
		totalChallengeLevels:  3,
	}
	ht.topLevelAssertionHash = chainabstraction.AssertionHash{Hash: common.BytesToHash([]byte("foo"))}
	ctx := context.Background()
	edge := newEdge(&newCfg{t: t, edgeId: "blk-0.a-16.a", createdAt: 1, claimId: "foo"})

	t.Run("getting top level assertion errored", func(t *testing.T) {
		ht.metadataReader = &mockMetadataReader{
			assertionErr: errors.New("bad request"),
		}
		err := ht.AddEdge(ctx, edge)
		require.ErrorContains(t, err, "could not get top level assertion for edge")
	})
	t.Run("ignores if disagrees with top level assertion hash of edge", func(t *testing.T) {
		ht.metadataReader = &mockMetadataReader{
			assertionErr:  nil,
			assertionHash: chainabstraction.AssertionHash{Hash: common.BytesToHash([]byte("bar"))},
		}
		err := ht.AddEdge(ctx, edge)
		require.ErrorIs(t, err, ErrMismatchedChallengeAssertionHash)
	})
	t.Run("getting claim heights errored", func(t *testing.T) {
		ht.metadataReader = &mockMetadataReader{
			assertionErr:    nil,
			assertionHash:   ht.topLevelAssertionHash,
			claimHeightsErr: errors.New("bad request"),
		}
		ht.royalRootEdgesByLevel.Put(chainabstraction.ChallengeLevel(2), threadsafe.NewSlice[chainabstraction.SpecEdge]())
		honestBlockEdges := ht.royalRootEdgesByLevel.Get(chainabstraction.ChallengeLevel(2))
		honestBlockEdges.Push(edge)
		err := ht.AddEdge(ctx, edge)
		require.ErrorContains(t, err, "could not get claim heights for edge")
	})
	t.Run("checking if agrees with commit errored", func(t *testing.T) {
		ht.metadataReader = &mockMetadataReader{
			assertionErr:  nil,
			assertionHash: ht.topLevelAssertionHash,
		}
		start, startCommit := edge.StartCommitment()
		end, endCommit := edge.EndCommitment()
		mockStateManager := &mocks.MockStateManager{}
		mockStateManager.On(
			"AgreesWithHistoryCommitment",
			ctx,
			chainabstraction.NewBlockChallengeLevel(),
			&layer2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []layer2stateprovider.Height{},
				UpToHeight:                  option.Some(layer2stateprovider.Height(end)),
			},
			layer2stateprovider.History{
				Height:     uint64(start),
				MerkleRoot: startCommit,
			},
		).Return(false, errors.New("something went wrong"))
		mockStateManager.On(
			"AgreesWithHistoryCommitment",
			ctx,
			chainabstraction.NewBlockChallengeLevel(),
			&layer2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []layer2stateprovider.Height{},
				UpToHeight:                  option.Some(layer2stateprovider.Height(end)),
			},
			layer2stateprovider.History{
				Height:     uint64(end),
				MerkleRoot: endCommit,
			},
		).Return(false, errors.New("something went wrong"))
		ht.histChecker = mockStateManager
		err := ht.AddEdge(ctx, edge)
		require.ErrorContains(t, err, "could not check history commitment agreement")
	})
	t.Run("fully disagrees with edge", func(t *testing.T) {
		ht.metadataReader = &mockMetadataReader{
			assertionErr:  nil,
			assertionHash: ht.topLevelAssertionHash,
		}
		badEdge := newEdge(&newCfg{t: t, edgeId: "blk-0.f-16.a", createdAt: 1, claimId: "foo"})
		endHeight, endCommit := badEdge.EndCommitment()
		mockStateManager := &mocks.MockStateManager{}
		mockStateManager.On(
			"AgreesWithHistoryCommitment",
			ctx,
			chainabstraction.NewBlockChallengeLevel(),
			&layer2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []layer2stateprovider.Height{},
				UpToHeight:                  option.Some(layer2stateprovider.Height(endHeight)),
			},
			layer2stateprovider.History{
				Height:     uint64(endHeight),
				MerkleRoot: endCommit,
			},
		).Return(false, nil)
		ht.histChecker = mockStateManager
		err := ht.AddEdge(ctx, badEdge)
		require.NoError(t, err)
		_, ok := badEdge.AsVerifiedHonest()
		require.Equal(t, false, ok)

		// Check the edge is not kept track of in the honest edge, but we do track its mutual id.
		_, ok = ht.edges.TryGet(badEdge.Id())
		require.Equal(t, false, ok)
		key := buildEdgeCreationTimeKey(chainabstraction.OriginId{}, badEdge.MutualId())
		_, ok = ht.edgeCreationTimes.TryGet(key)
		require.Equal(t, true, ok)
	})
	t.Run("agrees with edge but is not royal", func(t *testing.T) {
		ht.metadataReader = &mockMetadataReader{
			assertionErr:  nil,
			assertionHash: ht.topLevelAssertionHash,
		}
		rootEdge := newEdge(&newCfg{t: t, edgeId: "blk-0.a-32.a", createdAt: 1, claimId: "foo"})
		ht.royalRootEdgesByLevel.Put(chainabstraction.ChallengeLevel(2), threadsafe.NewSlice[chainabstraction.SpecEdge]())
		honestBlockEdges := ht.royalRootEdgesByLevel.Get(chainabstraction.ChallengeLevel(2))
		honestBlockEdges.Push(rootEdge)

		edge := newEdge(&newCfg{t: t, edgeId: "blk-0.a-16.a", createdAt: 2})
		startHeight, startCommit := edge.StartCommitment()
		endHeight, endCommit := edge.EndCommitment()
		mockStateManager := &mocks.MockStateManager{}
		mockStateManager.On(
			"AgreesWithHistoryCommitment",
			ctx,
			chainabstraction.NewBlockChallengeLevel(),
			&layer2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []layer2stateprovider.Height{},
				UpToHeight:                  option.Some(layer2stateprovider.Height(endHeight)),
			},
			layer2stateprovider.History{
				Height:     uint64(startHeight),
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
				UpToHeight:                  option.Some(layer2stateprovider.Height(endHeight)),
			},
			layer2stateprovider.History{
				Height:     uint64(endHeight),
				MerkleRoot: endCommit,
			},
		).Return(true, nil)
		ht.histChecker = mockStateManager
		err := ht.AddEdge(ctx, edge)
		require.NoError(t, err)
		_, ok := edge.AsVerifiedHonest()
		require.Equal(t, false, ok)

		// Not tracked.
		_, ok = ht.edges.TryGet(edge.Id())
		require.Equal(t, false, ok)
		// However, exists in the mutual ids mapping.
		key := buildEdgeCreationTimeKey(chainabstraction.OriginId{}, edge.MutualId())
		_, ok = ht.edgeCreationTimes.TryGet(key)
		require.Equal(t, true, ok)

		// However, we should not have a level zero edge being tracked yet.
		blockChallengeEdges := ht.royalRootEdgesByLevel.Get(chainabstraction.ChallengeLevel(2))
		found := blockChallengeEdges.Find(func(_ int, e chainabstraction.SpecEdge) bool {
			return e.Id() == edge.Id()
		})
		require.Equal(t, false, found)
	})
	t.Run("agrees with edge and is a level zero edge", func(t *testing.T) {
		ht.metadataReader = &mockMetadataReader{
			assertionErr:  nil,
			assertionHash: ht.topLevelAssertionHash,
		}
		edge := newEdge(&newCfg{t: t, edgeId: "blk-0.a-32.a", createdAt: 1, claimId: "foo"})
		endHeight, endCommit := edge.EndCommitment()
		mockStateManager := &mocks.MockStateManager{}
		mockStateManager.On(
			"AgreesWithHistoryCommitment",
			ctx,
			chainabstraction.NewBlockChallengeLevel(),
			&layer2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []layer2stateprovider.Height{},
				UpToHeight:                  option.Some(layer2stateprovider.Height(endHeight)),
			},
			layer2stateprovider.History{
				Height:     uint64(endHeight),
				MerkleRoot: endCommit,
			},
		).Return(true, nil)
		ht.histChecker = mockStateManager
		err := ht.AddEdge(ctx, edge)
		require.NoError(t, err)

		// Exists.
		_, ok := ht.edges.TryGet(edge.Id())
		require.Equal(t, true, ok)
		// Exists in the mutual ids mapping.
		key := buildEdgeCreationTimeKey(chainabstraction.OriginId{}, edge.MutualId())
		_, ok = ht.edgeCreationTimes.TryGet(key)
		require.Equal(t, true, ok)

		// We should have a level zero edge being tracked.
		require.Equal(t, false, ht.royalRootEdgesByLevel.IsEmpty())
		_, ok = ht.royalRootEdgesByLevel.TryGet(chainabstraction.ChallengeLevel(2))
		require.Equal(t, true, ok)
	})
}

func TestAddHonestEdge(t *testing.T) {
	createdAt := uint64(1)
	edge := newEdge(&newCfg{t: t, edgeId: "big-0.a-32.a", createdAt: createdAt, claimId: "bar"})
	ht := &RoyalChallengeTree{
		edges:                 threadsafe.NewMap[chainabstraction.EdgeId, chainabstraction.SpecEdge](),
		edgeCreationTimes:     threadsafe.NewMap[OriginPlusMutualId, *threadsafe.Map[chainabstraction.EdgeId, creationTime]](),
		royalRootEdgesByLevel: threadsafe.NewMap[chainabstraction.ChallengeLevel, *threadsafe.Slice[chainabstraction.SpecEdge]](),
	}
	ht.topLevelAssertionHash = chainabstraction.AssertionHash{Hash: common.BytesToHash([]byte("foo"))}
	edge.MarkAsHonest()
	verifiedHonest, _ := edge.AsVerifiedHonest()
	err := ht.AddRoyalEdge(verifiedHonest)
	require.NoError(t, err)

	// We now check if the challenge tree has a populated
	// block challenge level zero edge.
	require.Equal(t, 1, ht.royalRootEdgesByLevel.Get(chainabstraction.ChallengeLevel(1)).Len())

	// Check if it exists in the mutual ids mapping.
	mutualId := edge.MutualId()
	key := buildEdgeCreationTimeKey(chainabstraction.OriginId{}, mutualId)
	mutuals, ok := ht.edgeCreationTimes.TryGet(key)
	require.Equal(t, true, ok)
	gotCreatedAt, ok := mutuals.TryGet(edge.Id())
	require.Equal(t, true, ok)
	require.Equal(t, createdAt, uint64(gotCreatedAt))

	// Does not add it again.
	err = ht.AddRoyalEdge(verifiedHonest)
	require.NoError(t, err)

	require.Equal(t, 1, ht.royalRootEdgesByLevel.Get(chainabstraction.ChallengeLevel(1)).Len())
}

type mockMetadataReader struct {
	assertionHash            chainabstraction.AssertionHash
	assertionErr             error
	claimHeights             chainabstraction.OriginHeights
	claimHeightsErr          error
	unrivaledAssertionBlocks uint64
	mockManager              *mocks.MockSpecChallengeManager
}

func (m *mockMetadataReader) TopLevelAssertion(
	_ context.Context, _ chainabstraction.EdgeId,
) (chainabstraction.AssertionHash, error) {
	return m.assertionHash, m.assertionErr
}

func (m *mockMetadataReader) AssertionUnrivaledBlocks(
	_ context.Context, _ chainabstraction.AssertionHash,
) (uint64, error) {
	return m.unrivaledAssertionBlocks, nil
}

func (m *mockMetadataReader) TopLevelClaimHeights(
	_ context.Context, _ chainabstraction.EdgeId,
) (chainabstraction.OriginHeights, error) {
	return m.claimHeights, m.claimHeightsErr
}

func (m *mockMetadataReader) SpecChallengeManager() chainabstraction.SpecChallengeManager {
	return m.mockManager
}
func (m *mockMetadataReader) ReadAssertionCreationInfo(
	_ context.Context, _ chainabstraction.AssertionHash,
) (*chainabstraction.AssertionCreatedInfo, error) {
	return &chainabstraction.AssertionCreatedInfo{InboxMaxCount: big.NewInt(1)}, nil
}

type newCfg struct {
	t         *testing.T
	originId  mock.OriginId
	edgeId    mock.EdgeId
	claimId   string
	createdAt uint64
}

func newEdge(cfg *newCfg) *mock.Edge {
	cfg.t.Helper()
	items := strings.Split(string(cfg.edgeId), "-")
	var typ chainabstraction.ChallengeLevel
	switch items[0] {
	case "blk":
		typ = 0
	case "big":
		typ = 1
	case "smol":
		typ = 2
	}
	startData := strings.Split(items[1], ".")
	startHeight, err := strconv.ParseUint(startData[0], 10, 64)
	require.NoError(cfg.t, err)
	startCommit := startData[1]

	endData := strings.Split(items[2], ".")
	endHeight, err := strconv.ParseUint(endData[0], 10, 64)
	require.NoError(cfg.t, err)
	endCommit := endData[1]

	return &mock.Edge{
		EdgeType:             typ,
		OriginID:             cfg.originId,
		ID:                   cfg.edgeId,
		StartHeight:          startHeight,
		ClaimID:              cfg.claimId,
		StartCommit:          mock.Commit(startCommit),
		EndHeight:            endHeight,
		EndCommit:            mock.Commit(endCommit),
		LowerChildID:         "",
		UpperChildID:         "",
		CreationBlock:        cfg.createdAt,
		TotalChallengeLevels: 3,
	}
}
