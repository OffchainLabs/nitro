// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package util

import (
	"fmt"
	"github.com/ethereum/go-ethereum/core/tracing"
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
	Tracer   *tracing.Hooks
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
		Contract: vm.NewContract(addressHolder{to}, addressHolder{from}, uint256.NewInt(0), 0),
		Depth:    evm.Depth(),
	}
}

func (info *TracingInfo) RecordEmitLog(topics []common.Hash, data []byte) {
	size := uint64(len(data))
	var args []uint256.Int
	args = append(args, *uint256.NewInt(0))    // offset: byte offset in the memory in bytes
	args = append(args, *uint256.NewInt(size)) // size: byte size to copy (length of data)
	for _, topic := range topics {
		args = append(args, HashToUint256(topic)) // topic: 32-byte value. Max topics count is 4
	}
	memory := vm.NewMemory()
	memory.Resize(size)
	memory.Set(0, size, data)
	scope := &vm.ScopeContext{
		Memory:   memory,
		Stack:    TracingStackFromArgs(args...),
		Contract: info.Contract,
	}
	logType := fmt.Sprintf("LOG%d", len(topics))
	info.Tracer.OnOpcode(0, byte(vm.StringToOp(logType)), 0, 0, scope, []byte{}, info.Depth, nil)
}

func (info *TracingInfo) RecordStorageGet(key common.Hash) {
	tracer := info.Tracer
	if info.Scenario == TracingDuringEVM {
		scope := &vm.ScopeContext{
			Memory:   vm.NewMemory(),
			Stack:    TracingStackFromArgs(HashToUint256(key)),
			Contract: info.Contract,
		}
		tracer.OnOpcode(0, byte(vm.SLOAD), 0, 0, scope, []byte{}, info.Depth, nil)
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
		tracer.OnOpcode(0, byte(vm.SSTORE), 0, 0, scope, []byte{}, info.Depth, nil)
	} else {
		tracer.CaptureArbitrumStorageSet(key, value, info.Depth, info.Scenario == TracingBeforeEVM)
	}
}

func (info *TracingInfo) MockCall(input []byte, gas uint64, from, to common.Address, amount *big.Int) {
	tracer := info.Tracer
	depth := info.Depth

	contract := vm.NewContract(addressHolder{to}, addressHolder{from}, uint256.MustFromBig(amount), gas)

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
	tracer.OnOpcode(0, byte(vm.CALL), 0, 0, scope, []byte{}, depth, nil)
	tracer.OnEnter(depth, byte(vm.INVALID), from, to, input, 0, amount)

	retScope := &vm.ScopeContext{
		Memory: vm.NewMemory(),
		Stack: TracingStackFromArgs(
			*uint256.NewInt(0), // return offset
			*uint256.NewInt(0), // return size
		),
		Contract: contract,
	}
	tracer.OnOpcode(0, byte(vm.RETURN), 0, 0, retScope, []byte{}, depth+1, nil)
	tracer.OnExit(depth+1, nil, 0, nil, false)

	popScope := &vm.ScopeContext{
		Memory: vm.NewMemory(),
		Stack: TracingStackFromArgs(
			*uint256.NewInt(1), // CALL result success
		),
		Contract: contract,
	}
	tracer.OnOpcode(0, byte(vm.POP), 0, 0, popScope, []byte{}, depth, nil)
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
