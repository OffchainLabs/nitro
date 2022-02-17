//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

// race detection makes things slow and miss timeouts
//go:build fullchallengetest
// +build fullchallengetest

package arbtest

import "testing"

func TestStakersMakeNodesFaulty(t *testing.T) {
	stakerTestImpl(t, true, false)
}

func TestStakersStakeLatestFaulty(t *testing.T) {
	stakerTestImpl(t, false, true)
}
