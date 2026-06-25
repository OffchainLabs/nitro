// Copyright 2025-2026, Offchain Labs, Inc.
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

func TestMultiGasFeesCommitNextToCurrent(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	baseFees := OpenMultiGasFees(sto)

	for i := range int(multigas.NumResourceKind) {
		// #nosec G115 safe: NumResourceKind < 2^32
		kind := multigas.ResourceKind(i)

		next, err := baseFees.GetNextBlockFee(kind)
		require.NoError(t, err)
		require.Zero(t, next.Sign())

		current, err := baseFees.GetCurrentBlockFee(kind)
		require.NoError(t, err)
		require.Zero(t, current.Sign())
	}

	for i := range int(multigas.NumResourceKind) {
		// #nosec G115 safe: NumResourceKind < 2^32
		kind := multigas.ResourceKind(i)
		err := baseFees.SetNextBlockFee(kind, big.NewInt(int64(i+1)))
		require.NoError(t, err)
	}

	err := baseFees.CommitNextToCurrent()
	require.NoError(t, err)

	// Re-open to ensure values are persisted.
	baseFees = OpenMultiGasFees(sto)

	for i := range int(multigas.NumResourceKind) {
		// #nosec G115 safe: NumResourceKind < 2^32
		kind := multigas.ResourceKind(i)

		current, err := baseFees.GetCurrentBlockFee(kind)
		require.NoError(t, err)
		expected := big.NewInt(int64(i + 1))
		require.Zero(t, current.Cmp(expected))
	}
}
