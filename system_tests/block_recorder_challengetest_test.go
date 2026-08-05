// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build challengetest && !race

package arbtest

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/execution/gethexec"
)

func challengeBlockRecorderTestCases(t testing.TB) []blockRecorderTestCase {
	cases := blockRecorderTestCases()
	filtered := make([]blockRecorderTestCase, 0, len(cases))
	for _, tc := range cases {
		if tc.stateScheme == rawdb.HashScheme {
			filtered = append(filtered, tc)
		}
	}
	if len(filtered) == 0 {
		t.Skip("BOLD challenge tests require hash state scheme")
	}
	return filtered
}

func withBlockRecorderTestCase(tc blockRecorderTestCase) func(*gethexec.Config) {
	return func(config *gethexec.Config) {
		config.RecordingDatabase.Mode = tc.recorderMode
		config.Caching.StateScheme = tc.stateScheme
	}
}
