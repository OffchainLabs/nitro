// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package warmbuffer

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMakeWarmArray(t *testing.T) {
	// An element larger than one byte exercises the per-page stride.
	type hash [32]byte
	a := MakeWarmArray[hash](1000)
	require.Len(t, a, 1000)
	require.Equal(t, hash{}, a[0])
	require.Equal(t, hash{}, a[999])
	a[500][0] = 1 // usable
	require.Equal(t, byte(1), a[500][0])

	// Byte slices spanning several pages and the empty case must not panic.
	require.Len(t, MakeWarmArray[byte](4096*3+7), 4096*3+7)
	require.Len(t, MakeWarmArray[byte](0), 0)
}

func TestMakeWarmBuffer(t *testing.T) {
	const size = 1024 * 1024
	b := MakeWarmBuffer(size)
	require.Len(t, b, size)
	require.Equal(t, byte(0), b[0])
	require.Equal(t, byte(0), b[size-1])
	require.Len(t, MakeWarmBuffer(0), 0)
}

func TestMakeWarmMapLeavesEmptyUsableMap(t *testing.T) {
	calls := 0
	var counter uint64
	m := MakeWarmMap[[32]byte, struct{}](100, func() [32]byte {
		var k [32]byte
		binary.LittleEndian.PutUint64(k[:8], counter)
		counter++
		calls++
		return k
	})
	require.Equal(t, 100, calls, "nextKey should be called capacity times")
	require.Equal(t, 0, len(m), "warmed map should be empty")

	var k [32]byte
	m[k] = struct{}{}
	_, ok := m[k]
	require.True(t, ok, "map should be usable after warming")
}
