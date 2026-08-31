// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !wasm

package programs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStorageCacheLimitResult(t *testing.T) {
	data, msg, err := userStorageCacheLimitExceeded.toResult(nil, false)
	require.Nil(t, data)
	require.Equal(t, ErrStorageCacheLimitExceeded.Error(), msg)
	require.ErrorIs(t, err, ErrStorageCacheLimitExceeded)
}

func TestNativeStackOverflowResult(t *testing.T) {
	data, msg, err := userNativeStackOverflow.toResult([]byte("host failure"), false)
	require.Nil(t, data)
	require.Empty(t, msg)
	require.ErrorIs(t, err, ErrNativeStackOverflow)
}

func TestStylusSystemResult(t *testing.T) {
	data, msg, err := userSystemError.toResult([]byte("host failure"), false)
	require.Nil(t, data)
	require.Equal(t, "host failure", msg)
	require.ErrorIs(t, err, ErrStylusSystem)
}
