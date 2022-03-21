//go:build fullchallengetest
// +build fullchallengetest

//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"testing"
)

func TestFullChallengeAsserterIncorrect(t *testing.T) {
	RunChallengeTest(t, false)
}

func TestFullChallengeAsserterCorrect(t *testing.T) {
	RunChallengeTest(t, true)
}
