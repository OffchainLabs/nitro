// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

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
