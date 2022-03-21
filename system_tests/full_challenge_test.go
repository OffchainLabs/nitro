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
	runChallengeTest(t, false)
}

func TestFullChallengeAsserterCorrect(t *testing.T) {
	runChallengeTest(t, true)
}
