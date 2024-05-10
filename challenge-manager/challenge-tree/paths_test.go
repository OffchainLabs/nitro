package challengetree

import (
	"context"
	"testing"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/challenge-manager/challenge-tree/mock"
	"github.com/OffchainLabs/bold/containers/threadsafe"
	"github.com/stretchr/testify/require"
)

func TestComputePathWeight(t *testing.T) {
	ctx := context.Background()
	ht := &RoyalChallengeTree{
		edges: threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
	}
	t.Run("edges not found", func(t *testing.T) {
		unseenEdge := newEdge(&newCfg{t: t, edgeId: "blk-0.a-4.a", createdAt: 4})
		unseenAncestor := newEdge(&newCfg{t: t, edgeId: "blk-0.a-8.a", createdAt: 2})
		_, err := ht.ComputePathWeight(
			ctx,
			ComputePathWeightArgs{
				Child:    unseenEdge.Id(),
				Ancestor: unseenAncestor.Id(),
				BlockNum: 10,
			},
		)
		require.ErrorContains(t, err, "child edge not yet tracked")
		ht.edges.Put(unseenEdge.Id(), unseenEdge)
		_, err = ht.ComputePathWeight(
			ctx,
			ComputePathWeightArgs{
				Child:    unseenEdge.Id(),
				Ancestor: unseenAncestor.Id(),
				BlockNum: 10,
			},
		)
		require.ErrorContains(t, err, "ancestor not yet tracked")
	})
	// To see the relationship between the edges, their creation times,
	// and the time they became rivaled, see the setupEssentialPathsTest function.
	tree, honestEdges := setupEssentialPathsTest(t)
	tree.royalRootEdgesByLevel.Put(2, threadsafe.NewSlice[protocol.SpecEdge]())
	tree.royalRootEdgesByLevel.Put(1, threadsafe.NewSlice[protocol.SpecEdge]())
	tree.royalRootEdgesByLevel.Put(0, threadsafe.NewSlice[protocol.SpecEdge]())

	blockRootEdges := tree.royalRootEdgesByLevel.Get(2 /* big step level */)
	blockRootEdges.Push(tree.edges.Get(id("blk-0.a-4.a")))
	bigStepRootEdges := tree.royalRootEdgesByLevel.Get(1 /* big step level */)
	bigStepRootEdges.Push(tree.edges.Get(id("big-0.a-4.a")))

	t.Run("length 0 path", func(t *testing.T) {
		child := protocol.SpecEdge(honestEdges["blk-2.a-4.a"])
		ancestor := child
		weight, err := tree.ComputePathWeight(
			ctx,
			ComputePathWeightArgs{
				Child:    child.Id(),
				Ancestor: ancestor.Id(),
				BlockNum: 10,
			},
		)
		require.NoError(t, err)
		require.Equal(t, uint64(1), weight)

		// Querying at a future block number should not change the result,
		// as the terminal node is rivaled.
		weight, err = tree.ComputePathWeight(
			ctx,
			ComputePathWeightArgs{
				Child:    child.Id(),
				Ancestor: ancestor.Id(),
				BlockNum: 20,
			},
		)
		require.NoError(t, err)
		require.Equal(t, uint64(1), weight)
	})

	t.Run("length 1 path with rivaled terminal", func(t *testing.T) {
		child := protocol.SpecEdge(honestEdges["blk-2.a-4.a"])
		ancestor := protocol.SpecEdge(honestEdges["blk-0.a-4.a"])
		weight, err := tree.ComputePathWeight(
			ctx,
			ComputePathWeightArgs{
				Child:    child.Id(),
				Ancestor: ancestor.Id(),
				BlockNum: 10,
			},
		)
		require.NoError(t, err)
		require.Equal(t, uint64(2), weight)

		// Querying at a future block number should not change the result,
		// as the terminal node is rivaled.
		weight, err = tree.ComputePathWeight(
			ctx,
			ComputePathWeightArgs{
				Child:    child.Id(),
				Ancestor: ancestor.Id(),
				BlockNum: 20,
			},
		)
		require.NoError(t, err)
		require.Equal(t, uint64(2), weight)
	})
	t.Run("length 2 path with unrivaled terminal", func(t *testing.T) {
		child := protocol.SpecEdge(honestEdges["blk-0.a-2.a"])
		ancestor := protocol.SpecEdge(honestEdges["blk-0.a-4.a"])
		weight, err := tree.ComputePathWeight(
			ctx,
			ComputePathWeightArgs{
				Child:    child.Id(),
				Ancestor: ancestor.Id(),
				BlockNum: 10,
			},
		)
		require.NoError(t, err)
		require.Equal(t, uint64(8), weight)

		isUnrivaled, err := tree.IsUnrivaledAtBlockNum(child, 20)
		require.NoError(t, err)
		require.True(t, isUnrivaled)

		// Should increase if we query at a future block number,
		// as in the tree setup, blk-3.a-4.a is unrivaled.
		weight, err = tree.ComputePathWeight(
			ctx,
			ComputePathWeightArgs{
				Child:    child.Id(),
				Ancestor: ancestor.Id(),
				BlockNum: 20,
			},
		)
		require.NoError(t, err)
		require.Equal(t, uint64(18), weight)
	})
	t.Run("length 3 path with unrivaled terminal", func(t *testing.T) {
		child := protocol.SpecEdge(honestEdges["blk-3.a-4.a"])
		ancestor := protocol.SpecEdge(honestEdges["blk-0.a-4.a"])
		weight, err := tree.ComputePathWeight(
			ctx,
			ComputePathWeightArgs{
				Child:    child.Id(),
				Ancestor: ancestor.Id(),
				BlockNum: 10,
			},
		)
		require.NoError(t, err)
		require.Equal(t, uint64(7), weight)

		isUnrivaled, err := tree.IsUnrivaledAtBlockNum(child, 20)
		require.NoError(t, err)
		require.True(t, isUnrivaled)

		// Should increase if we query at a future block number,
		// as in the tree setup, blk-3.a-4.a is unrivaled.
		weight, err = tree.ComputePathWeight(
			ctx,
			ComputePathWeightArgs{
				Child:    child.Id(),
				Ancestor: ancestor.Id(),
				BlockNum: 20,
			},
		)
		require.NoError(t, err)
		require.Equal(t, uint64(17), weight)
	})
	t.Run("path ending in refinement node across challenge level", func(t *testing.T) {
		child := protocol.SpecEdge(honestEdges["big-0.a-4.a"])
		ancestor := protocol.SpecEdge(honestEdges["blk-0.a-4.a"])
		weight, err := tree.ComputePathWeight(
			ctx,
			ComputePathWeightArgs{
				Child:    child.Id(),
				Ancestor: ancestor.Id(),
				BlockNum: 10,
			},
		)
		require.NoError(t, err)
		require.Equal(t, uint64(6), weight)

		isUnrivaled, err := tree.IsUnrivaledAtBlockNum(child, 20)
		require.NoError(t, err)
		require.True(t, isUnrivaled)

		// Should increase if we query at a future block number,
		// as in the tree setup, blk-3.a-4.a is unrivaled.
		weight, err = tree.ComputePathWeight(
			ctx,
			ComputePathWeightArgs{
				Child:    child.Id(),
				Ancestor: ancestor.Id(),
				BlockNum: 20,
			},
		)
		require.NoError(t, err)
		require.Equal(t, uint64(16), weight)
	})
}

