// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/challenge-protocol-v2/blob/main/LICENSE

package threadsafe

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMap(t *testing.T) {
	m := NewMap[string, uint64]()
	t.Run("Get", func(t *testing.T) {
		_, ok := m.TryGet("foo")
		require.Equal(t, false, ok)
		m.Put("foo", 5)
		got, ok := m.TryGet("foo")
		require.Equal(t, true, ok)
		require.Equal(t, uint64(5), got)
		require.Equal(t, uint64(5), m.Get("foo"))
	})
	t.Run("Delete", func(t *testing.T) {
		m.Delete("foo")
		_, ok := m.TryGet("foo")
		require.Equal(t, false, ok)
	})
	t.Run("ForEach", func(t *testing.T) {
		m.Put("foo", 5)
		m.Put("bar", 10)
		var total uint64
		err := m.ForEach(func(_ string, v uint64) error {
			total += v
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, uint64(15), total)
	})
}

func TestSet(t *testing.T) {
	m := NewSet[uint64]()
	t.Run("Has", func(t *testing.T) {
		ok := m.Has(5)
		require.Equal(t, false, ok)
		m.Insert(5)
		ok = m.Has(5)
		require.Equal(t, true, ok)
	})
	t.Run("Delete", func(t *testing.T) {
		m.Delete(5)
		ok := m.Has(5)
		require.Equal(t, false, ok)
	})
	t.Run("ForEach", func(t *testing.T) {
		m.Insert(5)
		m.Insert(10)
		var total uint64
		m.ForEach(func(elem uint64) {
			total += elem
		})
		require.Equal(t, uint64(15), total)
	})
}
