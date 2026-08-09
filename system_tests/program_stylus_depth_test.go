// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/vm"
)

func TestProgramMaxStylusCallDepth(t *testing.T) {
	testMaxStylusCallDepth(t, true)
}

func TestProgramMaxStylusCallDepthNative(t *testing.T) {
	testMaxStylusCallDepth(t, false)
}

// testMaxStylusCallDepth asserts that setting
// --execution.stylus-target.max-stylus-call-depth rejects an off-chain call
// chain once the N+1th Stylus frame would enter the VM. The gate is off-chain
// only, so this test drives it through eth_call.
//
// Execution shape (depth N comes from N-1 argsForMulticall wraps around a
// 0-call leaf): eth_call → Stylus (depth 1) → contract_call → Stylus (depth 2)
// → ... → Stylus (depth N).
func testMaxStylusCallDepth(t *testing.T, jit bool) {
	const depthLimit uint16 = 3
	builder, auth, cleanup := setupProgramTest(t, jit, func(b *NodeBuilder) {
		b.execConfig.StylusTarget.MaxStylusCallDepth = depthLimit
	})
	defer cleanup()
	ctx := builder.ctx
	l2client := builder.L2.Client

	callsAddr := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))

	// argsAtStylusDepth(n) produces multicall calldata that, when invoked on
	// callsAddr, causes n Stylus frames to be simultaneously live during
	// execution: the top-level entry is depth 1, each wrap adds one more.
	argsAtStylusDepth := func(n int) []byte {
		args := []byte{0} // leaf: 0 inner calls (lowest frame is just the entry)
		for i := 1; i < n; i++ {
			args = argsForMulticall(vm.CALL, callsAddr, nil, args)
		}
		return args
	}

	// Exactly at the limit must be allowed — the check uses >=, so the Nth
	// frame sees counter = N-1 and passes.
	atLimit := ethereum.CallMsg{To: &callsAddr, Gas: 32_000_000, Data: argsAtStylusDepth(int(depthLimit))}
	if _, err := l2client.CallContract(ctx, atLimit, nil); err != nil {
		Fatal(t, "eth_call at exactly the Stylus depth limit should succeed, got:", err)
	}

	// One over the limit must be rejected. The inner ErrStylusCallDepthExceeded
	// propagates through Stylus contract_call failures as a generic "execution
	// reverted" at eth_call — the specific sentinel is asserted in the unit
	// test for ExecuteWASM (arbos/tx_processor_stylus_depth_test.go). Here we
	// only confirm the depth gate flips the outcome from success (at-limit,
	// above) to revert (over-limit).
	overLimit := ethereum.CallMsg{To: &callsAddr, Gas: 32_000_000, Data: argsAtStylusDepth(int(depthLimit) + 1)}
	_, err := l2client.CallContract(ctx, overLimit, nil)
	if err == nil || !strings.Contains(err.Error(), "execution reverted") {
		Fatal(t, "eth_call one over the Stylus depth limit should revert, got:", err)
	}
}
