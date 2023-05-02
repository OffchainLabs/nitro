// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build challengetest
// +build challengetest

package arbtest

import "testing"

func TestChallengeManagerFullAsserterIncorrect(t *testing.T) {
	RunChallengeTest(t, false)
}

func TestChallengeManagerFullAsserterCorrect(t *testing.T) {
	RunChallengeTest(t, true)
}