// Set up a challenge tree, down to two challenge levels.
func setupEssentialPathsTest(t *testing.T) (*RoyalChallengeTree, map[mock.EdgeId]*mock.Edge) {
	t.Helper()
	tree := &RoyalChallengeTree{
		edges:                 threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		edgeCreationTimes:     threadsafe.NewMap[OriginPlusMutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
		metadataReader:        &mockMetadataReader{},
		totalChallengeLevels:  3,
		royalRootEdgesByLevel: threadsafe.NewMap[protocol.ChallengeLevel, *threadsafe.Slice[protocol.SpecEdge]](),
	}
	tree.royalRootEdgesByLevel.Put(2, threadsafe.NewSlice[protocol.SpecEdge]())
	tree.royalRootEdgesByLevel.Put(1, threadsafe.NewSlice[protocol.SpecEdge]())
	tree.royalRootEdgesByLevel.Put(0, threadsafe.NewSlice[protocol.SpecEdge]())
	honestAssertion := "assertion.a"
	evilAssertion := "assertion.b"
	honestEdges := buildEdges(
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-4.a", claimId: honestAssertion, createdAt: 1}),
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-2.a", createdAt: 3}),
		newEdge(&newCfg{t: t, edgeId: "blk-2.a-4.a", createdAt: 3}),
		newEdge(&newCfg{t: t, edgeId: "blk-2.a-3.a", createdAt: 5}),
		newEdge(&newCfg{t: t, edgeId: "blk-3.a-4.a", createdAt: 5}),
		newEdge(&newCfg{t: t, edgeId: "big-0.a-4.a", claimId: "blk-2.a-3.a", createdAt: 7}),
	)
	evilEdges := buildEdges(
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-4.b", claimId: evilAssertion, createdAt: 2}),
		newEdge(&newCfg{t: t, edgeId: "blk-2.a-4.b", createdAt: 4}),
		newEdge(&newCfg{t: t, edgeId: "blk-2.a-3.b", createdAt: 6}),
		newEdge(&newCfg{t: t, edgeId: "blk-3.b-4.b", createdAt: 6}),
		newEdge(&newCfg{t: t, edgeId: "big-0.a-4.b", claimId: "blk-2.a-3.b", createdAt: 8}),
	)

	// Child-relationship linking.
	// Honest.
	honestEdges["blk-0.a-4.a"].LowerChildID = "blk-0.a-2.a"
	honestEdges["blk-0.a-4.a"].UpperChildID = "blk-2.a-4.a"
	honestEdges["blk-2.a-4.a"].LowerChildID = "blk-2.a-3.a"
	honestEdges["blk-2.a-4.a"].UpperChildID = "blk-3.a-4.a"
	// Evil.
	evilEdges["blk-0.a-4.b"].LowerChildID = "blk-0.a-2.a"
	evilEdges["blk-0.a-4.b"].UpperChildID = "blk-2.a-4.b"
	evilEdges["blk-2.a-4.b"].LowerChildID = "blk-2.a-3.b"
	evilEdges["blk-2.a-4.b"].UpperChildID = "blk-3.b-4.b"

	transformedEdges := make(map[protocol.EdgeId]protocol.SpecEdge)
	for _, v := range honestEdges {
		transformedEdges[v.Id()] = v
	}
	allEdges := threadsafe.NewMapFromItems(transformedEdges)
	tree.edges = allEdges

	// Set up rivaled edges.
	mutual := honestEdges["blk-0.a-4.a"].MutualId()
	key := buildEdgeCreationTimeKey(protocol.OriginId{}, mutual)
	tree.edgeCreationTimes.Put(key, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals := tree.edgeCreationTimes.Get(key)
	a := honestEdges["blk-0.a-4.a"]
	b := evilEdges["blk-0.a-4.b"]
	aCreation, err := a.CreatedAtBlock()
	require.NoError(t, err)
	bCreation, err := b.CreatedAtBlock()
	require.NoError(t, err)
	mutuals.Put(a.Id(), creationTime(aCreation))
	mutuals.Put(b.Id(), creationTime(bCreation))

	mutual = honestEdges["blk-2.a-4.a"].MutualId()
	key = buildEdgeCreationTimeKey(protocol.OriginId{}, mutual)
	tree.edgeCreationTimes.Put(key, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.edgeCreationTimes.Get(key)
	a = honestEdges["blk-2.a-4.a"]
	b = evilEdges["blk-2.a-4.b"]
	aCreation, err = a.CreatedAtBlock()
	require.NoError(t, err)
	bCreation, err = b.CreatedAtBlock()
	require.NoError(t, err)
	mutuals.Put(a.Id(), creationTime(aCreation))
	mutuals.Put(b.Id(), creationTime(bCreation))

	mutual = honestEdges["blk-2.a-3.a"].MutualId()
	key = buildEdgeCreationTimeKey(protocol.OriginId{}, mutual)
	tree.edgeCreationTimes.Put(key, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.edgeCreationTimes.Get(key)
	a = honestEdges["blk-2.a-3.a"]
	b = evilEdges["blk-2.a-3.b"]
	aCreation, err = a.CreatedAtBlock()
	require.NoError(t, err)
	bCreation, err = b.CreatedAtBlock()
	require.NoError(t, err)
	mutuals.Put(a.Id(), creationTime(aCreation))
	mutuals.Put(b.Id(), creationTime(bCreation))

	originId := protocol.OriginId(honestEdges["blk-2.a-3.a"].MutualId())
	mutual = honestEdges["big-0.a-4.a"].MutualId()
	key = buildEdgeCreationTimeKey(originId, mutual)
	tree.edgeCreationTimes.Put(key, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.edgeCreationTimes.Get(key)
	a = honestEdges["big-0.a-4.a"]
	b = evilEdges["big-0.a-4.b"]
	aCreation, err = a.CreatedAtBlock()
	require.NoError(t, err)
	bCreation, err = b.CreatedAtBlock()
	require.NoError(t, err)
	mutuals.Put(a.Id(), creationTime(aCreation))
	mutuals.Put(b.Id(), creationTime(bCreation))
	return tree, honestEdges
}

func Test_isProofNode(t *testing.T) {
	ctx := context.Background()
	edge := newEdge(&newCfg{t: t, edgeId: "blk-0.a-32.a"})
	require.Equal(t, false, isProofNode(ctx, edge))
	edge = newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.a"})
	require.Equal(t, false, isProofNode(ctx, edge))
	edge = newEdge(&newCfg{t: t, edgeId: "smol-0.a-2.a"})
	require.Equal(t, false, isProofNode(ctx, edge))
	edge = newEdge(&newCfg{t: t, edgeId: "smol-0.a-1.a"})
	require.Equal(t, true, isProofNode(ctx, edge))
}

func Test_isClaimedEdge(t *testing.T) {
	ctx := context.Background()
	ht := &RoyalChallengeTree{
		edges: threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
	}
	edge := newEdge(&newCfg{t: t, edgeId: "blk-0.a-32.a"})
	ok, _ := ht.isClaimedEdge(ctx, edge)
	require.False(t, ok)

	edge = newEdge(&newCfg{t: t, edgeId: "smol-0.a-1.a"})
	ok, _ = ht.isClaimedEdge(ctx, edge)
	require.False(t, ok)

	edge = newEdge(&newCfg{t: t, edgeId: "smol-0.a-1.a"})
	ok, _ = ht.isClaimedEdge(ctx, edge)
	require.False(t, ok)

	claimedEdge := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.a"})
	claimingEdge := newEdge(&newCfg{t: t, edgeId: "big-0.a-32.a", claimId: "blk-0.a-1.a"})
	ht.edges.Put(claimedEdge.Id(), claimedEdge)
	ht.edges.Put(claimingEdge.Id(), claimingEdge)

	ok, expectedClaiming := ht.isClaimedEdge(ctx, claimedEdge)
	require.True(t, ok)
	require.Equal(t, expectedClaiming.Id(), claimingEdge.Id())
}
