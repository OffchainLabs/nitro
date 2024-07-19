// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build stylustest && !race
// +build stylustest,!race

package arbtest

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
)

func TestProgramArbitratorKeccak(t *testing.T) {
	keccakTest(t, false)
}

func TestProgramArbitratorErrors(t *testing.T) {
	errorTest(t, false)
}

func TestProgramArbitratorStorage(t *testing.T) {
	storageTest(t, false)
}

func TestProgramArbitratorTransientStorage(t *testing.T) {
	transientStorageTest(t, false)
}

func TestProgramArbitratorMath(t *testing.T) {
	fastMathTest(t, false)
}

func TestProgramArbitratorCalls(t *testing.T) {
	testCalls(t, false)
}

func TestProgramArbitratorReturnData(t *testing.T) {
	testReturnData(t, false)
}

func TestProgramArbitratorLogs(t *testing.T) {
	testLogs(t, false, false)
}

func TestProgramArbitratorCreate(t *testing.T) {
	testCreate(t, false)
}

//func TestProgramArbitratorEvmData(t *testing.T) {
//	testEvmData(t, false)
//}

func TestProgramArbitratorMemory(t *testing.T) {
	testMemory(t, false)
}

func TestProgramArbitratorActivateTwice(t *testing.T) {
	t.Parallel()
	testActivateTwice(t, false)
}

func TestProgramArbitratorActivateFails(t *testing.T) {
	t.Parallel()
	testActivateFails(t, false)
}

func TestProgramArbitratorEarlyExit(t *testing.T) {
	testEarlyExit(t, false)
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
	testProgramResursiveCalls(t, fullRecurseTest(), true)
}

func TestProgramLongArbitratorCall(t *testing.T) {
	testProgramResursiveCalls(t, fullRecurseTest(), false)
}

func TestProgramArbitratorStylusUpgrade(t *testing.T) {
	testStylusUpgrade(t, false)
}
