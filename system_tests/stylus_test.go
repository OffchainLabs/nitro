// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build stylustest && !race

package arbtest

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
)

func TestProgramArbitratorKeccak(t *testing.T) {
	keccakTest(t, false)
}

func TestProgramArbitrator(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, false)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	// Deploy shared WASMs once
	multicallAddr := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))
	storageAddr := deployWasm(t, ctx, auth, l2client, rustFile("storage"))
	keccakAddr := deployWasm(t, ctx, auth, l2client, rustFile("keccak"))
	logAddr := deployWasm(t, ctx, auth, l2client, rustFile("log"))

	t.Run("Errors", func(t *testing.T) { errorTest(t, builder, auth, multicallAddr) })
	t.Run("Storage", func(t *testing.T) { storageTest(t, builder, auth, storageAddr) })
	t.Run("TransientStorage", func(t *testing.T) { transientStorageTest(t, builder, auth, storageAddr, multicallAddr) })
	t.Run("Math", func(t *testing.T) { fastMathTest(t, builder, auth) })
	t.Run("Calls", func(t *testing.T) { testCalls(t, builder, auth, multicallAddr, storageAddr, keccakAddr) })
	t.Run("ReturnData", func(t *testing.T) { testReturnData(t, builder, auth) })
	t.Run("Logs", func(t *testing.T) { testLogs(t, builder, auth, logAddr, multicallAddr, false) })
	t.Run("ActivateFails", func(t *testing.T) { testActivateFails(t, builder, auth) })
	t.Run("EarlyExit", func(t *testing.T) { testEarlyExit(t, builder, auth) })

	validateBlocks(t, 1, false, builder)
}

func TestProgramArbitratorCreate(t *testing.T) {
	testCreate(t, false)
}

func TestProgramArbitratorEvmData(t *testing.T) {
	testEvmData(t, false)
}

func TestProgramArbitratorMemory(t *testing.T) {
	testMemory(t, false)
}

func TestProgramArbitratorActivateTwice(t *testing.T) {
	testActivateTwice(t, false)
}

func fullRecurseTest() [][]multiCallRecurse {
	result := make([][]multiCallRecurse, 0)
	for _, op0 := range []vm.OpCode{vm.SSTORE, vm.SLOAD} {
		for _, contract0 := range []string{"multicall-rust", "multicall-evm"} {
			for _, op1 := range []vm.OpCode{vm.CALL, vm.STATICCALL, vm.DELEGATECALL} {
				for _, contract1 := range []string{"multicall-rust", "multicall-rust-b", "multicall-evm"} {
					for _, op2 := range []vm.OpCode{vm.CALL, vm.STATICCALL, vm.DELEGATECALL} {
						for _, contract2 := range []string{"multicall-rust", "multicall-rust-b", "multicall-evm"} {
							for _, op3 := range []vm.OpCode{vm.CALL, vm.STATICCALL, vm.DELEGATECALL} {
								for _, contract3 := range []string{"multicall-rust", "multicall-rust-b", "multicall-evm"} {
									recurse := make([]multiCallRecurse, 4)
									recurse[0].opcode = op0
									recurse[0].Name = contract0
									recurse[1].opcode = op1
									recurse[1].Name = contract1
									recurse[2].opcode = op2
									recurse[2].Name = contract2
									recurse[3].opcode = op3
									recurse[3].Name = contract3
									result = append(result, recurse)
								}
							}
						}
					}
				}
			}
		}
	}
	return result
}

func TestProgramLongCall(t *testing.T) {
	testProgramRecursiveCalls(t, fullRecurseTest(), true)
}

func TestProgramLongArbitratorCall(t *testing.T) {
	testProgramRecursiveCalls(t, fullRecurseTest(), false)
}

func TestProgramArbitratorStylusUpgrade(t *testing.T) {
	testStylusUpgrade(t, false)
}
