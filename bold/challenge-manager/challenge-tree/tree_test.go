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

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/challenge-manager/challenge-tree/mock"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/containers/threadsafe"
	"github.com/offchainlabs/nitro/bold/layer2-state-provider"
	"github.com/offchainlabs/nitro/bold/testing/mocks"
)

func simpleAssertionMetadata() *l2stateprovider.AssociatedAssertionMetadata {
	return &l2stateprovider.AssociatedAssertionMetadata{
		WasmModuleRoot: common.Hash{},
		FromState: protocol.GoGlobalState{
			Batch:      0,
			PosInBatch: 0,
		},
		BatchLimit: 1,
	}
}

func TestAddEdge(t *testing.T) {
	ht := &RoyalChallengeTree{
		edges:                 threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		edgeCreationTimes:     threadsafe.NewMap[OriginPlusMutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
		royalRootEdgesByLevel: threadsafe.NewMap[protocol.ChallengeLevel, *threadsafe.Slice[protocol.SpecEdge]](),
		totalChallengeLevels:  3,
	}
	ht.topLevelAssertionHash = protocol.AssertionHash{Hash: common.BytesToHash([]byte("foo"))}
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
			assertionHash: protocol.AssertionHash{Hash: common.BytesToHash([]byte("bar"))},
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
		ht.royalRootEdgesByLevel.Put(protocol.ChallengeLevel(2), threadsafe.NewSlice[protocol.SpecEdge]())
		honestBlockEdges := ht.royalRootEdgesByLevel.Get(protocol.ChallengeLevel(2))
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
			protocol.NewBlockChallengeLevel(),
			&l2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				UpToHeight:                  option.Some(l2stateprovider.Height(end)),
			},
			l2stateprovider.History{
				Height:     uint64(start),
				MerkleRoot: startCommit,
			},
		).Return(false, errors.New("something went wrong"))
		mockStateManager.On(
			"AgreesWithHistoryCommitment",
			ctx,
			protocol.NewBlockChallengeLevel(),
			&l2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				UpToHeight:                  option.Some(l2stateprovider.Height(end)),
			},
			l2stateprovider.History{
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
			protocol.NewBlockChallengeLevel(),
			&l2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				UpToHeight:                  option.Some(l2stateprovider.Height(endHeight)),
			},
			l2stateprovider.History{
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
		key := buildEdgeCreationTimeKey(protocol.OriginId{}, badEdge.MutualId())
		_, ok = ht.edgeCreationTimes.TryGet(key)
		require.Equal(t, true, ok)
	})
	t.Run("agrees with edge but is not royal", func(t *testing.T) {
		ht.metadataReader = &mockMetadataReader{
			assertionErr:  nil,
			assertionHash: ht.topLevelAssertionHash,
		}
		rootEdge := newEdge(&newCfg{t: t, edgeId: "blk-0.a-32.a", createdAt: 1, claimId: "foo"})
		ht.royalRootEdgesByLevel.Put(protocol.ChallengeLevel(2), threadsafe.NewSlice[protocol.SpecEdge]())
		honestBlockEdges := ht.royalRootEdgesByLevel.Get(protocol.ChallengeLevel(2))
		honestBlockEdges.Push(rootEdge)

		edge := newEdge(&newCfg{t: t, edgeId: "blk-0.a-16.a", createdAt: 2})
		startHeight, startCommit := edge.StartCommitment()
		endHeight, endCommit := edge.EndCommitment()
		mockStateManager := &mocks.MockStateManager{}
		mockStateManager.On(
			"AgreesWithHistoryCommitment",
			ctx,
			protocol.NewBlockChallengeLevel(),
			&l2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				UpToHeight:                  option.Some(l2stateprovider.Height(endHeight)),
			},
			l2stateprovider.History{
				Height:     uint64(startHeight),
				MerkleRoot: startCommit,
			},
		).Return(true, nil)
		mockStateManager.On(
			"AgreesWithHistoryCommitment",
			ctx,
			protocol.NewBlockChallengeLevel(),
			&l2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				UpToHeight:                  option.Some(l2stateprovider.Height(endHeight)),
			},
			l2stateprovider.History{
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
		key := buildEdgeCreationTimeKey(protocol.OriginId{}, edge.MutualId())
		_, ok = ht.edgeCreationTimes.TryGet(key)
		require.Equal(t, true, ok)

		// However, we should not have a level zero edge being tracked yet.
		blockChallengeEdges := ht.royalRootEdgesByLevel.Get(protocol.ChallengeLevel(2))
		found := blockChallengeEdges.Find(func(_ int, e protocol.SpecEdge) bool {
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
			protocol.NewBlockChallengeLevel(),
			&l2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           simpleAssertionMetadata(),
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				UpToHeight:                  option.Some(l2stateprovider.Height(endHeight)),
			},
			l2stateprovider.History{
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
		key := buildEdgeCreationTimeKey(protocol.OriginId{}, edge.MutualId())
		_, ok = ht.edgeCreationTimes.TryGet(key)
		require.Equal(t, true, ok)

		// We should have a level zero edge being tracked.
		require.Equal(t, false, ht.royalRootEdgesByLevel.IsEmpty())
		_, ok = ht.royalRootEdgesByLevel.TryGet(protocol.ChallengeLevel(2))
		require.Equal(t, true, ok)
	})
}

func TestAddHonestEdge(t *testing.T) {
	createdAt := uint64(1)
	edge := newEdge(&newCfg{t: t, edgeId: "big-0.a-32.a", createdAt: createdAt, claimId: "bar"})
	ht := &RoyalChallengeTree{
		edges:                 threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		edgeCreationTimes:     threadsafe.NewMap[OriginPlusMutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
		royalRootEdgesByLevel: threadsafe.NewMap[protocol.ChallengeLevel, *threadsafe.Slice[protocol.SpecEdge]](),
	}
	ht.topLevelAssertionHash = protocol.AssertionHash{Hash: common.BytesToHash([]byte("foo"))}
	edge.MarkAsHonest()
	verifiedHonest, _ := edge.AsVerifiedHonest()
	err := ht.AddRoyalEdge(verifiedHonest)
	require.NoError(t, err)

	// We now check if the challenge tree has a populated
	// block challenge level zero edge.
	require.Equal(t, 1, ht.royalRootEdgesByLevel.Get(protocol.ChallengeLevel(1)).Len())

	// Check if it exists in the mutual ids mapping.
	mutualId := edge.MutualId()
	key := buildEdgeCreationTimeKey(protocol.OriginId{}, mutualId)
	mutuals, ok := ht.edgeCreationTimes.TryGet(key)
	require.Equal(t, true, ok)
	gotCreatedAt, ok := mutuals.TryGet(edge.Id())
	require.Equal(t, true, ok)
	require.Equal(t, createdAt, uint64(gotCreatedAt))

	// Does not add it again.
	err = ht.AddRoyalEdge(verifiedHonest)
	require.NoError(t, err)

	require.Equal(t, 1, ht.royalRootEdgesByLevel.Get(protocol.ChallengeLevel(1)).Len())
}

type mockMetadataReader struct {
	assertionHash            protocol.AssertionHash
	assertionErr             error
	claimHeights             protocol.OriginHeights
	claimHeightsErr          error
	unrivaledAssertionBlocks uint64
	mockManager              *mocks.MockSpecChallengeManager
}

func (m *mockMetadataReader) TopLevelAssertion(
	_ context.Context, _ protocol.EdgeId,
) (protocol.AssertionHash, error) {
	return m.assertionHash, m.assertionErr
}

func (m *mockMetadataReader) AssertionUnrivaledBlocks(
	_ context.Context, _ protocol.AssertionHash,
) (uint64, error) {
	return m.unrivaledAssertionBlocks, nil
}

func (m *mockMetadataReader) TopLevelClaimHeights(
	_ context.Context, _ protocol.EdgeId,
) (protocol.OriginHeights, error) {
	return m.claimHeights, m.claimHeightsErr
}

func (m *mockMetadataReader) SpecChallengeManager() protocol.SpecChallengeManager {
	return m.mockManager
}
func (m *mockMetadataReader) ReadAssertionCreationInfo(
	_ context.Context, _ protocol.AssertionHash,
) (*protocol.AssertionCreatedInfo, error) {
	return &protocol.AssertionCreatedInfo{InboxMaxCount: big.NewInt(1)}, nil
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
	var typ protocol.ChallengeLevel
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
