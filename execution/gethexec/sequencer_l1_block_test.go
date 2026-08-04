// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func TestNextSequencerParentChainBlockNumber(t *testing.T) {
	tests := []struct {
		name     string
		format   uint64
		previous uint64
		observed uint64
		expected uint64
	}{
		{
			name:     "legacy header is unchanged",
			observed: 102,
			expected: 102,
		},
		{
			name:     "same parent chain block",
			format:   params.ArbosVersion_51,
			previous: 100,
			observed: 100,
			expected: 100,
		},
		{
			name:     "next parent chain block",
			format:   params.ArbosVersion_51,
			previous: 100,
			observed: 101,
			expected: 101,
		},
		{
			name:     "one skipped parent chain block",
			format:   params.ArbosVersion_51,
			previous: 100,
			observed: 102,
			expected: 101,
		},
		{
			name:     "multiple skipped parent chain blocks",
			format:   params.ArbosVersion_51,
			previous: 100,
			observed: 500,
			expected: 101,
		},
		{
			name:     "lower observation is unchanged",
			format:   params.ArbosVersion_51,
			previous: 100,
			observed: 99,
			expected: 99,
		},
		{
			name:     "maximum block number does not overflow",
			format:   params.ArbosVersion_51,
			previous: math.MaxUint64 - 1,
			observed: math.MaxUint64,
			expected: math.MaxUint64,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lastL2Block := arbosL2Header(test.previous, test.format)
			actual := nextSequencerParentChainBlockNumber(test.observed, lastL2Block)
			if actual != test.expected {
				t.Fatalf("expected parent chain block %d, got %d", test.expected, actual)
			}
		})
	}
}

func TestNextSequencerParentChainBlockNumberCatchesUpInOrder(t *testing.T) {
	const observed = uint64(105)
	lastL2Block := arbosL2Header(100, params.ArbosVersion_51)

	for expected := uint64(101); expected <= observed; expected++ {
		actual := nextSequencerParentChainBlockNumber(observed, lastL2Block)
		if actual != expected {
			t.Fatalf("expected parent chain block %d, got %d", expected, actual)
		}
		lastL2Block = arbosL2Header(actual, params.ArbosVersion_51)
	}
}

func arbosL2Header(l1BlockNumber uint64, arbosFormatVersion uint64) *types.Header {
	header := &types.Header{
		BaseFee:    big.NewInt(1),
		Difficulty: big.NewInt(1),
	}
	types.HeaderInfo{
		L1BlockNumber:      l1BlockNumber,
		ArbOSFormatVersion: arbosFormatVersion,
	}.UpdateHeaderWithInfo(header)
	return header
}
