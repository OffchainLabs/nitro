// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package warmbuffer

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

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
