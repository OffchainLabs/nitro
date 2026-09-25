// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"strings"
	"testing"

	"github.com/offchainlabs/nitro/arbos/programs"
)

// TestStylusActivationRefusedOverRpc pins both halves of the node-level switch:
// a default node refuses the activation that gas estimation drives, while the
// transaction that activates onchain is untouched.
func TestStylusActivationRefusedOverRpc(t *testing.T) {
	env := setupActivationGasTest(t, func(b *NodeBuilder) {
		b.execConfig.StylusTarget.AllowOffchainActivation = false
	})
	defer env.cleanup()

	program := deployUnactivatedWasm(t, env, rustFile("keccak"))

	// A zero gas limit makes bind estimate first, and estimation is not
	// executed onchain.
	env.auth.GasLimit = 0
	env.auth.Value = oneEth
	_, err := env.arbWasm.ActivateProgram(&env.auth, program)
	if err == nil {
		Fatal(t, "estimated activation should have been refused")
	}
	if !strings.Contains(err.Error(), programs.ErrOffchainActivationNotAllowed.Error()) {
		Fatal(t, "estimated activation failed with", err, "want", programs.ErrOffchainActivationNotAllowed)
	}

	// The same activation still succeeds as a transaction.
	env.auth.GasLimit = 32_000_000
	env.ensure(env.arbWasm.ActivateProgram(&env.auth, program))
}

// TestStylusActivationAllowedOverRpcWhenConfigured is the opt-in half: the node
// serves the estimate, so the ordinary deploy flow works again.
func TestStylusActivationAllowedOverRpcWhenConfigured(t *testing.T) {
	env := setupActivationGasTest(t, func(b *NodeBuilder) {
		b.execConfig.StylusTarget.AllowOffchainActivation = true
	})
	defer env.cleanup()

	program := deployUnactivatedWasm(t, env, rustFile("keccak"))

	env.auth.GasLimit = 0
	env.auth.Value = oneEth
	env.ensure(env.arbWasm.ActivateProgram(&env.auth, program))
}
