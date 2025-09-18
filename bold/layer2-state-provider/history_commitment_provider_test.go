// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package l2stateprovider

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/offchainlabs/nitro/bold/containers/option"
)

func Test_computeRequiredNumberOfHashes(t *testing.T) {
	provider := &HistoryCommitmentProvider{
		challengeLeafHeights: []Height{
			4,
			8,
			16,
		},
	}

	challengeLevel := uint64(0)
	_, err := provider.computeRequiredNumberOfHashes(
		challengeLevel,
		option.Some(Height(5)),
	)
	require.ErrorContains(t, err, "end 5 was greater than max height for level 4")

	got, err := provider.computeRequiredNumberOfHashes(
		challengeLevel,
		option.Some(Height(4)),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(5), got)

	challengeLevel = uint64(1)
	got, err = provider.computeRequiredNumberOfHashes(
		challengeLevel,
		option.Some(Height(4)),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(5), got)

	got, err = provider.computeRequiredNumberOfHashes(
		challengeLevel,
		option.None[Height](),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(9), got)

	challengeLevel = uint64(2)
	got, err = provider.computeRequiredNumberOfHashes(
		challengeLevel,
		option.None[Height](),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(17), got)

	challengeLevel = uint64(1)
	got, err = provider.computeRequiredNumberOfHashes(
		challengeLevel,
		option.Some(Height(8)),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(9), got)
}

func Test_computeMachineStartIndex(t *testing.T) {
	t.Run("block challenge level", func(t *testing.T) {
		provider := &HistoryCommitmentProvider{
			challengeLeafHeights: []Height{
				32,
				1 << 10,
				1 << 10,
			},
		}
		machineStartIdx, err := provider.computeMachineStartIndex(validatedStartHeights{}, 1)
		require.NoError(t, err)
		require.Equal(t, OpcodeIndex(0), machineStartIdx)
	})
	t.Run("three subchallenge levels", func(t *testing.T) {
		provider := &HistoryCommitmentProvider{
			challengeLeafHeights: []Height{
				32, // block challenge level = 0
				32, // challenge level = 1
				32, // challenge level = 2
				32, // challenge level = 3
			},
		}
		heights := []Height{
			0,
			3,
			4,
		}
		// The first height is ignored, as it is for block challenges and not over machine opcodes.
		//
		//	  3 * (32 * 32)
		//	+ 4 * (32)
		//	+ 5
		//  = 3205
		got, err := provider.computeMachineStartIndex(validatedStartHeights(heights), 5)
		require.NoError(t, err)
		require.Equal(t, OpcodeIndex(3205), got)
	})
	t.Run("four challenge levels", func(t *testing.T) {
		// Take, for example, that we have 4 challenge kinds:
		//
		// block_challenge    => over a range of L2 message hashes
		// megastep_challenge => over ranges of 1048576 (2^20) opcodes at a time.
		// kilostep_challenge => over ranges of 1024 (2^10) opcodes at a time
		// step_challenge     => over a range of individual WASM opcodes
		//
		// We only directly step through WASM machines when in a subchallenge (starting at megastep),
		// so we can ignore block challenges for this calculation.
		//
		// Let's say we want to figure out the machine start opcode index for the following inputs:
		//
		// megastep=4, kilostep=5, step=10
		//
		// We can compute the opcode index using the following algorithm for the example above.
		//
		//	  4 * (1048576)
		//	+ 5 * (1024)
		//	+ 10
		//	= 4,199,434
		provider := &HistoryCommitmentProvider{
			challengeLeafHeights: []Height{
				32, // Block challenge level.
				1 << 10,
				1 << 10,
				1 << 10,
			},
		}
		heights := []Height{
			0,
			4,
			5,
		}
		got, err := provider.computeMachineStartIndex(validatedStartHeights(heights), 10)
		require.NoError(t, err)
		require.Equal(t, OpcodeIndex(4199434), got)
	})
}

func Test_computeStepSize(t *testing.T) {
	provider := &HistoryCommitmentProvider{
		challengeLeafHeights: []Height{
			1,
			2,
			4,
			8,
		},
	}
	t.Run("small step size", func(t *testing.T) {
		challengeLevel := uint64(3)
		stepSize, err := provider.computeStepSize(challengeLevel)
		require.NoError(t, err)
		// The step size for the last challenge level is always 1 opcode at a time.
		require.Equal(t, StepSize(1), stepSize)
	})
	t.Run("product of height constants for next challenge levels", func(t *testing.T) {
		challengeLevel := uint64(0)
		stepSize, err := provider.computeStepSize(challengeLevel)
		require.NoError(t, err)
		// Product of height constants for challenge levels 1, 2, 3.
		require.Equal(t, StepSize(2*4*8), stepSize)
	})

}

func TestValidateOriginHeights(t *testing.T) {
	p := &HistoryCommitmentProvider{
		challengeLeafHeights: []Height{1, 2, 3, 4, 5},
	}

	// Test case: valid upperChallengeOriginHeights
	_, err := p.validateOriginHeights([]Height{1, 2})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test case: too many upperChallengeOriginHeights
	_, err = p.validateOriginHeights([]Height{1, 2, 3, 4, 5, 6})
	if err == nil {
		t.Errorf("Expected an error but got none")
	} else if fmt.Sprintf("%v", err) != "challenge level 6 is out of range for challenge leaf heights [1 2 3 4 5]" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestLeafHeightAtChallengeLevel(t *testing.T) {
	p := &HistoryCommitmentProvider{
		challengeLeafHeights: []Height{1, 2, 3, 4, 5},
	}

	// Test case: valid challengeLevel
	height, err := p.leafHeightAtChallengeLevel(2)
	if err != nil || height != 3 {
		t.Errorf("Expected height 3, got %d with error %v", height, err)
	}

	// Test case: invalid challengeLevel
	_, err = p.leafHeightAtChallengeLevel(10)
	if err == nil {
		t.Errorf("Expected an error but got none")
	} else if fmt.Sprintf("%v", err) != "challenge level 10 is out of range for challenge leaf heights [1 2 3 4 5]" {
		t.Errorf("Unexpected error: %v", err)
	}
}
