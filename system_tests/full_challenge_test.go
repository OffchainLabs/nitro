// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build challengetest
// +build challengetest

//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"testing"
)

func TestChallengeManagerFullAsserterIncorrect(t *testing.T) {
	t.Parallel()
	RunChallengeTest(t, false, false, MsgPerBatch+1)
}

func TestChallengeManagerFullAsserterCorrect(t *testing.T) {
	t.Parallel()
	RunChallengeTest(t, true, false, MsgPerBatch+2)
}
