package challengetree

import (
	"container/heap"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUint64HeapWithTestify(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	h := &uint64Heap{}
	heap.Init(h)

	// Test Push
	heap.Push(h, uint64(5))
	heap.Push(h, uint64(3))
	heap.Push(h, uint64(10))
	require.Equal(uint64(3), h.Peek(), "Peek should return the smallest element after push operations")

	// Test Pop
	val := heap.Pop(h).(uint64)
	assert.Equal(uint64(3), val, "Pop should return the smallest element")
	assert.Equal(uint64(5), h.Peek(), "Peek should now return the next smallest element")

	// Ensure proper order on subsequent Pops
	heap.Push(h, uint64(1))
	val = heap.Pop(h).(uint64)
	require.Equal(uint64(1), val, "Pop should return the smallest element, even after subsequent pushes")
	val = heap.Pop(h).(uint64)
	assert.Equal(uint64(5), val, "Expect the next smallest element")
	val = heap.Pop(h).(uint64)
	assert.Equal(uint64(10), val, "Expect the next smallest element")
}

func TestPathMinHeapWithTestify(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	h := newPathMinHeap()

	// Test Push and Peek
	h.Push(8)
	result := h.Peek()
	require.False(result.IsNone(), "Peek should not return None after a push")
	require.Equal(uint64(8), result.Unwrap(), "Peek should return the value pushed")

	// Test Pop
	h.Push(3)
	val := h.Pop(3)
	assert.Equal(uint64(3), val, "Pop should return the value just pushed")

	result = h.Peek()
	require.False(result.IsNone(), "Peek should not return None after pop")
	require.Equal(uint64(8), result.Unwrap(), "Peek should return the next smallest value")

	// Test Peek with empty heap
	h.Pop(8) // Empty the heap
	result = h.Peek()
	assert.True(result.IsNone(), "Peek should return None when the heap is empty")
}

func TestPathTracker(t *testing.T) {
	p := newPathTracker()
	_, _, _ = p.isConfirmable(isConfirmableArgs{})
}
