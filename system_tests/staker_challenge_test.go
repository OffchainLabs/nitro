// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

// race detection makes things slow and miss timeouts
//go:build challengetest
// +build challengetest

package arbtest

import "testing"

func TestChallengeStakersFaultyHonestActive(t *testing.T) {
	t.Skip("TODO: staker challenge tests are broken due to seq inbox changes")
	stakerTestImpl(t, true, false)
}

func TestChallengeStakersFaultyHonestInactive(t *testing.T) {
	t.Skip("TODO: staker challenge tests are broken due to seq inbox changes")
	stakerTestImpl(t, true, true)
}
