// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package arbtest

import "testing"

func TestMockChallengeManagerAsserterIncorrect(t *testing.T) {
	t.Parallel()
	for i := int64(1); i <= MsgPerBatch*3; i++ {
		RunChallengeTest(t, false, true, i)
	}
}

func TestMockChallengeManagerAsserterCorrect(t *testing.T) {
	t.Parallel()
	for i := int64(1); i <= MsgPerBatch*3; i++ {
		RunChallengeTest(t, true, true, i)
	}
}
