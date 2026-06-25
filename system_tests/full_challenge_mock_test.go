// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// race detection makes things slow and miss timeouts
//go:build !race

package arbtest

import "testing"

func TestMockChallengeManagerAsserterIncorrect(t *testing.T) {
	defaultWasmRootDir := ""
	for i := int64(1); i <= makeBatch_MsgsPerBatch*3; i++ {
		RunChallengeTest(t, false, true, i, defaultWasmRootDir)
	}
}

func TestMockChallengeManagerAsserterCorrect(t *testing.T) {
	defaultWasmRootDir := ""
	for i := int64(1); i <= makeBatch_MsgsPerBatch*3; i++ {
		RunChallengeTest(t, true, true, i, defaultWasmRootDir)
	}
}
