package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestThreadSafeSlice(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		s := &ThreadSafeSlice[int]{
			items: nil,
		}
		require.Equal(t, None[int](), s.Get(0))
		require.Equal(t, None[int](), s.Last())
		require.Equal(t, true, s.Empty())
		require.Equal(t, 0, s.Len())
	})
	t.Run("appending to empty", func(t *testing.T) {
		s := &ThreadSafeSlice[int]{
			items: nil,
		}
		s.Append(1)
		require.Equal(t, Some[int](1), s.Get(0))
		require.Equal(t, Some[int](1), s.Last())
		require.Equal(t, false, s.Empty())
		require.Equal(t, 1, s.Len())
	})
	t.Run("appending to existing", func(t *testing.T) {
		s := &ThreadSafeSlice[int]{
			items: []int{1, 2, 3},
		}
		s.Append(4)
		require.Equal(t, Some[int](1), s.Get(0))
		require.Equal(t, Some[int](4), s.Last())
		require.Equal(t, false, s.Empty())
		require.Equal(t, 4, s.Len())
	})
}
