package challengetree

import (
	"context"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util/option"
	"github.com/OffchainLabs/challenge-protocol-v2/util/threadsafe"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// Tests the following tree, all the way down to the small
// step subchallenge level.
//
//		Block challenge:
//
//			      /--5---6-----8-----------16 = Alice
//			0-----4
//			      \--5'--6'----8'----------16' = Bob
//
//		Big step challenge:
//
//			      /--5---6-----8-----------16 = Alice (claim_id = id(5, 6) in the level above)
//			0-----4
//			      \--5'--6'----8'----------16' = Bob
//
//	 Small step challenge:
//
//			      /--5---6-----8-----------16 = Alice (claim_id = id(5, 6) in the level above)
//			0-----4
//			      \--5'--6'----8'----------16' = Bob
//
// From here, the list of ancestors can be determined all the way to the top.
func TestAncestors_AllChallengeLevels(t *testing.T) {
	ctx := context.Background()
	tree := &HonestChallengeTree{
		edges:                         threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		mutualIds:                     threadsafe.NewMap[protocol.MutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
		honestBigStepLevelZeroEdges:   threadsafe.NewSlice[protocol.ReadOnlyEdge](),
		honestSmallStepLevelZeroEdges: threadsafe.NewSlice[protocol.ReadOnlyEdge](),
		metadataReader:                &mockMetadataReader{},
	}
	// Edge ids that belong to block challenges are prefixed with "blk".
	// For big step, prefixed with "big", and small step, prefixed with "smol".
	setupBlockChallengeTreeSnapshot(t, tree)
	tree.honestBlockChalLevelZeroEdge = option.Some(protocol.ReadOnlyEdge(tree.edges.Get(id("blk-0.a-16.a"))))
	claimId := "blk-4.a-5.a"
	setupBigStepChallengeSnapshot(t, tree, claimId)
	tree.honestBigStepLevelZeroEdges.Push(tree.edges.Get(id("big-0.a-16.a")))
	claimId = "big-4.a-5.a"
	setupSmallStepChallengeSnapshot(t, tree, claimId)
	tree.honestSmallStepLevelZeroEdges.Push(tree.edges.Get(id("smol-0.a-16.a")))
	blockNum := uint64(30)

	t.Run("junk edge fails", func(t *testing.T) {
		// We start by querying for ancestors for a block edge id.
		_, _, err := tree.HonestPathTimer(ctx, id("foo"), blockNum)
		require.ErrorContains(t, err, "not found in honest challenge tree")
	})
	t.Run("dishonest edge lookup fails", func(t *testing.T) {
		_, _, err := tree.HonestPathTimer(ctx, id("blk-0.a-16.b"), blockNum)
		require.ErrorContains(t, err, "not found in honest challenge tree")
	})
	t.Run("block challenge: level zero edge has no ancestors", func(t *testing.T) {
		_, ancestors, err := tree.HonestPathTimer(ctx, id("blk-0.a-16.a"), blockNum)
		require.NoError(t, err)
		require.Equal(t, 0, len(ancestors))
	})
	t.Run("block challenge: single ancestor", func(t *testing.T) {
		_, ancestors, err := tree.HonestPathTimer(ctx, id("blk-0.a-8.a"), blockNum)
		require.NoError(t, err)
		require.Equal(t, HonestAncestors{id("blk-0.a-16.a")}, ancestors)
		_, ancestors, err = tree.HonestPathTimer(ctx, id("blk-8.a-16.a"), blockNum)
		require.NoError(t, err)
		require.Equal(t, HonestAncestors{id("blk-0.a-16.a")}, ancestors)
	})
	t.Run("block challenge: many ancestors", func(t *testing.T) {
		_, ancestors, err := tree.HonestPathTimer(ctx, id("blk-4.a-5.a"), blockNum)
		require.NoError(t, err)
		wanted := HonestAncestors{
			id("blk-4.a-6.a"),
			id("blk-4.a-8.a"),
			id("blk-0.a-8.a"),
			id("blk-0.a-16.a"),
		}
		require.Equal(t, wanted, ancestors)
	})
	t.Run("big step challenge: level zero edge has ancestors from block challenge", func(t *testing.T) {
		_, ancestors, err := tree.HonestPathTimer(ctx, id("big-0.a-16.a"), blockNum)
		require.NoError(t, err)
		wanted := HonestAncestors{
			id("blk-4.a-5.a"),
			id("blk-4.a-6.a"),
			id("blk-4.a-8.a"),
			id("blk-0.a-8.a"),
			id("blk-0.a-16.a"),
		}
		require.Equal(t, wanted, ancestors)
	})
	t.Run("big step challenge: many ancestors plus block challenge ancestors", func(t *testing.T) {
		_, ancestors, err := tree.HonestPathTimer(ctx, id("big-5.a-6.a"), blockNum)
		require.NoError(t, err)
		wanted := HonestAncestors{
			// Big step chal.
			id("big-4.a-6.a"),
			id("big-4.a-8.a"),
			id("big-0.a-8.a"),
			id("big-0.a-16.a"),
			// Block chal.
			id("blk-4.a-5.a"),
			id("blk-4.a-6.a"),
			id("blk-4.a-8.a"),
			id("blk-0.a-8.a"),
			id("blk-0.a-16.a"),
		}
		require.Equal(t, wanted, ancestors)
	})
	t.Run("small step challenge: level zero edge has ancestors from big and block challenge", func(t *testing.T) {
		_, ancestors, err := tree.HonestPathTimer(ctx, id("smol-0.a-16.a"), blockNum)
		require.NoError(t, err)
		wanted := HonestAncestors{
			// Big step chal.
			id("big-4.a-5.a"),
			id("big-4.a-6.a"),
			id("big-4.a-8.a"),
			id("big-0.a-8.a"),
			id("big-0.a-16.a"),
			// Block chal.
			id("blk-4.a-5.a"),
			id("blk-4.a-6.a"),
			id("blk-4.a-8.a"),
			id("blk-0.a-8.a"),
			id("blk-0.a-16.a"),
		}
		require.Equal(t, wanted, ancestors)
	})
	t.Run("small step challenge: lowest level edge has full ancestry", func(t *testing.T) {
		_, ancestors, err := tree.HonestPathTimer(ctx, id("smol-5.a-6.a"), blockNum)
		require.NoError(t, err)
		wanted := HonestAncestors{
			// Small step chal.
			id("smol-4.a-6.a"),
			id("smol-4.a-8.a"),
			id("smol-0.a-8.a"),
			id("smol-0.a-16.a"),
			// Big step chal.
			id("big-4.a-5.a"),
			id("big-4.a-6.a"),
			id("big-4.a-8.a"),
			id("big-0.a-8.a"),
			id("big-0.a-16.a"),
			// Block chal.
			id("blk-4.a-5.a"),
			id("blk-4.a-6.a"),
			id("blk-4.a-8.a"),
			id("blk-0.a-8.a"),
			id("blk-0.a-16.a"),
		}
		require.Equal(t, wanted, ancestors)
	})
}

func Test_findOriginEdge(t *testing.T) {
	edges := threadsafe.NewSlice[protocol.ReadOnlyEdge]()
	origin := protocol.OriginId(common.BytesToHash([]byte("foo")))
	_, ok := findOriginEdge(origin, edges)
	require.Equal(t, false, ok)
	edges.Push(newEdge(&newCfg{
		t:         t,
		originId:  "bar",
		edgeId:    "blk-0.a-4.a",
		claimId:   "",
		createdAt: 2,
	}))

	_, ok = findOriginEdge(origin, edges)
	require.Equal(t, false, ok)

	origin = protocol.OriginId(common.BytesToHash([]byte("bar")))
	got, ok := findOriginEdge(origin, edges)
	require.Equal(t, true, ok)
	require.Equal(t, got.Id(), protocol.EdgeId(common.BytesToHash([]byte("blk-0.a-4.a"))))

	edges.Push(newEdge(&newCfg{
		t:         t,
		originId:  "baz",
		edgeId:    "blk-0.b-4.b",
		claimId:   "",
		createdAt: 2,
	}))

	origin = protocol.OriginId(common.BytesToHash([]byte("baz")))
	got, ok = findOriginEdge(origin, edges)
	require.Equal(t, true, ok)
	require.Equal(t, got.Id(), protocol.EdgeId(common.BytesToHash([]byte("blk-0.b-4.b"))))
}

func buildEdges(allEdges ...*edge) map[edgeId]*edge {
	m := make(map[edgeId]*edge)
	for _, e := range allEdges {
		m[e.id] = e
	}
	return m
}

// Sets up the following block challenge snapshot:
//
//	      /--5---6-----8-----------16 = Alice
//	0-----4
//	      \--5'--6'----8'----------16' = Bob
//
// and then inserts the respective edges into a challenge tree.
func setupBlockChallengeTreeSnapshot(t *testing.T, tree *HonestChallengeTree) {
	t.Helper()
	aliceEdges := buildEdges(
		// Alice.
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-16.a", createdAt: 1}),
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-8.a", createdAt: 3}),
		newEdge(&newCfg{t: t, edgeId: "blk-8.a-16.a", createdAt: 3}),
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-4.a", createdAt: 5}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-8.a", createdAt: 5}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-6.a", createdAt: 7}),
		newEdge(&newCfg{t: t, edgeId: "blk-6.a-8.a", createdAt: 7}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-5.a", createdAt: 9}),
		newEdge(&newCfg{t: t, edgeId: "blk-5.a-6.a", createdAt: 9}),
	)
	bobEdges := buildEdges(
		// Bob.
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-16.b", createdAt: 2}),
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-8.b", createdAt: 4}),
		newEdge(&newCfg{t: t, edgeId: "blk-8.b-16.b", createdAt: 4}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-8.b", createdAt: 6}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-6.b", createdAt: 6}),
		newEdge(&newCfg{t: t, edgeId: "blk-6.b-8.b", createdAt: 8}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-5.b", createdAt: 10}),
		newEdge(&newCfg{t: t, edgeId: "blk-5.b-6.b", createdAt: 10}),
	)
	// Child-relationship linking.
	// Alice.
	aliceEdges["blk-0.a-16.a"].lowerChildId = "blk-0.a-8.a"
	aliceEdges["blk-0.a-16.a"].upperChildId = "blk-8.a-16.a"
	aliceEdges["blk-0.a-8.a"].lowerChildId = "blk-0.a-4.a"
	aliceEdges["blk-0.a-8.a"].upperChildId = "blk-4.a-8.a"
	aliceEdges["blk-4.a-8.a"].lowerChildId = "blk-4.a-6.a"
	aliceEdges["blk-4.a-8.a"].upperChildId = "blk-6.a-8.a"
	aliceEdges["blk-4.a-6.a"].lowerChildId = "blk-4.a-5.a"
	aliceEdges["blk-4.a-6.a"].upperChildId = "blk-5.a-6.a"
	// Bob.
	bobEdges["blk-0.a-16.b"].lowerChildId = "blk-0.a-8.b"
	bobEdges["blk-0.a-16.b"].upperChildId = "blk-8.b-16.b"
	bobEdges["blk-0.a-8.b"].lowerChildId = "blk-0.a-4.a"
	bobEdges["blk-0.a-8.b"].upperChildId = "blk-4.a-8.b"
	bobEdges["blk-4.a-8.b"].lowerChildId = "blk-4.a-6.b"
	bobEdges["blk-4.a-8.b"].upperChildId = "blk-6.b-6.8"
	bobEdges["blk-4.a-6.b"].lowerChildId = "blk-4.a-5.b"
	bobEdges["blk-4.a-6.b"].upperChildId = "blk-5.b-6.b"

	transformedEdges := make(map[protocol.EdgeId]protocol.SpecEdge)
	for _, v := range aliceEdges {
		transformedEdges[v.Id()] = v
	}
	allEdges := threadsafe.NewMapFromItems(transformedEdges)
	tree.edges = allEdges

	// Set up rivaled edges.
	mutual := aliceEdges["blk-0.a-16.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals := tree.mutualIds.Get(mutual)
	a := aliceEdges["blk-0.a-16.a"]
	b := bobEdges["blk-0.a-16.b"]
	mutuals.Put(a.Id(), creationTime(a.CreatedAtBlock()))
	mutuals.Put(b.Id(), creationTime(b.CreatedAtBlock()))

	mutual = aliceEdges["blk-0.a-8.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.mutualIds.Get(mutual)
	a = aliceEdges["blk-0.a-8.a"]
	b = bobEdges["blk-0.a-8.b"]
	mutuals.Put(a.Id(), creationTime(a.CreatedAtBlock()))
	mutuals.Put(b.Id(), creationTime(b.CreatedAtBlock()))

	mutual = aliceEdges["blk-4.a-8.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.mutualIds.Get(mutual)
	a = aliceEdges["blk-4.a-8.a"]
	b = bobEdges["blk-4.a-8.b"]
	mutuals.Put(a.Id(), creationTime(a.CreatedAtBlock()))
	mutuals.Put(b.Id(), creationTime(b.CreatedAtBlock()))

	mutual = aliceEdges["blk-4.a-6.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.mutualIds.Get(mutual)
	a = aliceEdges["blk-4.a-6.a"]
	b = bobEdges["blk-4.a-6.b"]
	mutuals.Put(a.Id(), creationTime(a.CreatedAtBlock()))
	mutuals.Put(b.Id(), creationTime(b.CreatedAtBlock()))

	mutual = aliceEdges["blk-4.a-5.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.mutualIds.Get(mutual)
	a = aliceEdges["blk-4.a-5.a"]
	b = bobEdges["blk-4.a-5.b"]
	mutuals.Put(a.Id(), creationTime(a.CreatedAtBlock()))
	mutuals.Put(b.Id(), creationTime(b.CreatedAtBlock()))
}

func id(eId edgeId) protocol.EdgeId {
	return protocol.EdgeId(common.BytesToHash([]byte(eId)))
}

// Sets up the following big step challenge snapshot:
//
//	      /--5---6-----8-----------16 = Alice
//	0-----4
//	      \--5'--6'----8'----------16' = Bob
//
// and then inserts the respective edges into a challenge tree.
func setupBigStepChallengeSnapshot(t *testing.T, tree *HonestChallengeTree, claimId string) {
	t.Helper()
	originEdge := tree.edges.Get(id(edgeId(claimId))).(*edge)
	originId := originId(originEdge.computeMutualId())
	aliceEdges := buildEdges(
		// Alice.
		newEdge(&newCfg{t: t, edgeId: "big-0.a-16.a", originId: originId, claimId: claimId, createdAt: 11}),
		newEdge(&newCfg{t: t, edgeId: "big-0.a-8.a", originId: originId, createdAt: 13}),
		newEdge(&newCfg{t: t, edgeId: "big-8.a-16.a", originId: originId, createdAt: 13}),
		newEdge(&newCfg{t: t, edgeId: "big-0.a-4.a", originId: originId, createdAt: 15}),
		newEdge(&newCfg{t: t, edgeId: "big-4.a-8.a", originId: originId, createdAt: 15}),
		newEdge(&newCfg{t: t, edgeId: "big-4.a-6.a", originId: originId, createdAt: 17}),
		newEdge(&newCfg{t: t, edgeId: "big-6.a-8.a", originId: originId, createdAt: 17}),
		newEdge(&newCfg{t: t, edgeId: "big-4.a-5.a", originId: originId, createdAt: 19}),
		newEdge(&newCfg{t: t, edgeId: "big-5.a-6.a", originId: originId, createdAt: 19}),
	)
	bobEdges := buildEdges(
		// Bob.
		newEdge(&newCfg{t: t, edgeId: "big-0.a-16.b", originId: originId, createdAt: 12}),
		newEdge(&newCfg{t: t, edgeId: "big-0.a-8.b", originId: originId, createdAt: 14}),
		newEdge(&newCfg{t: t, edgeId: "big-8.b-16.b", originId: originId, createdAt: 14}),
		newEdge(&newCfg{t: t, edgeId: "big-4.a-8.b", originId: originId, createdAt: 16}),
		newEdge(&newCfg{t: t, edgeId: "big-4.a-6.b", originId: originId, createdAt: 18}),
		newEdge(&newCfg{t: t, edgeId: "big-6.b-8.b", originId: originId, createdAt: 18}),
		newEdge(&newCfg{t: t, edgeId: "big-4.a-5.b", originId: originId, createdAt: 20}),
		newEdge(&newCfg{t: t, edgeId: "big-5.b-6.b", originId: originId, createdAt: 20}),
	)
	// Child-relationship linking.
	// Alice.
	aliceEdges["big-0.a-16.a"].lowerChildId = "big-0.a-8.a"
	aliceEdges["big-0.a-16.a"].upperChildId = "big-8.a-16.a"
	aliceEdges["big-0.a-8.a"].lowerChildId = "big-0.a-4.a"
	aliceEdges["big-0.a-8.a"].upperChildId = "big-4.a-8.a"
	aliceEdges["big-4.a-8.a"].lowerChildId = "big-4.a-6.a"
	aliceEdges["big-4.a-8.a"].upperChildId = "big-6.a-8.a"
	aliceEdges["big-4.a-6.a"].lowerChildId = "big-4.a-5.a"
	aliceEdges["big-4.a-6.a"].upperChildId = "big-5.a-6.a"
	// Bob.
	bobEdges["big-0.a-16.b"].lowerChildId = "big-0.a-8.b"
	bobEdges["big-0.a-16.b"].upperChildId = "big-8.b-16.b"
	bobEdges["big-0.a-8.b"].lowerChildId = "big-0.a-4.a"
	bobEdges["big-0.a-8.b"].upperChildId = "big-4.a-8.b"
	bobEdges["big-4.a-8.b"].lowerChildId = "big-4.a-6.b"
	bobEdges["big-4.a-8.b"].upperChildId = "big-6.b-6.8"
	bobEdges["big-4.a-6.b"].lowerChildId = "big-4.a-5.b"
	bobEdges["big-4.a-6.b"].upperChildId = "big-5.b-6.b"

	for _, v := range aliceEdges {
		tree.edges.Put(v.Id(), v)
	}

	// Set up rivaled edges.
	mutual := aliceEdges["big-0.a-16.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals := tree.mutualIds.Get(mutual)
	a := aliceEdges["big-0.a-16.a"]
	b := bobEdges["big-0.a-16.b"]
	mutuals.Put(a.Id(), creationTime(a.CreatedAtBlock()))
	mutuals.Put(b.Id(), creationTime(b.CreatedAtBlock()))

	mutual = aliceEdges["big-0.a-8.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.mutualIds.Get(mutual)
	a = aliceEdges["big-0.a-8.a"]
	b = bobEdges["big-0.a-8.b"]
	mutuals.Put(a.Id(), creationTime(a.CreatedAtBlock()))
	mutuals.Put(b.Id(), creationTime(b.CreatedAtBlock()))

	mutual = aliceEdges["big-4.a-8.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.mutualIds.Get(mutual)
	a = aliceEdges["big-4.a-8.a"]
	b = bobEdges["big-4.a-8.b"]
	mutuals.Put(a.Id(), creationTime(a.CreatedAtBlock()))
	mutuals.Put(b.Id(), creationTime(b.CreatedAtBlock()))

	mutual = aliceEdges["big-4.a-6.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.mutualIds.Get(mutual)
	a = aliceEdges["big-4.a-6.a"]
	b = bobEdges["big-4.a-6.b"]
	mutuals.Put(a.Id(), creationTime(a.CreatedAtBlock()))
	mutuals.Put(b.Id(), creationTime(b.CreatedAtBlock()))

	mutual = aliceEdges["big-4.a-5.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.mutualIds.Get(mutual)
	a = aliceEdges["big-4.a-5.a"]
	b = bobEdges["big-4.a-5.b"]
	mutuals.Put(a.Id(), creationTime(a.CreatedAtBlock()))
	mutuals.Put(b.Id(), creationTime(b.CreatedAtBlock()))
}

// Sets up the following small step challenge snapshot:
//
//	      /--5---6-----8-----------16 = Alice
//	0-----4
//	      \--5'--6'----8'----------16' = Bob
//
// and then inserts the respective edges into a challenge tree.
//
// and then inserts the respective edges into a challenge tree.
func setupSmallStepChallengeSnapshot(t *testing.T, tree *HonestChallengeTree, claimId string) {
	t.Helper()
	originEdge := tree.edges.Get(id(edgeId(claimId))).(*edge)
	originId := originId(originEdge.computeMutualId())
	aliceEdges := buildEdges(
		// Alice.
		newEdge(&newCfg{t: t, edgeId: "smol-0.a-16.a", originId: originId, claimId: claimId, createdAt: 21}),
		newEdge(&newCfg{t: t, edgeId: "smol-0.a-8.a", originId: originId, createdAt: 23}),
		newEdge(&newCfg{t: t, edgeId: "smol-8.a-16.a", originId: originId, createdAt: 23}),
		newEdge(&newCfg{t: t, edgeId: "smol-0.a-4.a", originId: originId, createdAt: 25}),
		newEdge(&newCfg{t: t, edgeId: "smol-4.a-8.a", originId: originId, createdAt: 25}),
		newEdge(&newCfg{t: t, edgeId: "smol-4.a-6.a", originId: originId, createdAt: 27}),
		newEdge(&newCfg{t: t, edgeId: "smol-6.a-8.a", originId: originId, createdAt: 27}),
		newEdge(&newCfg{t: t, edgeId: "smol-4.a-5.a", originId: originId, createdAt: 29}),
		newEdge(&newCfg{t: t, edgeId: "smol-5.a-6.a", originId: originId, createdAt: 29}),
	)
	bobEdges := buildEdges(
		// Bob.
		newEdge(&newCfg{t: t, edgeId: "smol-0.a-16.b", originId: originId, createdAt: 22}),
		newEdge(&newCfg{t: t, edgeId: "smol-0.a-8.b", originId: originId, createdAt: 24}),
		newEdge(&newCfg{t: t, edgeId: "smol-8.b-16.b", originId: originId, createdAt: 24}),
		newEdge(&newCfg{t: t, edgeId: "smol-4.a-8.b", originId: originId, createdAt: 26}),
		newEdge(&newCfg{t: t, edgeId: "smol-4.a-6.b", originId: originId, createdAt: 28}),
		newEdge(&newCfg{t: t, edgeId: "smol-6.b-8.b", originId: originId, createdAt: 28}),
		newEdge(&newCfg{t: t, edgeId: "smol-4.a-5.b", originId: originId, createdAt: 30}),
		newEdge(&newCfg{t: t, edgeId: "smol-5.b-6.b", originId: originId, createdAt: 30}),
	)
	// Child-relationship linking.
	// Alice.
	aliceEdges["smol-0.a-16.a"].lowerChildId = "smol-0.a-8.a"
	aliceEdges["smol-0.a-16.a"].upperChildId = "smol-8.a-16.a"
	aliceEdges["smol-0.a-8.a"].lowerChildId = "smol-0.a-4.a"
	aliceEdges["smol-0.a-8.a"].upperChildId = "smol-4.a-8.a"
	aliceEdges["smol-4.a-8.a"].lowerChildId = "smol-4.a-6.a"
	aliceEdges["smol-4.a-8.a"].upperChildId = "smol-6.a-8.a"
	aliceEdges["smol-4.a-6.a"].lowerChildId = "smol-4.a-5.a"
	aliceEdges["smol-4.a-6.a"].upperChildId = "smol-5.a-6.a"
	// Bob.
	bobEdges["smol-0.a-16.b"].lowerChildId = "smol-0.a-8.b"
	bobEdges["smol-0.a-16.b"].upperChildId = "smol-8.b-16.b"
	bobEdges["smol-0.a-8.b"].lowerChildId = "smol-0.a-4.a"
	bobEdges["smol-0.a-8.b"].upperChildId = "smol-4.a-8.b"
	bobEdges["smol-4.a-8.b"].lowerChildId = "smol-4.a-6.b"
	bobEdges["smol-4.a-8.b"].upperChildId = "smol-6.b-6.8"
	bobEdges["smol-4.a-6.b"].lowerChildId = "smol-4.a-5.b"
	bobEdges["smol-4.a-6.b"].upperChildId = "smol-5.b-6.b"

	for _, v := range aliceEdges {
		tree.edges.Put(v.Id(), v)
	}

	// Set up rivaled edges.
	mutual := aliceEdges["smol-0.a-16.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals := tree.mutualIds.Get(mutual)
	a := aliceEdges["smol-0.a-16.a"]
	b := bobEdges["smol-0.a-16.b"]
	mutuals.Put(a.Id(), creationTime(a.CreatedAtBlock()))
	mutuals.Put(b.Id(), creationTime(b.CreatedAtBlock()))

	mutual = aliceEdges["smol-0.a-8.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.mutualIds.Get(mutual)
	a = aliceEdges["smol-0.a-8.a"]
	b = bobEdges["smol-0.a-8.b"]
	mutuals.Put(a.Id(), creationTime(a.CreatedAtBlock()))
	mutuals.Put(b.Id(), creationTime(b.CreatedAtBlock()))

	mutual = aliceEdges["smol-4.a-8.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.mutualIds.Get(mutual)
	a = aliceEdges["smol-4.a-8.a"]
	b = bobEdges["smol-4.a-8.b"]
	mutuals.Put(a.Id(), creationTime(a.CreatedAtBlock()))
	mutuals.Put(b.Id(), creationTime(b.CreatedAtBlock()))

	mutual = aliceEdges["smol-4.a-6.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.mutualIds.Get(mutual)
	a = aliceEdges["smol-4.a-6.a"]
	b = bobEdges["smol-4.a-6.b"]
	mutuals.Put(a.Id(), creationTime(a.CreatedAtBlock()))
	mutuals.Put(b.Id(), creationTime(b.CreatedAtBlock()))

	mutual = aliceEdges["smol-4.a-5.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = tree.mutualIds.Get(mutual)
	a = aliceEdges["smol-4.a-5.a"]
	b = bobEdges["smol-4.a-5.b"]
	mutuals.Put(a.Id(), creationTime(a.CreatedAtBlock()))
	mutuals.Put(b.Id(), creationTime(b.CreatedAtBlock()))
}
