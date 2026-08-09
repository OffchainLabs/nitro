// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbos

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
)

// These tests cover the reject branch of the depth gate in
// TxProcessor.ExecuteWASM (counter >= limit → ErrStylusCallDepthExceeded
// without reaching CallProgram) plus supporting invariants: zero-limit
// disables the gate, on-chain run contexts are exempt (so the limit cannot
// diverge consensus), and the defer decrement rebalances the counter even
// when CallProgram returns an error or panics. The real-VM allow path is
// covered by the integration test in system_tests/.

func newTxProcessorWithDepthLimit(t *testing.T, limit uint16, runCtx *core.MessageRunContext) (*TxProcessor, *vm.EVM) {
	t.Helper()
	chainConfig := chaininfo.ArbitrumDevTestChainConfig()
	_, statedb := arbosState.NewArbosMemoryBackedArbOSStateWithConfig(chainConfig)
	statedb.Database().SetArbNodeConfig(&programs.ArbNodeConfig{MaxStylusCallDepth: limit})
	evm := vm.NewEVM(vm.BlockContext{}, statedb, chainConfig, vm.Config{})
	msg := &core.Message{TxRunContext: runCtx}
	return NewTxProcessor(evm, msg), evm
}

func dummyScope() *vm.ScopeContext {
	contract := vm.NewContract(common.Address{}, common.Address{1}, new(uint256.Int), 1_000_000, nil)
	return &vm.ScopeContext{Contract: contract}
}

func TestExecuteWASM_AtDepthLimit_OffChain_Rejects(t *testing.T) {
	// Off-chain modes with limit 2 and two Stylus frames already on the stack:
	// the next entry must reject via the depth gate, never reaching CallProgram.
	// Both ethcall and gas-estimation are !IsExecutedOnChain, so both should
	// reject symmetrically — pinning the "all off-chain modes" contract.
	for _, tc := range []struct {
		name   string
		runCtx *core.MessageRunContext
	}{
		{"ethcall", core.NewMessageEthcallContext()},
		{"gas-estimation", core.NewMessageGasEstimationContext()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, evm := newTxProcessorWithDepthLimit(t, 2, tc.runCtx)
			tx.stylusCallDepth = 2

			_, err := tx.ExecuteWASM(dummyScope(), nil, evm)
			require.ErrorIs(t, err, programs.ErrStylusCallDepthExceeded,
				"at-limit off-chain entry must reject with ErrStylusCallDepthExceeded")
			require.Equal(t, uint16(2), tx.stylusCallDepth,
				"rejected entry must not mutate the depth counter")
		})
	}
}

func TestExecuteWASM_OverDepthLimit_OffChain_Rejects(t *testing.T) {
	// Check uses >=, so any counter at-or-beyond the limit rejects off-chain.
	tx, evm := newTxProcessorWithDepthLimit(t, 2, core.NewMessageEthcallContext())
	tx.stylusCallDepth = 3

	_, err := tx.ExecuteWASM(dummyScope(), nil, evm)
	require.ErrorIs(t, err, programs.ErrStylusCallDepthExceeded)
	require.Equal(t, uint16(3), tx.stylusCallDepth)
}

func TestExecuteWASM_OnChain_ExemptEvenOverLimit(t *testing.T) {
	// On-chain run contexts (replay, commit, recording, sequencing) must be
	// exempt from the gate even when the counter is over the configured limit.
	// This is the local invariant that underwrites the cross-validator claim
	// that the limit is not consensus-relevant.
	//
	// CallProgram will fail on the dummy scope/state with an unrelated error
	// (or panic). We assert (a) the depth sentinel never surfaces, (b) some
	// error did surface — confirming CallProgram was actually reached rather
	// than an unrelated short-circuit, and (c) the counter was rebalanced by
	// the deferred decrement.
	tx, evm := newTxProcessorWithDepthLimit(t, 2, core.NewMessageReplayContext())
	tx.stylusCallDepth = 100

	var err error
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		_, err = tx.ExecuteWASM(dummyScope(), nil, evm)
	}()
	require.NotErrorIs(t, err, programs.ErrStylusCallDepthExceeded,
		"on-chain run context must be exempt from the depth gate")
	require.True(t, panicked || err != nil,
		"dummy scope should have reached CallProgram and failed — else the test passes for the wrong reason")
	require.Equal(t, uint16(100), tx.stylusCallDepth,
		"counter must be rebalanced even on the on-chain exempt path")
}

func TestExecuteWASM_ZeroLimit_DoesNotReject(t *testing.T) {
	// Limit 0 disables the gate regardless of run context. Even with a high
	// counter, ExecuteWASM must fall through to CallProgram and never surface
	// the depth sentinel.
	tx, evm := newTxProcessorWithDepthLimit(t, 0, core.NewMessageEthcallContext())
	tx.stylusCallDepth = 100

	var err error
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		_, err = tx.ExecuteWASM(dummyScope(), nil, evm)
	}()
	require.NotErrorIs(t, err, programs.ErrStylusCallDepthExceeded,
		"zero limit must disable the depth gate regardless of the counter value")
	require.True(t, panicked || err != nil,
		"dummy scope should have reached CallProgram and failed — else the test passes for the wrong reason")
	require.Equal(t, uint16(100), tx.stylusCallDepth,
		"counter must be rebalanced after CallProgram returns or panics")
}

func TestExecuteWASM_AllowPath_CounterRebalancedOnFailure(t *testing.T) {
	// The allow branch enters CallProgram, which on this dummy scope either
	// returns an error or panics. Either way the deferred decrement must
	// restore the counter to its pre-entry value — otherwise a real panic
	// during VM execution would permanently leak depth and eventually reject
	// legitimate calls.
	tx, evm := newTxProcessorWithDepthLimit(t, 10, core.NewMessageEthcallContext())
	tx.stylusCallDepth = 0

	var err error
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		_, err = tx.ExecuteWASM(dummyScope(), nil, evm)
	}()
	require.True(t, panicked || err != nil,
		"dummy scope should have reached CallProgram and failed — else the test passes for the wrong reason")
	require.Equal(t, uint16(0), tx.stylusCallDepth,
		"defer must rebalance the counter when CallProgram returns or panics")
}
