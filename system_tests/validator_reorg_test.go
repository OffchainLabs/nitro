// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

//go:build validatorreorgtest
// +build validatorreorgtest

package arbtest

import "testing"

func TestBlockValidatorReorg(t *testing.T) {
	testSequencerInboxReaderImpl(t, true)
}
