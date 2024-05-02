package challengetree

import (
	"context"
	"testing"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/threadsafe"
	"github.com/stretchr/testify/require"
)

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

func TestIsConfirmableEssentialNode(t *testing.T) {
	// aliceEdges := buildEdges(
	// 	// Alice.
	// 	newEdge(&newCfg{t: t, edgeId: "blk-0.a-16.a", claimId: claimId, createdAt: 1}),
	// 	newEdge(&newCfg{t: t, edgeId: "blk-0.a-8.a", createdAt: 3}),
	// 	newEdge(&newCfg{t: t, edgeId: "blk-8.a-16.a", createdAt: 3}),
	// 	newEdge(&newCfg{t: t, edgeId: "blk-0.a-4.a", createdAt: 5}),
	// 	newEdge(&newCfg{t: t, edgeId: "blk-4.a-8.a", createdAt: 5}),
	// 	newEdge(&newCfg{t: t, edgeId: "blk-4.a-6.a", createdAt: 7}),
	// 	newEdge(&newCfg{t: t, edgeId: "blk-6.a-8.a", createdAt: 7}),
	// 	newEdge(&newCfg{t: t, edgeId: "blk-4.a-5.a", createdAt: 9}),
	// 	newEdge(&newCfg{t: t, edgeId: "blk-5.a-6.a", createdAt: 9}),
	// )
	// bobEdges := buildEdges(
	// 	// Bob.
	// 	newEdge(&newCfg{t: t, edgeId: "blk-0.a-16.b", createdAt: 2}),
	// 	newEdge(&newCfg{t: t, edgeId: "blk-0.a-8.b", createdAt: 4}),
	// 	newEdge(&newCfg{t: t, edgeId: "blk-8.b-16.b", createdAt: 4}),
	// 	newEdge(&newCfg{t: t, edgeId: "blk-4.a-8.b", createdAt: 6}),
	// 	newEdge(&newCfg{t: t, edgeId: "blk-4.a-6.b", createdAt: 6}),
	// 	newEdge(&newCfg{t: t, edgeId: "blk-6.b-8.b", createdAt: 8}),
	// 	newEdge(&newCfg{t: t, edgeId: "blk-4.a-5.b", createdAt: 10}),
	// 	newEdge(&newCfg{t: t, edgeId: "blk-5.b-6.b", createdAt: 10}),
	// )
	// // Child-relationship linking.
	// // Alice.
	// aliceEdges["blk-0.a-16.a"].LowerChildID = "blk-0.a-8.a"
	// aliceEdges["blk-0.a-16.a"].UpperChildID = "blk-8.a-16.a"
	// aliceEdges["blk-0.a-8.a"].LowerChildID = "blk-0.a-4.a"
	// aliceEdges["blk-0.a-8.a"].UpperChildID = "blk-4.a-8.a"
	// aliceEdges["blk-4.a-8.a"].LowerChildID = "blk-4.a-6.a"
	// aliceEdges["blk-4.a-8.a"].UpperChildID = "blk-6.a-8.a"
	// aliceEdges["blk-4.a-6.a"].LowerChildID = "blk-4.a-5.a"
	// aliceEdges["blk-4.a-6.a"].UpperChildID = "blk-5.a-6.a"
	// // Bob.
	// bobEdges["blk-0.a-16.b"].LowerChildID = "blk-0.a-8.b"
	// bobEdges["blk-0.a-16.b"].UpperChildID = "blk-8.b-16.b"
	// bobEdges["blk-0.a-8.b"].LowerChildID = "blk-0.a-4.a"
	// bobEdges["blk-0.a-8.b"].UpperChildID = "blk-4.a-8.b"
	// bobEdges["blk-4.a-8.b"].LowerChildID = "blk-4.a-6.b"
	// bobEdges["blk-4.a-8.b"].UpperChildID = "blk-6.b-6.8"
	// bobEdges["blk-4.a-6.b"].LowerChildID = "blk-4.a-5.b"
	// bobEdges["blk-4.a-6.b"].UpperChildID = "blk-5.b-6.b"

	// transformedEdges := make(map[protocol.EdgeId]protocol.SpecEdge)
	// for _, v := range aliceEdges {
	// 	transformedEdges[v.Id()] = v
	// }
	// allEdges := threadsafe.NewMapFromItems(transformedEdges)
	// tree.edges = allEdges
}
