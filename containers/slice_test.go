package containers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReverse(t *testing.T) {
	type testCase[T any] struct {
		items  []T
		wanted []T
	}
	testCases := []testCase[uint64]{
		{
			items:  []uint64{},
			wanted: []uint64{},
		},
		{
			items:  []uint64{1},
			wanted: []uint64{1},
		},
		{
			items:  []uint64{1, 2, 3},
			wanted: []uint64{3, 2, 1},
		},
	}
	for _, tt := range testCases {
		items := tt.items
		Reverse(items)
		require.Equal(t, tt.wanted, items)
	}
}
