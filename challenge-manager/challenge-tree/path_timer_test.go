package challengetree

import (
	"context"
	"testing"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/containers/option"
	"github.com/OffchainLabs/challenge-protocol-v2/containers/threadsafe"
	"github.com/stretchr/testify/require"
)

// The following tests checks a scenario where the honest
// and dishonest parties take turns making challenge moves,
// and as a result, their edges will be unrivaled for some time,
// contributing to the path timer of edges we will query in this test.
//
// We first setup the following challenge tree, where branch `a` is honest.
//
//	 0-----4a----- 8a-------16a
//		     \------8b-------16b
//
// Here are the creation times of each edge:
//
//	Alice (honest)
//	  0-16a        = T1
//	  0-8a, 8a-16a = T3
//	  0-4a, 4a-8a  = T5
//
//	Bob (evil)
//	  0-16b        = T2
//	  0-8b, 8b-16b = T4
//	  4a-8b        = T6
//
// In this contrived example, Alice and Bob's edges will have
// a time interval of 1 in which they are unrivaled.
func TestPathTimer_FlipFlop(t *testing.T) {
	edges := buildEdges(
		// Alice.
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-16.a", createdAt: 1}),
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-8.a", createdAt: 3}),
		newEdge(&newCfg{t: t, edgeId: "blk-8.a-16.a", createdAt: 3}),
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-4.a", createdAt: 5}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-8.a", createdAt: 5}),
		// Bob.
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-16.b", createdAt: 2}),
		newEdge(&newCfg{t: t, edgeId: "blk-0.a-8.b", createdAt: 4}),
		newEdge(&newCfg{t: t, edgeId: "blk-8.b-16.b", createdAt: 4}),
		newEdge(&newCfg{t: t, edgeId: "blk-4.a-8.b", createdAt: 6}),
	)
	// Child-relationship linking.
	// Alice.
	edges["blk-0.a-16.a"].lowerChildId = "blk-0.a-8.a"
	edges["blk-0.a-16.a"].upperChildId = "blk-8.a-16.a"
	edges["blk-0.a-8.a"].lowerChildId = "blk-0.a-4.a"
	edges["blk-0.a-8.a"].upperChildId = "blk-4.a-8.a"
	// Bob.
	edges["blk-0.a-16.b"].lowerChildId = "blk-0.a-8.b"
	edges["blk-0.a-16.b"].upperChildId = "blk-8.b-16.b"
	edges["blk-0.a-8.b"].lowerChildId = "blk-0.a-4.a"
	edges["blk-0.a-8.b"].upperChildId = "blk-4.a-8.b"

	transformedEdges := make(map[protocol.EdgeId]protocol.SpecEdge)
	timers := make(map[protocol.EdgeId]uint64)
	for _, v := range edges {
		transformedEdges[v.Id()] = v
		timers[v.Id()] = 0
	}
	allEdges := threadsafe.NewMapFromItems(transformedEdges)
	ht := &HonestChallengeTree{
		edges:                         allEdges,
		mutualIds:                     threadsafe.NewMap[protocol.MutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
		honestBigStepLevelZeroEdges:   threadsafe.NewSlice[protocol.ReadOnlyEdge](),
		honestSmallStepLevelZeroEdges: threadsafe.NewSlice[protocol.ReadOnlyEdge](),
		metadataReader:                &mockMetadataReader{},
	}
	// Three pairs of edges are rivaled in this test: 0-16, 0-8, and 4-8.
	mutual := edges["blk-0.a-16.a"].MutualId()

	ht.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals := ht.mutualIds.Get(mutual)
	idd := id("blk-0.a-16.a")
	mutuals.Put(idd, creationTime(ht.edges.Get(idd).CreatedAtBlock()))
	idd = id("blk-0.a-16.b")
	mutuals.Put(idd, creationTime(ht.edges.Get(idd).CreatedAtBlock()))

	mutual = edges["blk-0.a-8.a"].MutualId()

	ht.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = ht.mutualIds.Get(mutual)
	idd = id("blk-0.a-8.a")
	mutuals.Put(idd, creationTime(ht.edges.Get(idd).CreatedAtBlock()))
	idd = id("blk-0.a-8.b")
	mutuals.Put(idd, creationTime(ht.edges.Get(idd).CreatedAtBlock()))

	mutual = edges["blk-4.a-8.a"].MutualId()

	ht.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
	mutuals = ht.mutualIds.Get(mutual)
	idd = id("blk-4.a-8.a")
	mutuals.Put(idd, creationTime(ht.edges.Get(idd).CreatedAtBlock()))
	idd = id("blk-4.a-8.b")
	mutuals.Put(idd, creationTime(ht.edges.Get(idd).CreatedAtBlock()))

	ht.honestBlockChalLevelZeroEdge = option.Some(protocol.ReadOnlyEdge(ht.edges.Get(id("blk-0.a-16.a"))))
	ctx := context.Background()

	t.Run("querying path timer before creation should return zero", func(t *testing.T) {
		edge := ht.edges.Get(id("blk-0.a-16.a"))
		timer, _, err := ht.HonestPathTimer(ctx, edge.Id(), 0)
		require.NoError(t, err)
		require.Equal(t, PathTimer(0), timer)
	})
	t.Run("at creation time should be zero if no parents", func(t *testing.T) {
		edge := ht.edges.Get(id("blk-0.a-16.a"))
		timer, _, err := ht.HonestPathTimer(ctx, edge.Id(), edge.CreatedAtBlock())
		require.NoError(t, err)
		require.Equal(t, PathTimer(0), timer)
	})
	t.Run("OK", func(t *testing.T) {
		// Top-level edge should have spent 1 second unrivaled
		// as its rival was created 1 second after its creation.
		edge := ht.edges.Get(id("blk-0.a-16.a"))
		timer, _, err := ht.HonestPathTimer(ctx, edge.Id(), edge.CreatedAtBlock()+1)
		require.NoError(t, err)
		require.Equal(t, PathTimer(1), timer)

		// Now we look at the lower honest child, 0.a-8.a. It will have spent
		// 1 second unrivaled and will inherit the local timers of its honest ancestors.
		// which is 1 for a total of 2.
		edge = ht.edges.Get(id("blk-0.a-8.a"))
		timer, _, err = ht.HonestPathTimer(ctx, edge.Id(), edge.CreatedAtBlock()+1)
		require.NoError(t, err)
		require.Equal(t, PathTimer(2), timer)

		// Now we look at the upper honest grandchild, 4.a-8.a. It will
		// have spent 1 second unrivaled.
		edge = ht.edges.Get(id("blk-4.a-8.a"))
		timer, _, err = ht.HonestPathTimer(ctx, edge.Id(), edge.CreatedAtBlock()+1)
		require.NoError(t, err)
		require.Equal(t, PathTimer(3), timer)

		// The lower-most child, which is unrivaled, and is 0.a-4.a,
		// will inherit the path timers of its ancestors AND also increase
		// its local timer each time we query it as it has no rival
		// to contend it.
		edge = ht.edges.Get(id("blk-0.a-4.a"))

		// Querying it at creation time+1 should just have the path timers
		// of its ancestors that count, which is a total of 3.
		timer, _, err = ht.HonestPathTimer(ctx, edge.Id(), edge.CreatedAtBlock()+1)
		require.NoError(t, err)
		require.Equal(t, PathTimer(3), timer)

		// Continuing to query it at time T+i should increase the timer
		// as it is unrivaled.
		for i := uint64(2); i < 10; i++ {
			timer, _, err = ht.HonestPathTimer(ctx, edge.Id(), edge.CreatedAtBlock()+i)
			require.NoError(t, err)
			require.Equal(t, PathTimer(2)+PathTimer(i), timer)
		}
	})
	t.Run("new ancestors created late", func(t *testing.T) {
		// We add a new set of edges that were created late that rival the lower-most,
		// unrivaled honest edge from before. This means that edge will no longer have
		// an ever-increasing unrivaled timer after these new edges are being tracked.
		edges = buildEdges(
			// Charlie.
			newEdge(&newCfg{t: t, edgeId: "blk-0.a-16.c", createdAt: 7}),
			newEdge(&newCfg{t: t, edgeId: "blk-0.a-8.c", createdAt: 8}),
			newEdge(&newCfg{t: t, edgeId: "blk-8.c-16.c", createdAt: 8}),
			newEdge(&newCfg{t: t, edgeId: "blk-0.a-4.c", createdAt: 9}),
			newEdge(&newCfg{t: t, edgeId: "blk-4.c-8.c", createdAt: 9}),
		)
		// Child-relationship linking.
		edges["blk-0.a-16.c"].lowerChildId = "blk-0.a-8.c"
		edges["blk-0.a-16.c"].upperChildId = "blk-8.c-16.c"
		edges["blk-0.a-8.c"].lowerChildId = "blk-0.a-4.c"
		edges["blk-0.a-8.c"].upperChildId = "blk-4.c-8.c"

		// Add the new edges into the mapping.
		for k, v := range edges {
			ht.edges.Put(id(k), v)
		}

		// Three pairs of edges are rivaled in this test: 0-16, 0-8, 0-4
		mutual := edges["blk-0.a-16.c"].MutualId()
		mutuals := ht.mutualIds.Get(mutual)
		idd := id("blk-0.a-16.c")
		mutuals.Put(idd, creationTime(ht.edges.Get(idd).CreatedAtBlock()))

		mutual = edges["blk-0.a-8.c"].MutualId()

		mutuals = ht.mutualIds.Get(mutual)
		idd = id("blk-0.a-8.c")
		mutuals.Put(idd, creationTime(ht.edges.Get(idd).CreatedAtBlock()))

		mutual = edges["blk-0.a-4.c"].MutualId()

		ht.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
		mutuals = ht.mutualIds.Get(mutual)
		idd = id("blk-0.a-4.a")
		mutuals.Put(idd, creationTime(ht.edges.Get(idd).CreatedAtBlock()))
		idd = id("blk-0.a-4.c")
		mutuals.Put(idd, creationTime(ht.edges.Get(idd).CreatedAtBlock()))

		// The path timer of the old, unrivaled edge should no longer increase
		// as it is now rivaled as of the time of the last created edge above.
		lastCreated := ht.edges.Get(id("blk-0.a-4.c"))
		edge := ht.edges.Get(id("blk-0.a-4.a"))
		timer, _, err := ht.HonestPathTimer(ctx, edge.Id(), lastCreated.CreatedAtBlock())
		require.NoError(t, err)
		ancestorTimers := uint64(2)
		require.Equal(t, PathTimer(lastCreated.CreatedAtBlock()-edge.CreatedAtBlock()+ancestorTimers), timer)

		// Should no longer increase.
		for i := 0; i < 10; i++ {
			timer, _, err := ht.HonestPathTimer(ctx, edge.Id(), lastCreated.CreatedAtBlock()+uint64(i))
			require.NoError(t, err)
			require.Equal(t, PathTimer(lastCreated.CreatedAtBlock()-edge.CreatedAtBlock()+ancestorTimers), timer)
		}
	})
}

