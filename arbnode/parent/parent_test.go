// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package parent

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/params"
)

func TestBlobConfigSelectsNextWhenHeaderTimePastActivation(t *testing.T) {
	now := time.Now()
	// #nosec G115
	currentActivation := uint64(now.Add(-1 * time.Hour).Unix()) // activated 1h ago
	// #nosec G115
	nextActivation := uint64(now.Add(10 * time.Minute).Unix()) // activates in 10 min

	currentBlob := &params.BlobConfig{Target: 6, Max: 9, UpdateFraction: 5007716}
	nextBlob := &params.BlobConfig{Target: 10, Max: 15, UpdateFraction: 8346193}

	currentEthConfigEntry := ethConfigEntry{
		BlobSchedule:   currentBlob,
		ActivationTime: currentActivation,
	}

	nextEthConfigEntry := ethConfigEntry{
		BlobSchedule:   nextBlob,
		ActivationTime: nextActivation,
	}

	pc := &ParentChain{ChainID: big.NewInt(1)}
	pc.cachedEthConfig.Store(&ethConfigResponse{
		Current: &currentEthConfigEntry,
		Next:    &nextEthConfigEntry,
	})

	// Header before next activation → should return current
	// #nosec G115
	headerBeforeNext := uint64(now.Unix()) // now is before nextActivation
	got, err := pc.blobConfig(headerBeforeNext)
	require.NoError(t, err)
	require.Equal(t, currentBlob.Target, got.Target, "expected current target")
	require.Equal(t, currentBlob.Max, got.Max, "expected current max")

	// Header at or after next activation → should return next
	headerAfterNext := nextActivation // exactly at activation
	got, err = pc.blobConfig(headerAfterNext)
	require.NoError(t, err)
	require.Equal(t, nextBlob.Target, got.Target, "expected next target")
	require.Equal(t, nextBlob.Max, got.Max, "expected next max")
}

func TestBlobConfigFallsBackToStaticWhenHeaderBeforeCurrentActivation(t *testing.T) {
	now := time.Now()
	// #nosec G115
	currentActivation := uint64(now.Add(-30 * time.Minute).Unix()) // activated 30 min ago

	currentBlob := &params.BlobConfig{Target: 99, Max: 99, UpdateFraction: 99}

	currentEthConfigEntry := ethConfigEntry{
		BlobSchedule:   currentBlob,
		ActivationTime: currentActivation,
	}

	// Use a known chain ID so static config lookup works
	pc := &ParentChain{ChainID: big.NewInt(1)} // mainnet
	pc.cachedEthConfig.Store(&ethConfigResponse{
		Current: &currentEthConfigEntry,
	})

	// Header from before the current config's activation but still recent
	// enough to be within cachedBlobConfigMaxAge (12h)
	// #nosec G115
	headerTime := uint64(now.Add(-1 * time.Hour).Unix()) // 1h ago, before currentActivation
	got, err := pc.blobConfig(headerTime)
	require.NoError(t, err)
	// Should NOT be the cached config (target=99), should be from static mainnet config
	expectedFork := params.MainnetChainConfig.LatestFork(headerTime, 0)
	expectedBlob := params.MainnetChainConfig.BlobConfig(expectedFork)
	require.NotNil(t, expectedBlob)
	require.NotEqual(t, expectedBlob.Target, currentBlob.Target, "expected blob target should not be equal to current blob")
	require.NotEqual(t, expectedBlob.Max, currentBlob.Max, "expected blob max should not be equal to current blob")
	require.Equal(t, expectedBlob.Target, got.Target, "expected static mainnet target")
	require.Equal(t, expectedBlob.Max, got.Max, "expected static mainnet max")
	require.Equal(t, expectedBlob.UpdateFraction, got.UpdateFraction, "expected static mainnet updateFraction")
}

func TestBlobConfigNoNextField(t *testing.T) {
	now := time.Now()
	// #nosec G115
	currentActivation := uint64(now.Add(-1 * time.Hour).Unix())

	currentBlob := &params.BlobConfig{Target: 6, Max: 9, UpdateFraction: 5007716}

	pc := &ParentChain{ChainID: big.NewInt(1)}
	pc.cachedEthConfig.Store(&ethConfigResponse{
		Current: &ethConfigEntry{
			BlobSchedule:   currentBlob,
			ActivationTime: currentActivation,
		},
		// Next is nil — no upcoming fork
	})

	// #nosec G115
	headerTime := uint64(now.Unix())
	got, err := pc.blobConfig(headerTime)
	require.NoError(t, err)
	require.Equal(t, currentBlob.Target, got.Target, "expected current target")
	require.Equal(t, currentBlob.Max, got.Max, "expected current max")
}
