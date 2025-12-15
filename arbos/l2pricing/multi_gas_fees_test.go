// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l2pricing

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
)

func TestMultiGasFeesCommitCurrentToLast(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	baseFees := OpenMultiGasFees(sto)

	for i := range int(multigas.NumResourceKind) {
		// #nosec G115 safe: NumResourceKind < 2^32
		kind := multigas.ResourceKind(i)

		cur, err := baseFees.GetCurrent(kind)
		require.NoError(t, err)
		require.Zero(t, cur.Sign())

		last, err := baseFees.GetLast(kind)
		require.NoError(t, err)
		require.Zero(t, last.Sign())
	}

	for i := range int(multigas.NumResourceKind) {
		// #nosec G115 safe: NumResourceKind < 2^32
		kind := multigas.ResourceKind(i)
		err := baseFees.SetCurrent(kind, big.NewInt(int64(i+1)))
		require.NoError(t, err)
	}

	err := baseFees.CommitCurrentToLast()
	require.NoError(t, err)

	// Re-open to ensure values are persisted.
	baseFees = OpenMultiGasFees(sto)

	for i := range int(multigas.NumResourceKind) {
		// #nosec G115 safe: NumResourceKind < 2^32
		kind := multigas.ResourceKind(i)

		cur, err := baseFees.GetCurrent(kind)
		require.NoError(t, err)
		require.Zero(t, cur.Sign())

		last, err := baseFees.GetLast(kind)
		require.NoError(t, err)
		expected := big.NewInt(int64(i + 1))
		require.Zero(t, last.Cmp(expected))
	}

	err = baseFees.CommitCurrentToLast()
	require.NoError(t, err)

	for i := range int(multigas.NumResourceKind) {
		// #nosec G115 safe: NumResourceKind < 2^32
		kind := multigas.ResourceKind(i)

		cur, err := baseFees.GetCurrent(kind)
		require.NoError(t, err)
		require.Zero(t, cur.Sign())

		last, err := baseFees.GetLast(kind)
		require.NoError(t, err)
		require.Zero(t, last.Sign())
	}
}
