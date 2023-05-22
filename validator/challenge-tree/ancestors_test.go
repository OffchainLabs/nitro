package challengetree

import (
	"context"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
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
	tree := &HonestChallengeTree{
		edges:                         threadsafe.NewMap[protocol.EdgeId, protocol.ReadOnlyEdge](),
		mutualIds:                     threadsafe.NewMap[protocol.MutualId, *threadsafe.Set[protocol.EdgeId]](),
		honestBigStepLevelZeroEdges:   threadsafe.NewSlice[protocol.ReadOnlyEdge](),
		honestSmallStepLevelZeroEdges: threadsafe.NewSlice[protocol.ReadOnlyEdge](),
	}
	// Edge ids that belong to block challenges are prefixed with "blk".
	// For big step, prefixed with "big", and small step, prefixed with "smol".
	setupBlockChallengeTreeSnapshot(t, tree)
	tree.honestBlockChalLevelZeroEdge = util.Some(tree.edges.Get(id("blk-0.a-16.a")))
	claimId := "blk-4.a-5.a"
	setupBigStepChallengeSnapshot(t, tree, claimId)
	tree.honestBigStepLevelZeroEdges.Push(tree.edges.Get(id("big-0.a-16.a")))
	claimId = "big-4.a-5.a"
	setupSmallStepChallengeSnapshot(t, tree, claimId)
	tree.honestSmallStepLevelZeroEdges.Push(tree.edges.Get(id("smol-0.a-16.a")))
	ctx := context.Background()

	t.Run("junk edge fails", func(t *testing.T) {
		// We start by querying for ancestors for a block edge id.
		_, err := tree.AncestorsForHonestEdge(ctx, id("foo"))
		require.ErrorContains(t, err, "not found in honest challenge tree")
	})
	t.Run("dishonest edge lookup fails", func(t *testing.T) {
		_, err := tree.AncestorsForHonestEdge(ctx, id("blk-0.a-16.b"))
		require.ErrorContains(t, err, "not found in honest challenge tree")
	})
	t.Run("block challenge: level zero edge has no ancestors", func(t *testing.T) {
		ancestors, err := tree.AncestorsForHonestEdge(ctx, id("blk-0.a-16.a"))
		require.NoError(t, err)
		require.Equal(t, 0, len(ancestors))
	})
	t.Run("block challenge: single ancestor", func(t *testing.T) {
		ancestors, err := tree.AncestorsForHonestEdge(ctx, id("blk-0.a-8.a"))
		require.NoError(t, err)
		require.Equal(t, []protocol.EdgeId{id("blk-0.a-16.a")}, ancestors)
		ancestors, err = tree.AncestorsForHonestEdge(ctx, id("blk-8.a-16.a"))
		require.NoError(t, err)
		require.Equal(t, []protocol.EdgeId{id("blk-0.a-16.a")}, ancestors)
	})
	t.Run("block challenge: many ancestors", func(t *testing.T) {
		ancestors, err := tree.AncestorsForHonestEdge(ctx, id("blk-4.a-5.a"))
		require.NoError(t, err)
		wanted := []protocol.EdgeId{
			id("blk-4.a-6.a"),
			id("blk-4.a-8.a"),
			id("blk-0.a-8.a"),
			id("blk-0.a-16.a"),
		}
		require.Equal(t, wanted, ancestors)
	})
	t.Run("big step challenge: level zero edge has ancestors from block challenge", func(t *testing.T) {
		ancestors, err := tree.AncestorsForHonestEdge(ctx, id("big-0.a-16.a"))
		require.NoError(t, err)
		wanted := []protocol.EdgeId{
			id("blk-4.a-5.a"),
			id("blk-4.a-6.a"),
			id("blk-4.a-8.a"),
			id("blk-0.a-8.a"),
			id("blk-0.a-16.a"),
		}
		require.Equal(t, wanted, ancestors)
	})
	t.Run("big step challenge: many ancestors plus block challenge ancestors", func(t *testing.T) {
		ancestors, err := tree.AncestorsForHonestEdge(ctx, id("big-5.a-6.a"))
		require.NoError(t, err)
		wanted := []protocol.EdgeId{
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
		ancestors, err := tree.AncestorsForHonestEdge(ctx, id("smol-0.a-16.a"))
		require.NoError(t, err)
		wanted := []protocol.EdgeId{
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
		ancestors, err := tree.AncestorsForHonestEdge(ctx, id("smol-5.a-6.a"))
		require.NoError(t, err)
		wanted := []protocol.EdgeId{
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
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-16.a"}),
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-8.a"}),
		newEdge(&newCfg{t: t, edgeId: "blk-8.a-16.a"}),
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-4.a"}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-8.a"}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-6.a"}),
		newEdge(&newCfg{t: t, edgeId: "blk-6.a-8.a"}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-5.a"}),
		newEdge(&newCfg{t: t, edgeId: "blk-5.a-6.a"}),
	)
	bobEdges := buildEdges(
		// Bob.
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-16.b"}),
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-8.b"}),
		newEdge(&newCfg{t: t, edgeId: "blk-8.b-16.b"}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-8.b"}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-6.b"}),
		newEdge(&newCfg{t: t, edgeId: "blk-6.b-8.b"}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-5.b"}),
		newEdge(&newCfg{t: t, edgeId: "blk-5.b-6.b"}),
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

	transformedEdges := make(map[protocol.EdgeId]protocol.ReadOnlyEdge)
	for _, v := range aliceEdges {
		transformedEdges[v.Id()] = v
	}
	allEdges := threadsafe.NewMapFromItems(transformedEdges)
	tree.edges = allEdges

	// Set up rivaled edges.
	mutual := aliceEdges["blk-0.a-16.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewSet[protocol.EdgeId]())
	mutuals := tree.mutualIds.Get(mutual)
	mutuals.Insert(id("blk-0.a-16.a"))
	mutuals.Insert(id("blk-0.a-16.b"))

	mutual = aliceEdges["blk-0.a-8.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewSet[protocol.EdgeId]())
	mutuals = tree.mutualIds.Get(mutual)
	mutuals.Insert(id("blk-0.a-8.a"))
	mutuals.Insert(id("blk-0.a-8.b"))

	mutual = aliceEdges["blk-4.a-8.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewSet[protocol.EdgeId]())
	mutuals = tree.mutualIds.Get(mutual)
	mutuals.Insert(id("blk-4.a-8.a"))
	mutuals.Insert(id("blk-4.a-8.b"))

	mutual = aliceEdges["blk-4.a-6.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewSet[protocol.EdgeId]())
	mutuals = tree.mutualIds.Get(mutual)
	mutuals.Insert(id("blk-4.a-6.a"))
	mutuals.Insert(id("blk-4.a-6.b"))

	mutual = aliceEdges["blk-4.a-5.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewSet[protocol.EdgeId]())
	mutuals = tree.mutualIds.Get(mutual)
	mutuals.Insert(id("blk-4.a-5.a"))
	mutuals.Insert(id("blk-4.a-5.b"))
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
		newEdge(&newCfg{t: t, edgeId: "big-0.a-16.a", claimId: claimId, originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "big-0.a-8.a", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "big-8.a-16.a", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "big-0.a-4.a", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "big-4.a-8.a", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "big-4.a-6.a", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "big-6.a-8.a", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "big-4.a-5.a", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "big-5.a-6.a", originId: originId}),
	)
	bobEdges := buildEdges(
		// Bob.
		newEdge(&newCfg{t: t, edgeId: "big-0.a-16.b", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "big-0.a-8.b", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "big-8.b-16.b", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "big-4.a-8.b", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "big-4.a-6.b", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "big-6.b-8.b", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "big-4.a-5.b", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "big-5.b-6.b", originId: originId}),
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
	tree.mutualIds.Put(mutual, threadsafe.NewSet[protocol.EdgeId]())
	mutuals := tree.mutualIds.Get(mutual)
	mutuals.Insert(id("big-0.a-16.a"))
	mutuals.Insert(id("big-0.a-16.b"))

	mutual = aliceEdges["big-0.a-8.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewSet[protocol.EdgeId]())
	mutuals = tree.mutualIds.Get(mutual)
	mutuals.Insert(id("big-0.a-8.a"))
	mutuals.Insert(id("big-0.a-8.b"))

	mutual = aliceEdges["big-4.a-8.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewSet[protocol.EdgeId]())
	mutuals = tree.mutualIds.Get(mutual)
	mutuals.Insert(id("big-4.a-8.a"))
	mutuals.Insert(id("big-4.a-8.b"))

	mutual = aliceEdges["big-4.a-6.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewSet[protocol.EdgeId]())
	mutuals = tree.mutualIds.Get(mutual)
	mutuals.Insert(id("big-4.a-6.a"))
	mutuals.Insert(id("big-4.a-6.b"))

	mutual = aliceEdges["big-4.a-5.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewSet[protocol.EdgeId]())
	mutuals = tree.mutualIds.Get(mutual)
	mutuals.Insert(id("big-4.a-5.a"))
	mutuals.Insert(id("big-4.a-5.b"))
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
		newEdge(&newCfg{t: t, edgeId: "smol-0.a-16.a", claimId: claimId, originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "smol-0.a-8.a", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "smol-8.a-16.a", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "smol-0.a-4.a", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "smol-4.a-8.a", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "smol-4.a-6.a", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "smol-6.a-8.a", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "smol-4.a-5.a", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "smol-5.a-6.a", originId: originId}),
	)
	bobEdges := buildEdges(
		// Bob.
		newEdge(&newCfg{t: t, edgeId: "smol-0.a-16.b", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "smol-0.a-8.b", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "smol-8.b-16.b", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "smol-4.a-8.b", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "smol-4.a-6.b", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "smol-6.b-8.b", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "smol-4.a-5.b", originId: originId}),
		newEdge(&newCfg{t: t, edgeId: "smol-5.b-6.b", originId: originId}),
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
	tree.mutualIds.Put(mutual, threadsafe.NewSet[protocol.EdgeId]())
	mutuals := tree.mutualIds.Get(mutual)
	mutuals.Insert(id("smol-0.a-16.a"))
	mutuals.Insert(id("smol-0.a-16.b"))

	mutual = aliceEdges["smol-0.a-8.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewSet[protocol.EdgeId]())
	mutuals = tree.mutualIds.Get(mutual)
	mutuals.Insert(id("smol-0.a-8.a"))
	mutuals.Insert(id("smol-0.a-8.b"))

	mutual = aliceEdges["smol-4.a-8.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewSet[protocol.EdgeId]())
	mutuals = tree.mutualIds.Get(mutual)
	mutuals.Insert(id("smol-4.a-8.a"))
	mutuals.Insert(id("smol-4.a-8.b"))

	mutual = aliceEdges["smol-4.a-6.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewSet[protocol.EdgeId]())
	mutuals = tree.mutualIds.Get(mutual)
	mutuals.Insert(id("smol-4.a-6.a"))
	mutuals.Insert(id("smol-4.a-6.b"))

	mutual = aliceEdges["smol-4.a-5.a"].MutualId()
	tree.mutualIds.Put(mutual, threadsafe.NewSet[protocol.EdgeId]())
	mutuals = tree.mutualIds.Get(mutual)
	mutuals.Insert(id("smol-4.a-5.a"))
	mutuals.Insert(id("smol-4.a-5.b"))
}
