// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

// race detection makes things slow and miss timeouts
//go:build fullchallengetest
// +build fullchallengetest

package arbtest

import "testing"

func TestStakersFaultyHonestActive(t *testing.T) {
	stakerTestImpl(t, true, false)
}

func TestStakersFaultyHonestInactive(t *testing.T) {
	stakerTestImpl(t, true, true)
}
