package challengetree

import (
	"container/heap"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/option"
)

type uint64Heap []uint64

func (h uint64Heap) Len() int           { return len(h) }
func (h uint64Heap) Less(i, j int) bool { return h[i] < h[j] }
func (h uint64Heap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h uint64Heap) Peek() uint64 {
	return h[0]
}

func (h *uint64Heap) Push(x any) {
	*h = append(*h, x.(uint64))
}

func (h *uint64Heap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type pathMinHeap struct {
	items *uint64Heap
}

func newPathMinHeap() *pathMinHeap {
	items := &uint64Heap{}
	heap.Init(items)
	return &pathMinHeap{items}
}

func (h *pathMinHeap) Push(item uint64) {
	heap.Push(h.items, item)
}

func (h *pathMinHeap) Pop(item uint64) uint64 {
	return heap.Pop(h.items).(uint64)
}

func (h *pathMinHeap) Peek() option.Option[uint64] {
	if h.items.Len() == 0 {
		return option.None[uint64]()
	}
	return option.Some(h.items.Peek())
}

type pathTracker struct {
	essentialNodePathWeights map[protocol.EdgeId]*pathMinHeap
}

func newPathTracker() *pathTracker {
	return &pathTracker{
		essentialNodePathWeights: make(map[protocol.EdgeId]*pathMinHeap),
	}
}

type essentialPath []protocol.EdgeId

type isConfirmableArgs struct {
	essentialNode         protocol.EdgeId
	confirmationThreshold uint64
	blockNum              uint64
}

// Find all the paths down from an essential node, and
// compute the local timer of each edge along the path. This is
// a recursive computation that goes down the tree rooted at the essential
// node and ends once it finds edges that either do not have children,
// or are terminal nodes that end in children that are incorrectly constructed
// or non-essential.
//
// After the paths are computed, we then compute the path weight of each
// and insert each weight into a min-heap. If the min element of this heap
// has a weight >= the confirmation threshold, the
// essential node is then confirmable.
func (p *pathTracker) isConfirmable(
	args isConfirmableArgs,
) (bool, []essentialPath, error) {
	return false, nil, nil
}
