// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package server_arb

import "testing"

func TestFinishedMachineProof(t *testing.T) {
	mach := NewFinishedMachine()
	// Just test that this doesn't panic
	mach.ProveNextStep()
	mach.Destroy()
}
