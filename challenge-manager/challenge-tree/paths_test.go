package challengetree

import (
	"testing"

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

func TestIsConfirmableEssentialNode(t *testing.T) {

}
