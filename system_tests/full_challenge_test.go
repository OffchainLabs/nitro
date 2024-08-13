// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build challengetest
// +build challengetest

package arbtest

import "testing"

func TestChallengeManagerFullAsserterIncorrect(t *testing.T) {
	t.Parallel()
	RunChallengeTest(t, false, false, makeBatch_MsgsPerBatch+1)
}

func TestChallengeManagerFullAsserterCorrect(t *testing.T) {
	t.Parallel()
	RunChallengeTest(t, true, false, makeBatch_MsgsPerBatch+2)
}
