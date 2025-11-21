// Copyright 2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

// Package casttest exposes test helper functions to wrap safecast calls.
package casttest

import (
	"testing"

	"github.com/ccoveille/go-safecast"
	"github.com/stretchr/testify/require"
)

// ToUint wraps safecast.ToUint with a test assertion.
func ToUint[T safecast.Type](t testing.TB, i T) uint {
	t.Helper()
	u, err := safecast.ToUint(i)
	require.NoError(t, err)
	return u
}

// ToUint8 wraps safecast.ToUint8 with a test assertion.
func ToUint8[T safecast.Type](t testing.TB, i T) uint8 {
	t.Helper()
	u, err := safecast.ToUint8(i)
	require.NoError(t, err)
	return u
}

// ToUint64 wraps safecast.ToUint64 with a test assertion.
func ToUint64[T safecast.Type](t testing.TB, i T) uint64 {
	t.Helper()
	u, err := safecast.ToUint64(i)
	require.NoError(t, err)
	return u
}

// ToInt wraps safecast.ToInt with a test assertion.
func ToInt[T safecast.Type](t testing.TB, i T) int {
	t.Helper()
	u, err := safecast.ToInt(i)
	require.NoError(t, err)
	return u
}

// ToInt64 wraps safecast.ToInt64 with a test assertion.
func ToInt64[T safecast.Type](t testing.TB, i T) int64 {
	t.Helper()
	u, err := safecast.ToInt64(i)
	require.NoError(t, err)
	return u
}
