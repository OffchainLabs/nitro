// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package challengetree

import (
	"testing"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/challenge-manager/challenge-tree/mock"
	"github.com/OffchainLabs/bold/containers/option"
	"github.com/OffchainLabs/bold/containers/threadsafe"
	"github.com/stretchr/testify/require"
)

func Test_localTimer(t *testing.T) {
	ct := &RoyalChallengeTree{
		edges:             threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		edgeCreationTimes: threadsafe.NewMap[OriginPlusMutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
	}
	edgeA := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.a", createdAt: 3})
	ct.edges.Put(edgeA.Id(), edgeA)

	t.Run("zero if earlier than creation time", func(t *testing.T) {
		timer, err := ct.LocalTimer(edgeA, edgeA.CreationBlock-1)
		require.NoError(t, err)
		require.Equal(t, uint64(0), timer)
	})
	t.Run("no rival is simply difference between T and creation time", func(t *testing.T) {
		timer, err := ct.LocalTimer(edgeA, edgeA.CreationBlock)
		require.NoError(t, err)
		require.Equal(t, uint64(0), timer)
		timer, err = ct.LocalTimer(edgeA, edgeA.CreationBlock+3)
		require.NoError(t, err)
		require.Equal(t, uint64(3), timer)
		timer, err = ct.LocalTimer(edgeA, edgeA.CreationBlock+1000)
		require.NoError(t, err)
		require.Equal(t, uint64(1000), timer)
	})
	t.Run("if rivaled timer is difference between earliest rival and edge creation", func(t *testing.T) {
		edgeB := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.b", createdAt: 5})
		edgeC := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.c", createdAt: 10})
		ct.edges.Put(edgeB.Id(), edgeB)
		ct.edges.Put(edgeC.Id(), edgeC)
		mutual := edgeA.MutualId()

		key := buildEdgeCreationTimeKey(protocol.OriginId{}, mutual)
		ct.edgeCreationTimes.Put(key, threadsafe.NewMap[protocol.EdgeId, creationTime]())
		mutuals := ct.edgeCreationTimes.Get(key)
		mutuals.Put(edgeA.Id(), creationTime(edgeA.CreationBlock))
		mutuals.Put(edgeB.Id(), creationTime(edgeB.CreationBlock))
		mutuals.Put(edgeC.Id(), creationTime(edgeC.CreationBlock))

		// Should get same result regardless of specified time.
		timer, err := ct.LocalTimer(edgeA, 100)
		require.NoError(t, err)
		require.Equal(t, edgeB.CreationBlock-edgeA.CreationBlock, timer)
		timer, err = ct.LocalTimer(edgeA, 10000)
		require.NoError(t, err)
		require.Equal(t, edgeB.CreationBlock-edgeA.CreationBlock, timer)
		timer, err = ct.LocalTimer(edgeA, 1000000)
		require.NoError(t, err)
		require.Equal(t, edgeB.CreationBlock-edgeA.CreationBlock, timer)

		// EdgeB and EdgeC were already rivaled at creation, so they should have
		// a local timer of 0 regardless of specified time.
		timer, err = ct.LocalTimer(edgeB, 100)
		require.NoError(t, err)
		require.Equal(t, uint64(0), timer)
		timer, err = ct.LocalTimer(edgeC, 100)
		require.NoError(t, err)
		require.Equal(t, uint64(0), timer)
		timer, err = ct.LocalTimer(edgeB, 10000)
		require.NoError(t, err)
		require.Equal(t, uint64(0), timer)
		timer, err = ct.LocalTimer(edgeC, 10000)
		require.NoError(t, err)
		require.Equal(t, uint64(0), timer)
	})
}

func Test_earliestCreatedRivalBlockNumber(t *testing.T) {
	ct := &RoyalChallengeTree{
		edges:             threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		edgeCreationTimes: threadsafe.NewMap[OriginPlusMutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
	}
	edgeA := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.a", createdAt: 3})
	edgeB := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.b", createdAt: 5})
	edgeC := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.c", createdAt: 10})
	ct.edges.Put(edgeA.Id(), edgeA)
	t.Run("no rivals", func(t *testing.T) {
		res := ct.EarliestCreatedRivalBlockNumber(edgeA)

		require.Equal(t, option.None[uint64](), res)
	})
	t.Run("one rival", func(t *testing.T) {
		mutual := edgeA.MutualId()
		key := buildEdgeCreationTimeKey(protocol.OriginId{}, mutual)
		ct.edgeCreationTimes.Put(key, threadsafe.NewMap[protocol.EdgeId, creationTime]())
		mutuals := ct.edgeCreationTimes.Get(key)
		mutuals.Put(edgeA.Id(), creationTime(edgeA.CreationBlock))
		mutuals.Put(edgeB.Id(), creationTime(edgeB.CreationBlock))
		ct.edges.Put(edgeB.Id(), edgeB)

		res := ct.EarliestCreatedRivalBlockNumber(edgeA)

		require.Equal(t, uint64(5), res.Unwrap())
	})
	t.Run("multiple rivals", func(t *testing.T) {
		ct.edges.Put(edgeC.Id(), edgeC)
		mutual := edgeC.MutualId()

		key := buildEdgeCreationTimeKey(protocol.OriginId{}, mutual)
		mutuals := ct.edgeCreationTimes.Get(key)
		mutuals.Put(edgeC.Id(), creationTime(edgeC.CreationBlock))

		res := ct.EarliestCreatedRivalBlockNumber(edgeA)

		require.Equal(t, uint64(5), res.Unwrap())
	})
}

func Test_unrivaledAtBlockNum(t *testing.T) {
	ct := &RoyalChallengeTree{
		edges:             threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		edgeCreationTimes: threadsafe.NewMap[OriginPlusMutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
	}
	edgeA := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.a", createdAt: 3})
	edgeB := newEdge(&newCfg{t: t, edgeId: "blk-0.a-1.b", createdAt: 5})
	ct.edges.Put(edgeA.Id(), edgeA)
	t.Run("less than specified time", func(t *testing.T) {
		_, err := ct.UnrivaledAtBlockNum(edgeA, 0)
		require.ErrorContains(t, err, "less than specified")
	})
	t.Run("no rivals", func(t *testing.T) {
		unrivaled, err := ct.UnrivaledAtBlockNum(edgeA, 3)
		require.NoError(t, err)
		require.Equal(t, true, unrivaled)
		unrivaled, err = ct.UnrivaledAtBlockNum(edgeA, 1000)
		require.NoError(t, err)
		require.Equal(t, true, unrivaled)
	})
	t.Run("with rivals but unrivaled at creation time", func(t *testing.T) {
		mutual := edgeA.MutualId()
		key := buildEdgeCreationTimeKey(protocol.OriginId{}, mutual)
		ct.edgeCreationTimes.Put(key, threadsafe.NewMap[protocol.EdgeId, creationTime]())
		mutuals := ct.edgeCreationTimes.Get(key)
		mutuals.Put(edgeA.Id(), creationTime(edgeA.CreationBlock))
		mutuals.Put(edgeB.Id(), creationTime(edgeB.CreationBlock))
		ct.edges.Put(edgeB.Id(), edgeB)

		unrivaled, err := ct.UnrivaledAtBlockNum(edgeA, 3)
		require.NoError(t, err)
		require.Equal(t, true, unrivaled)
	})
	t.Run("rivaled at first rival creation time", func(t *testing.T) {
		unrivaled, err := ct.UnrivaledAtBlockNum(edgeA, 5)
		require.NoError(t, err)
		require.Equal(t, false, unrivaled)
		unrivaled, err = ct.UnrivaledAtBlockNum(edgeB, 5)
		require.NoError(t, err)
		require.Equal(t, false, unrivaled)
	})
}

func Test_rivalsWithCreationTimes(t *testing.T) {
	ct := &RoyalChallengeTree{
		edges:             threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		edgeCreationTimes: threadsafe.NewMap[OriginPlusMutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
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
		key := buildEdgeCreationTimeKey(protocol.OriginId{}, mutual)
		ct.edgeCreationTimes.Put(key, threadsafe.NewMap[protocol.EdgeId, creationTime]())
		mutuals := ct.edgeCreationTimes.Get(key)
		mutuals.Put(edgeB.Id(), creationTime(edgeB.CreationBlock))
		mutuals.Put(edgeA.Id(), creationTime(edgeA.CreationBlock))
		ct.edges.Put(edgeB.Id(), edgeB)
		rivals := ct.rivalsWithCreationTimes(edgeA)

		want := []*rival{
			{id: edgeB.Id(), createdAtBlock: creationTime(edgeB.CreationBlock)},
		}
		require.Equal(t, want, rivals)
		rivals = ct.rivalsWithCreationTimes(edgeB)

		want = []*rival{
			{id: edgeA.Id(), createdAtBlock: creationTime(edgeA.CreationBlock)},
		}
		require.Equal(t, want, rivals)
	})
	t.Run("multiple rivals", func(t *testing.T) {
		ct.edges.Put(edgeC.Id(), edgeC)
		mutual := edgeC.MutualId()
		key := buildEdgeCreationTimeKey(protocol.OriginId{}, mutual)
		mutuals := ct.edgeCreationTimes.Get(key)
		mutuals.Put(edgeC.Id(), creationTime(edgeC.CreationBlock))
		want := []mock.EdgeId{edgeA.ID, edgeB.ID}
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
