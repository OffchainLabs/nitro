// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"testing"

	"github.com/offchainlabs/nitro/util/arbmath"
)

const (
	testSamplingMaxTxDataSize = 1000
	testSamplingSmallTxSize   = 100
	testSamplingBigTxSize     = 900
)

func fixedBips(value arbmath.UBips) func() arbmath.UBips {
	return func() arbmath.UBips { return value }
}

func TestSizeSamplingExponentFollowsStreakAndCap(t *testing.T) {
	tests := []struct {
		name         string
		streak       uint
		maxExponent  uint
		graceBlocks  uint
		wantExponent uint
	}{
		{name: "no streak disables sampling", streak: 0, maxExponent: 10, graceBlocks: 0, wantExponent: 0},
		{name: "max exponent 0 disables sampling", streak: 5, maxExponent: 0, graceBlocks: 0, wantExponent: 0},
		{name: "streak below the cap sets k=streak", streak: 1, maxExponent: 10, graceBlocks: 0, wantExponent: 1},
		{name: "streak above the cap uses the cap", streak: 5, maxExponent: 2, graceBlocks: 0, wantExponent: 2},
		{name: "streak within the grace blocks disables sampling", streak: 1, maxExponent: 10, graceBlocks: 1, wantExponent: 0},
		{name: "grace blocks are subtracted from k", streak: 3, maxExponent: 10, graceBlocks: 1, wantExponent: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := FIFOConfig{SizeSamplingMaxExponent: tt.maxExponent, SizeSamplingGraceBlocks: tt.graceBlocks}
			s := newSizeSampling(config, testSamplingMaxTxDataSize, tt.streak)
			if s.exponent != tt.wantExponent {
				t.Fatalf("exponent = %d, want %d", s.exponent, tt.wantExponent)
			}
		})
	}
}

func TestSizeSamplingAdmits(t *testing.T) {
	const minTxSize = 100
	tests := []struct {
		name     string
		txSize   int
		exponent uint
		random   arbmath.UBips
		want     bool
	}{
		{name: "empty tx is always admitted", txSize: 0, exponent: 3, random: 9999, want: true},
		{name: "exponent 0 admits everything", txSize: 999, exponent: 0, random: 9999, want: true},
		{name: "tx at the limit is never admitted", txSize: testSamplingMaxTxDataSize, exponent: 1, random: 0, want: false},
		{name: "tx over the limit is never admitted", txSize: testSamplingMaxTxDataSize + 1, exponent: 1, random: 0, want: false},
		{name: "half-size tx at k=1 admits below 5000 bips", txSize: 500, exponent: 1, random: 4999, want: true},
		{name: "half-size tx at k=1 rejects at 5000 bips", txSize: 500, exponent: 1, random: 5000, want: false},
		{name: "half-size tx at k=2 admits below 2500 bips", txSize: 500, exponent: 2, random: 2499, want: true},
		{name: "half-size tx at k=2 rejects at 2500 bips", txSize: 500, exponent: 2, random: 2500, want: false},
		{name: "tx at the min size is always admitted", txSize: minTxSize, exponent: 10, random: 9999, want: true},
		{name: "tx just above the min size is sampled", txSize: minTxSize + 1, exponent: 10, random: 9999, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := sizeSampling{
				exponent:      tt.exponent,
				minTxSize:     minTxSize,
				maxTxDataSize: testSamplingMaxTxDataSize,
				randomBips:    fixedBips(tt.random),
			}
			item, _ := makeTestQueueItem(t, 0, testBaseFee)
			item.txSize = tt.txSize
			if got := s.admits(item); got != tt.want {
				t.Fatalf("admits(size %d, k %d, random %v) = %v, want %v", tt.txSize, tt.exponent, tt.random, got, tt.want)
			}
		})
	}
}
