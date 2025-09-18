package challengetree

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/challenge-manager/challenge-tree/mock"
	"github.com/offchainlabs/nitro/bold/containers/threadsafe"
)

func Test_findEssentialPaths_edgeCases(t *testing.T) {
	ctx := context.Background()
	tree, honestEdges := setupEdgeCaseTest(t)

	// Calculate the essential paths starting at the honest root.
	essentialHonestRoot := protocol.SpecEdge(honestEdges["blk-0.a-32.a"])
	blockNum := uint64(10)
	paths, pathLocalTimers, err := tree.findEssentialPaths(
		ctx,
		essentialHonestRoot,
		blockNum,
	)
	require.NoError(t, err)
	t.Log(paths)
	t.Log(pathLocalTimers)
}

func setupEdgeCaseTest(t *testing.T) (*RoyalChallengeTree, map[mock.EdgeId]*mock.Edge) {
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
	evilEdges := buildEdges(
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-32.b", claimId: evilAssertion, createdAt: 1}),
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-16.b", createdAt: 3}),
		newEdge(&newCfg{t: t, edgeId: "blk-16.b-32.b", createdAt: 3}),
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-8.b", createdAt: 5}),
		newEdge(&newCfg{t: t, edgeId: "blk-8.b-16.b", createdAt: 5}),
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-4.a", createdAt: 7}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-8.b", createdAt: 7}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-6.a", createdAt: 9}),
		newEdge(&newCfg{t: t, edgeId: "blk-6.a-8.b", createdAt: 9}),
		newEdge(&newCfg{t: t, edgeId: "blk-6.a-7.b", createdAt: 11}),
		newEdge(&newCfg{t: t, edgeId: "blk-7.b-8.b", createdAt: 11}),
	)
	honestEdges := buildEdges(
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-32.a", claimId: honestAssertion, createdAt: 2}),
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-16.a", createdAt: 4}),
		newEdge(&newCfg{t: t, edgeId: "blk-16.a-32.a", createdAt: 4}),
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-8.a", createdAt: 6}),
		newEdge(&newCfg{t: t, edgeId: "blk-8.a-16.a", createdAt: 6}),
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-4.a", createdAt: 7}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-8.a", createdAt: 8}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-6.a", createdAt: 9}),
		newEdge(&newCfg{t: t, edgeId: "blk-6.a-8.a", createdAt: 10}),
		newEdge(&newCfg{t: t, edgeId: "blk-6.a-7.a", createdAt: 12}),
		newEdge(&newCfg{t: t, edgeId: "blk-7.a-8.a", createdAt: 12}),
	)

	// Child-relationship linking.
	// Honest.
	honestEdges["blk-0.a-32.a"].LowerChildID = "blk-0.a-16.a"
	honestEdges["blk-0.a-32.a"].UpperChildID = "blk-16.a-32.a"

	honestEdges["blk-0.a-16.a"].LowerChildID = "blk-0.a-8.a"
	honestEdges["blk-0.a-16.a"].UpperChildID = "blk-8.a-16.a"

	honestEdges["blk-0.a-8.a"].LowerChildID = "blk-0.a-4.a"
	honestEdges["blk-0.a-8.a"].UpperChildID = "blk-4.a-8.a"

	honestEdges["blk-4.a-8.a"].LowerChildID = "blk-4.a-6.a"
	honestEdges["blk-4.a-8.a"].UpperChildID = "blk-6.a-8.a"

	honestEdges["blk-6.a-8.a"].LowerChildID = "blk-6.a-7.a"
	honestEdges["blk-6.a-8.a"].UpperChildID = "blk-7.a-8.a"

	// Evil.
	evilEdges["blk-0.a-32.b"].LowerChildID = "blk-0.a-16.b"
	evilEdges["blk-0.a-32.b"].UpperChildID = "blk-16.b-32.b"

	evilEdges["blk-0.a-16.b"].LowerChildID = "blk-0.a-8.b"
	evilEdges["blk-0.a-16.b"].UpperChildID = "blk-8.b-16.b"

	evilEdges["blk-0.a-8.b"].LowerChildID = "blk-0.a-4.a"
	evilEdges["blk-0.a-8.b"].UpperChildID = "blk-4.a-8.b"

	evilEdges["blk-4.a-8.b"].LowerChildID = "blk-4.a-6.a"
	evilEdges["blk-4.a-8.b"].UpperChildID = "blk-6.a-8.b"

	evilEdges["blk-6.a-8.b"].LowerChildID = "blk-6.a-7.b"
	evilEdges["blk-6.a-8.b"].UpperChildID = "blk-7.b-8.b"

	transformedEdges := make(map[protocol.EdgeId]protocol.SpecEdge)
	for _, v := range honestEdges {
		transformedEdges[v.Id()] = v
	}
	allEdges := threadsafe.NewMapFromItems(transformedEdges)
	tree.edges = allEdges

	// Set up rivaled edges.
	mutual := honestEdges["blk-0.a-32.a"].MutualId()
	key := buildEdgeCreationTimeKey(protocol.OriginId{}, mutual)
	tree.edgeCreationTimes.Put(key, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals := tree.edgeCreationTimes.Get(key)
	a := honestEdges["blk-0.a-32.a"]
	b := evilEdges["blk-0.a-32.b"]
	aCreation, err := a.CreatedAtBlock()
	require.NoError(t, err)
	bCreation, err := b.CreatedAtBlock()
	require.NoError(t, err)
	mutuals.Put(a.Id(), creationTime(aCreation))
	mutuals.Put(b.Id(), creationTime(bCreation))

	mutual = honestEdges["blk-0.a-16.a"].MutualId()
	key = buildEdgeCreationTimeKey(protocol.OriginId{}, mutual)
	tree.edgeCreationTimes.Put(key, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.edgeCreationTimes.Get(key)
	a = honestEdges["blk-0.a-16.a"]
	b = evilEdges["blk-0.a-16.b"]
	aCreation, err = a.CreatedAtBlock()
	require.NoError(t, err)
	bCreation, err = b.CreatedAtBlock()
	require.NoError(t, err)
	mutuals.Put(a.Id(), creationTime(aCreation))
	mutuals.Put(b.Id(), creationTime(bCreation))

	mutual = honestEdges["blk-0.a-8.a"].MutualId()
	key = buildEdgeCreationTimeKey(protocol.OriginId{}, mutual)
	tree.edgeCreationTimes.Put(key, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.edgeCreationTimes.Get(key)
	a = honestEdges["blk-0.a-8.a"]
	b = evilEdges["blk-0.a-8.b"]
	aCreation, err = a.CreatedAtBlock()
	require.NoError(t, err)
	bCreation, err = b.CreatedAtBlock()
	require.NoError(t, err)
	mutuals.Put(a.Id(), creationTime(aCreation))
	mutuals.Put(b.Id(), creationTime(bCreation))

	mutual = honestEdges["blk-4.a-8.a"].MutualId()
	key = buildEdgeCreationTimeKey(protocol.OriginId{}, mutual)
	tree.edgeCreationTimes.Put(key, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.edgeCreationTimes.Get(key)
	a = honestEdges["blk-4.a-8.a"]
	b = evilEdges["blk-4.a-8.b"]
	aCreation, err = a.CreatedAtBlock()
	require.NoError(t, err)
	bCreation, err = b.CreatedAtBlock()
	require.NoError(t, err)
	mutuals.Put(a.Id(), creationTime(aCreation))
	mutuals.Put(b.Id(), creationTime(bCreation))

	mutual = honestEdges["blk-6.a-8.a"].MutualId()
	key = buildEdgeCreationTimeKey(protocol.OriginId{}, mutual)
	tree.edgeCreationTimes.Put(key, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.edgeCreationTimes.Get(key)
	a = honestEdges["blk-6.a-8.a"]
	b = evilEdges["blk-6.a-8.b"]
	aCreation, err = a.CreatedAtBlock()
	require.NoError(t, err)
	bCreation, err = b.CreatedAtBlock()
	require.NoError(t, err)
	mutuals.Put(a.Id(), creationTime(aCreation))
	mutuals.Put(b.Id(), creationTime(bCreation))

	mutual = honestEdges["blk-6.a-7.a"].MutualId()
	key = buildEdgeCreationTimeKey(protocol.OriginId{}, mutual)
	tree.edgeCreationTimes.Put(key, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.edgeCreationTimes.Get(key)
	a = honestEdges["blk-6.a-7.a"]
	b = evilEdges["blk-6.a-7.b"]
	aCreation, err = a.CreatedAtBlock()
	require.NoError(t, err)
	bCreation, err = b.CreatedAtBlock()
	require.NoError(t, err)
	mutuals.Put(a.Id(), creationTime(aCreation))
	mutuals.Put(b.Id(), creationTime(bCreation))
	return tree, honestEdges
}
