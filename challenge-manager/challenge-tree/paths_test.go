package challengetree

import (
	"context"
	"testing"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/challenge-manager/challenge-tree/mock"
	"github.com/OffchainLabs/bold/containers/threadsafe"
	"github.com/stretchr/testify/require"
)

func TestIsConfirmableEssentialNode(t *testing.T) {
	ctx := context.Background()
	tree, honestEdges := setupEssentialPathsTest(t)

	// Calculate the essential paths starting at the honest root.
	// See setupEssentialPaths and Test_findEssentialpaths below to
	// understand the setup of the challenge tree.
	_, _, _, err := tree.IsConfirmableEssentialNode(
		ctx,
		isConfirmableArgs{
			essentialNode: protocol.EdgeId{},
		},
	)
	require.ErrorContains(t, err, "essential node not found")

	// Based on the setup, we expect the minimum path weight at block number 10
	// to be 6, which at a confirmation threshold of 10 is not enough to confirm
	// the essential node.
	essentialHonestRoot := protocol.SpecEdge(honestEdges["blk-0.a-4.a"])
	blockNum := uint64(10)
	isConfirmable, _, minPathWeight, err := tree.IsConfirmableEssentialNode(
		ctx,
		isConfirmableArgs{
			confirmationThreshold: 10,
			essentialNode:         essentialHonestRoot.Id(),
			blockNum:              blockNum,
		},
	)
	require.NoError(t, err)
	require.False(t, isConfirmable)
	require.Equal(t, uint64(6), minPathWeight)

	// At block number 14, we should exactly meet the confirmation threshold.
	essentialHonestRoot = protocol.SpecEdge(honestEdges["blk-0.a-4.a"])
	blockNum = uint64(14)
	isConfirmable, _, minPathWeight, err = tree.IsConfirmableEssentialNode(
		ctx,
		isConfirmableArgs{
			confirmationThreshold: 10,
			essentialNode:         essentialHonestRoot.Id(),
			blockNum:              blockNum,
		},
	)
	require.NoError(t, err)
	require.True(t, isConfirmable)
	require.Equal(t, uint64(10), minPathWeight)
}

func Test_findEssentialPaths(t *testing.T) {
	ctx := context.Background()
	tree, honestEdges := setupEssentialPathsTest(t)

	// Calculate the essential paths starting at the honest root.
	essentialHonestRoot := protocol.SpecEdge(honestEdges["blk-0.a-4.a"])
	blockNum := uint64(10)
	paths, pathLocalTimers, err := tree.findEssentialPaths(
		ctx,
		essentialHonestRoot,
		blockNum,
	)
	require.NoError(t, err)

	// There should be three total essential paths from honest root down
	// to terminal nodes in this test, as there are three terminal nodes,
	// namely, path A ending in blk-3.a-4.a, path B ending in big-0.a-4.a, and path C ending in blk-0.a-2.a.
	//
	// Path A, at block number 10, should have a total weight of 7 as
	// - blk-0.a-4.a has 1 block unrivaled
	// - blk-2.a-4.a has 1 block unrivaled
	// - blk-3.a-4.a has 5 blocks unrivaled
	//
	// Path B, at block number 10, should have a total weight of 6 as
	// - blk-0.a-4.a has 1 block unrivaled
	// - blk-2.a-4.a has 1 block unrivaled
	// - blk-2.a-3.a has 1 block unrivaled
	// - big-0.a-4.a has 3 block unrivaled
	//
	// Path C, at block number 10, should have a total weight of 8 as
	// - blk-0.a-4.a has 1 block unrivaled
	// - blk-0.a-2.a has 7 blocks unrivaled
	require.Equal(t, 3, len(paths))
	require.Equal(t, 3, len(pathLocalTimers))

	wantPathA := essentialPath{
		honestEdges["blk-3.a-4.a"].Id(),
		honestEdges["blk-2.a-4.a"].Id(),
		honestEdges["blk-0.a-4.a"].Id(),
	}
	wantATimers := essentialLocalTimers{5, 1, 1}
	require.Equal(t, wantPathA, paths[0])
	require.Equal(t, wantATimers, pathLocalTimers[0])

	wantPathB := essentialPath{
		honestEdges["big-0.a-4.a"].Id(),
		honestEdges["blk-2.a-3.a"].Id(),
		honestEdges["blk-2.a-4.a"].Id(),
		honestEdges["blk-0.a-4.a"].Id(),
	}
	wantBTimers := essentialLocalTimers{3, 1, 1, 1}
	require.Equal(t, wantPathB, paths[1])
	require.Equal(t, wantBTimers, pathLocalTimers[1])

	wantPathC := essentialPath{
		honestEdges["blk-0.a-2.a"].Id(),
		honestEdges["blk-0.a-4.a"].Id(),
	}
	wantCTimers := essentialLocalTimers{7, 1}
	require.Equal(t, wantPathC, paths[2])
	require.Equal(t, wantCTimers, pathLocalTimers[2])
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

func Test_pathWeightMinHeap(t *testing.T) {
	h := newPathWeightMinHeap()
	require.Equal(t, 0, h.Len())
	h.Push(uint64(3))
	h.Push(uint64(1))
	h.Push(uint64(2))
	require.Equal(t, uint64(1), h.Peek().Unwrap())
	require.Equal(t, uint64(1), h.Pop())
	require.Equal(t, uint64(2), h.Pop())
	require.Equal(t, uint64(3), h.Pop())
	require.Equal(t, 0, h.Len())
	require.True(t, h.Peek().IsNone())
}

func Test_stack(t *testing.T) {
	s := newStack[int]()
	require.Equal(t, 0, s.len())
	s.push(10)
	require.Equal(t, 1, s.len())

	result := s.pop()
	require.False(t, result.IsNone())
	require.Equal(t, 10, result.Unwrap())

	result = s.pop()
	require.True(t, result.IsNone())

	s.push(10)
	s.push(20)
	s.push(30)
	require.Equal(t, 3, s.len())
	s.pop()
	require.Equal(t, 2, s.len())
	s.pop()
	require.Equal(t, 1, s.len())
	s.pop()
	require.Equal(t, 0, s.len())
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