// Tests out path timers across all challenge levels, checking the lowest, small step challenge
// edge inherits the local timers of all its honest ancestors through a cumulative update
// for confirmation purposes.
func TestPathTimer_AllChallengeLevels(t *testing.T) {
	ht := &HonestChallengeTree{
		edges:                         threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		mutualIds:                     threadsafe.NewMap[protocol.MutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
		honestBigStepLevelZeroEdges:   threadsafe.NewSlice[protocol.ReadOnlyEdge](),
		honestSmallStepLevelZeroEdges: threadsafe.NewSlice[protocol.ReadOnlyEdge](),
		metadataReader:                &mockMetadataReader{},
	}
	// Edge ids that belong to block challenges are prefixed with "blk".
	// For big step, prefixed with "big", and small step, prefixed with "smol".
	setupBlockChallengeTreeSnapshot(t, ht)
	ht.honestBlockChalLevelZeroEdge = option.Some(protocol.ReadOnlyEdge(ht.edges.Get(id("blk-0.a-16.a"))))
	claimId := "blk-4.a-5.a"
	setupBigStepChallengeSnapshot(t, ht, claimId)
	ht.honestBigStepLevelZeroEdges.Push(ht.edges.Get(id("big-0.a-16.a")))
	claimId = "big-4.a-5.a"
	setupSmallStepChallengeSnapshot(t, ht, claimId)
	ht.honestSmallStepLevelZeroEdges.Push(ht.edges.Get(id("smol-0.a-16.a")))

	ctx := context.Background()
	lastCreated := ht.edges.Get(id("smol-4.a-5.a"))
	timer, ancestors, err := ht.HonestPathTimer(ctx, lastCreated.Id(), lastCreated.CreatedAtBlock()+1)
	require.NoError(t, err)

	// Should be the sum of the unrivaled timers of honest edges along the path
	// all the way to the block challenge level. There are 15 edges in total, including the one
	// we are querying for. The assertion was unrivaled for 0 seconds. However, due to a merge move
	// made into edge with commit 4a, the edge blk-4.a-6.b from the malicious party was created
	// before blk-4.a-6.a, so 4.a-6.a was rivaled at time of creation. This means the total time
	// unrivaled is 15 - 1, which is 14.
	wantedAncestors := HonestAncestors{
		id("smol-4.a-6.a"),
		id("smol-4.a-8.a"),
		id("smol-0.a-8.a"),
		id("smol-0.a-16.a"),

		id("big-4.a-5.a"),
		id("big-4.a-6.a"),
		id("big-4.a-8.a"),
		id("big-0.a-8.a"),
		id("big-0.a-16.a"),

		id("blk-4.a-5.a"),
		id("blk-4.a-6.a"),
		id("blk-4.a-8.a"),
		id("blk-0.a-8.a"),
		id("blk-0.a-16.a"),
	}
	require.Equal(t, wantedAncestors, ancestors)

	// This gives a total of 14 seconds unrivaled along the honest path.
	require.Equal(t, PathTimer(14), timer)
}

func Test_localTimer(t *testing.T) {
	ct := &HonestChallengeTree{
		edges:     threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		mutualIds: threadsafe.NewMap[protocol.MutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
	}
	edgeA := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.a", createdAt: 3})
	ct.edges.Put(edgeA.Id(), edgeA)

	t.Run("zero if earlier than creation time", func(t *testing.T) {
		timer, err := ct.localTimer(edgeA, edgeA.creationBlock-1)
		require.NoError(t, err)
		require.Equal(t, uint64(0), timer)
	})
	t.Run("no rival is simply difference between T and creation time", func(t *testing.T) {
		timer, err := ct.localTimer(edgeA, edgeA.creationBlock)
		require.NoError(t, err)
		require.Equal(t, uint64(0), timer)
		timer, err = ct.localTimer(edgeA, edgeA.creationBlock+3)
		require.NoError(t, err)
		require.Equal(t, uint64(3), timer)
		timer, err = ct.localTimer(edgeA, edgeA.creationBlock+1000)
		require.NoError(t, err)
		require.Equal(t, uint64(1000), timer)
	})
	t.Run("if rivaled timer is difference between earliest rival and edge creation", func(t *testing.T) {
		edgeB := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.b", createdAt: 5})
		edgeC := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.c", createdAt: 10})
		ct.edges.Put(edgeB.Id(), edgeB)
		ct.edges.Put(edgeC.Id(), edgeC)
		mutual := edgeA.MutualId()

		ct.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
		mutuals := ct.mutualIds.Get(mutual)
		mutuals.Put(edgeA.Id(), creationTime(edgeA.creationBlock))
		mutuals.Put(edgeB.Id(), creationTime(edgeB.creationBlock))
		mutuals.Put(edgeC.Id(), creationTime(edgeC.creationBlock))

		// Should get same result regardless of specified time.
		timer, err := ct.localTimer(edgeA, 100)
		require.NoError(t, err)
		require.Equal(t, edgeB.creationBlock-edgeA.creationBlock, timer)
		timer, err = ct.localTimer(edgeA, 10000)
		require.NoError(t, err)
		require.Equal(t, edgeB.creationBlock-edgeA.creationBlock, timer)
		timer, err = ct.localTimer(edgeA, 1000000)
		require.NoError(t, err)
		require.Equal(t, edgeB.creationBlock-edgeA.creationBlock, timer)

		// EdgeB and EdgeC were already rivaled at creation, so they should have
		// a local timer of 0 regardless of specified time.
		timer, err = ct.localTimer(edgeB, 100)
		require.NoError(t, err)
		require.Equal(t, uint64(0), timer)
		timer, err = ct.localTimer(edgeC, 100)
		require.NoError(t, err)
		require.Equal(t, uint64(0), timer)
		timer, err = ct.localTimer(edgeB, 10000)
		require.NoError(t, err)
		require.Equal(t, uint64(0), timer)
		timer, err = ct.localTimer(edgeC, 10000)
		require.NoError(t, err)
		require.Equal(t, uint64(0), timer)
	})
}

func Test_earliestCreatedRivalBlockNumber(t *testing.T) {
	ct := &HonestChallengeTree{
		edges:     threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		mutualIds: threadsafe.NewMap[protocol.MutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
	}
	edgeA := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.a", createdAt: 3})
	edgeB := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.b", createdAt: 5})
	edgeC := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.c", createdAt: 10})
	ct.edges.Put(edgeA.Id(), edgeA)
	t.Run("no rivals", func(t *testing.T) {
		res := ct.earliestCreatedRivalBlockNumber(edgeA)

		require.Equal(t, option.None[uint64](), res)
	})
	t.Run("one rival", func(t *testing.T) {
		mutual := edgeA.MutualId()
		ct.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
		mutuals := ct.mutualIds.Get(mutual)
		mutuals.Put(edgeA.Id(), creationTime(edgeA.creationBlock))
		mutuals.Put(edgeB.Id(), creationTime(edgeB.creationBlock))
		ct.edges.Put(edgeB.Id(), edgeB)

		res := ct.earliestCreatedRivalBlockNumber(edgeA)

		require.Equal(t, uint64(5), res.Unwrap())
	})
	t.Run("multiple rivals", func(t *testing.T) {
		ct.edges.Put(edgeC.Id(), edgeC)
		mutual := edgeC.MutualId()

		mutuals := ct.mutualIds.Get(mutual)
		mutuals.Put(edgeC.Id(), creationTime(edgeC.creationBlock))

		res := ct.earliestCreatedRivalBlockNumber(edgeA)

		require.Equal(t, uint64(5), res.Unwrap())
	})
}

func Test_unrivaledAtBlockNum(t *testing.T) {
	ct := &HonestChallengeTree{
		edges:     threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		mutualIds: threadsafe.NewMap[protocol.MutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
	}
	edgeA := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.a", createdAt: 3})
	edgeB := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.b", createdAt: 5})
	ct.edges.Put(edgeA.Id(), edgeA)
	t.Run("less than specified time", func(t *testing.T) {
		_, err := ct.unrivaledAtBlockNum(edgeA, 0)
		require.ErrorContains(t, err, "less than specified")
	})
	t.Run("no rivals", func(t *testing.T) {
		unrivaled, err := ct.unrivaledAtBlockNum(edgeA, 3)
		require.NoError(t, err)
		require.Equal(t, true, unrivaled)
		unrivaled, err = ct.unrivaledAtBlockNum(edgeA, 1000)
		require.NoError(t, err)
		require.Equal(t, true, unrivaled)
	})
	t.Run("with rivals but unrivaled at creation time", func(t *testing.T) {
		mutual := edgeA.MutualId()
		ct.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
		mutuals := ct.mutualIds.Get(mutual)
		mutuals.Put(edgeA.Id(), creationTime(edgeA.creationBlock))
		mutuals.Put(edgeB.Id(), creationTime(edgeB.creationBlock))
		ct.edges.Put(edgeB.Id(), edgeB)

		unrivaled, err := ct.unrivaledAtBlockNum(edgeA, 3)
		require.NoError(t, err)
		require.Equal(t, true, unrivaled)
	})
	t.Run("rivaled at first rival creation time", func(t *testing.T) {
		unrivaled, err := ct.unrivaledAtBlockNum(edgeA, 5)
		require.NoError(t, err)
		require.Equal(t, false, unrivaled)
		unrivaled, err = ct.unrivaledAtBlockNum(edgeB, 5)
		require.NoError(t, err)
		require.Equal(t, false, unrivaled)
	})
}

func Test_rivalsWithCreationTimes(t *testing.T) {
	ct := &HonestChallengeTree{
		edges:     threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		mutualIds: threadsafe.NewMap[protocol.MutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
	}
	edgeA := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.a", createdAt: 5})
	edgeB := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.b", createdAt: 5})
	edgeC := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.c", createdAt: 10})
	ct.edges.Put(edgeA.Id(), edgeA)
	t.Run("no rivals", func(t *testing.T) {
		rivals := ct.rivalsWithCreationTimes(edgeA)

		require.Equal(t, 0, len(rivals))
	})
	t.Run("single rival", func(t *testing.T) {
		mutual := edgeA.MutualId()
		ct.mutualIds.Put(mutual, threadsafe.NewMap[protocol.EdgeId, creationTime]())
		mutuals := ct.mutualIds.Get(mutual)
		mutuals.Put(edgeB.Id(), creationTime(edgeB.creationBlock))
		mutuals.Put(edgeA.Id(), creationTime(edgeA.creationBlock))
		ct.edges.Put(edgeB.Id(), edgeB)
		rivals := ct.rivalsWithCreationTimes(edgeA)

		want := []*rival{
			{id: edgeB.Id(), createdAtBlock: creationTime(edgeB.creationBlock)},
		}
		require.Equal(t, want, rivals)
		rivals = ct.rivalsWithCreationTimes(edgeB)

		want = []*rival{
			{id: edgeA.Id(), createdAtBlock: creationTime(edgeA.creationBlock)},
		}
		require.Equal(t, want, rivals)
	})
	t.Run("multiple rivals", func(t *testing.T) {
		ct.edges.Put(edgeC.Id(), edgeC)
		mutual := edgeC.MutualId()
		mutuals := ct.mutualIds.Get(mutual)
		mutuals.Put(edgeC.Id(), creationTime(edgeC.creationBlock))
		want := []edgeId{edgeA.id, edgeB.id}
		rivals := ct.rivalsWithCreationTimes(edgeC)

		require.Equal(t, true, len(rivals) > 0)
		got := make(map[protocol.EdgeId]bool)
		for _, r := range rivals {
			got[r.id] = true
		}
		for _, w := range want {
			require.Equal(t, true, got[id(w)])
		}
	})
}
