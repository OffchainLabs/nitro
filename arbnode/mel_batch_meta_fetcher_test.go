// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMelBatchMetaFetcher_GetDelayedAccAlwaysErrors(t *testing.T) {
	t.Parallel()
	// melBatchMetaFetcher.GetDelayedAcc should always return an error
	// since MEL does not track delayed message accumulators.
	fetcher := &melBatchMetaFetcher{} // extractor not needed for GetDelayedAcc
	_, err := fetcher.GetDelayedAcc(0)
	require.ErrorContains(t, err, "MEL does not support delayed message accumulators")
}

func TestNewMelBatchMetaFetcher_PanicsOnNil(t *testing.T) {
	t.Parallel()
	require.Panics(t, func() {
		newMelBatchMetaFetcher(nil)
	})
}
