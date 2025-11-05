// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

//go:build !debugblock

package arbtest

import (
	"testing"
)

func TestDebugBlockInjectionStub(t *testing.T) {
	testDebugBlockInjection(t, false)
}
