// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package util

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

type TracingScenario uint64

const (
	TracingBeforeEVM TracingScenario = iota
	TracingDuringEVM
	TracingAfterEVM
)

type TracingInfo struct {
	Tracer   vm.EVMLogger
	Scenario TracingScenario
	Contract *vm.Contract
	Depth    int
}

// holds an address to satisfy core/vm's ContractRef() interface
type addressHolder struct {
	addr common.Address
}

func (a addressHolder) Address() common.Address {
	return a.addr
}

func NewTracingInfo(evm *vm.EVM, from, to common.Address, scenario TracingScenario) *TracingInfo {
	if evm.Config.Tracer == nil {
		return nil
	}
	return &TracingInfo{
		Tracer:   evm.Config.Tracer,
		Scenario: scenario,
		Contract: vm.NewContract(addressHolder{to}, addressHolder{from}, big.NewInt(0), 0),
		Depth:    evm.Depth(),
	}
}

func (info *TracingInfo) RecordStorageGet(key common.Hash) {
	tracer := info.Tracer
	if info.Scenario == TracingDuringEVM {
		scope := &vm.ScopeContext{
			Memory:   vm.NewMemory(),
			Stack:    TracingStackFromArgs(HashToUint256(key)),
			Contract: info.Contract,
		}
		tracer.CaptureState(0, vm.SLOAD, 0, 0, scope, []byte{}, info.Depth, nil)
	} else {
		tracer.CaptureArbitrumStorageGet(key, info.Depth, info.Scenario == TracingBeforeEVM)
	}
}

func (info *TracingInfo) RecordStorageSet(key, value common.Hash) {
	tracer := info.Tracer
	if info.Scenario == TracingDuringEVM {
		scope := &vm.ScopeContext{
			Memory:   vm.NewMemory(),
			Stack:    TracingStackFromArgs(HashToUint256(key), HashToUint256(value)),
			Contract: info.Contract,
		}
		tracer.CaptureState(0, vm.SSTORE, 0, 0, scope, []byte{}, info.Depth, nil)
	} else {
		tracer.CaptureArbitrumStorageSet(key, value, info.Depth, info.Scenario == TracingBeforeEVM)
	}
}

func (info *TracingInfo) MockCall(input []byte, gas uint64, from, to common.Address, amount *big.Int) {
	tracer := info.Tracer
	depth := info.Depth

	contract := vm.NewContract(addressHolder{to}, addressHolder{from}, amount, gas)

	scope := &vm.ScopeContext{
		Memory: TracingMemoryFromBytes(input),
		Stack: TracingStackFromArgs(
			*uint256.NewInt(gas),                        // gas
			*uint256.NewInt(0).SetBytes(to.Bytes()),     // to address
			*uint256.NewInt(0).SetBytes(amount.Bytes()), // call value
			*uint256.NewInt(0),                          // memory offset
			*uint256.NewInt(uint64(len(input))),         // memory length
			*uint256.NewInt(0),                          // return offset
			*uint256.NewInt(0),                          // return size
		),
		Contract: contract,
	}
	tracer.CaptureState(0, vm.CALL, 0, 0, scope, []byte{}, depth, nil)
	tracer.CaptureEnter(vm.INVALID, from, to, input, 0, amount)

	retScope := &vm.ScopeContext{
		Memory: vm.NewMemory(),
		Stack: TracingStackFromArgs(
			*uint256.NewInt(0), // return offset
			*uint256.NewInt(0), // return size
		),
		Contract: contract,
	}
	tracer.CaptureState(0, vm.RETURN, 0, 0, retScope, []byte{}, depth+1, nil)
	tracer.CaptureExit(nil, 0, nil)

	popScope := &vm.ScopeContext{
		Memory: vm.NewMemory(),
		Stack: TracingStackFromArgs(
			*uint256.NewInt(1), // CALL result success
		),
		Contract: contract,
	}
	tracer.CaptureState(0, vm.POP, 0, 0, popScope, []byte{}, depth, nil)
}

func HashToUint256(hash common.Hash) uint256.Int {
	value := uint256.Int{}
	value.SetBytes(hash.Bytes())
	return value
}

// TracingMemoryFromBytes creates an EVM Memory consisting of the bytes provided
func TracingMemoryFromBytes(input []byte) *vm.Memory {
	memory := vm.NewMemory()
	inputLen := uint64(len(input))
	memory.Resize(inputLen)
	memory.Set(0, inputLen, input)
	return memory
}

// TracingStackFromArgs creates an EVM Stack with the given arguments in canonical order
func TracingStackFromArgs(args ...uint256.Int) *vm.Stack {
	stack := &vm.Stack{}
	for flip := 0; flip < len(args)/2; flip++ { // reverse the order
		flop := len(args) - flip - 1
		args[flip], args[flop] = args[flop], args[flip]
	}
	stack.SetData(args)
	return stack
}
