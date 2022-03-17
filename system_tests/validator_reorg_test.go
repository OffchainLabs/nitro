//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

//go:build validatorreorgtest
// +build validatorreorgtest

package arbtest

import "testing"

func TestBlockValidatorReorg(t *testing.T) {
	testSequencerInboxReaderImpl(t, true)
}
