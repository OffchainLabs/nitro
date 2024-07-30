// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build challengetest
// +build challengetest

package arbtest

import (
	"context"
	"testing"

	"github.com/offchainlabs/nitro/util/testhelpers/github"
)

func TestChallengeManagerFullAsserterIncorrect(t *testing.T) {
	t.Parallel()
	defaultWasmRootDir := ""
	RunChallengeTest(t, false, false, makeBatch_MsgsPerBatch+1, defaultWasmRootDir)
}

func TestChallengeManagerFullAsserterIncorrectWithPublishedMachine(t *testing.T) {
	t.Parallel()
	cr, err := github.LatestConsensusRelease(context.Background())
	Require(t, err)
	machPath := populateMachineDir(t, cr)
	RunChallengeTest(t, false, true, makeBatch_MsgsPerBatch+1, machPath)
}

func TestChallengeManagerFullAsserterCorrect(t *testing.T) {
	t.Parallel()
	defaultWasmRootDir := ""
	RunChallengeTest(t, true, false, makeBatch_MsgsPerBatch+2, defaultWasmRootDir)
}

func TestChallengeManagerFullAsserterCorrectWithPublishedMachine(t *testing.T) {
	t.Parallel()
	cr, err := github.LatestConsensusRelease(context.Background())
	Require(t, err)
	machPath := populateMachineDir(t, cr)
	RunChallengeTest(t, true, true, makeBatch_MsgsPerBatch+2, machPath)
}
