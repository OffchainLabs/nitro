// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package util

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
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
		Contract: vm.NewContract(addressHolder{to}, addressHolder{from}, uint256.NewInt(0), 0),
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

func (info *TracingInfo) CaptureEVMTraceForHostio(name string, args, outs []byte, startInk, endInk uint64, scope *vm.ScopeContext, depth int) {
	intToBytes := func(v int) []byte {
		return binary.BigEndian.AppendUint64(nil, uint64(v))
	}

	checkArgs := func(want int) bool {
		if len(args) < want {
			log.Warn("tracing: missing arguments bytes for hostio", "name", name, "want", want, "got", len(args))
			return false
		}
		return true
	}

	checkOuts := func(want int) bool {
		if len(outs) < want {
			log.Warn("tracing: missing outputs bytes for hostio", "name", name, "want", want, "got", len(args))
			return false
		}
		return true
	}

	firstOpcode := true
	capture := func(op vm.OpCode, memory []byte, stackValues ...[]byte) {
		const inkToGas = 10000
		var gas, cost uint64
		if firstOpcode {
			gas = startInk / inkToGas
			cost = (startInk - endInk) / inkToGas
			firstOpcode = false
		} else {
			// When capturing multiple opcodes, usually the first one is the relevant
			// action and the following ones just pop the result values from the stack.
			gas = endInk / inkToGas
			cost = 0
		}

		stack := []uint256.Int{}
		for _, value := range stackValues {
			stack = append(stack, *uint256.NewInt(0).SetBytes(value))
		}
		scope := &vm.ScopeContext{
			Memory:   TracingMemoryFromBytes(memory),
			Stack:    TracingStackFromArgs(stack...),
			Contract: scope.Contract,
		}

		info.Tracer.CaptureState(0, op, gas, cost, scope, []byte{}, depth, nil)
	}

	switch name {
	case "read_args":
		destOffset := []byte(nil)
		offset := []byte(nil)
		size := intToBytes(len(outs))
		capture(vm.CALLDATACOPY, outs, destOffset, offset, size)

	case "exit_early":
		if !checkArgs(4) {
			return
		}
		status := binary.BigEndian.Uint32(args[:4])
		var opcode vm.OpCode
		if status == 0 {
			opcode = vm.RETURN
		} else {
			opcode = vm.REVERT
		}
		offset := []byte(nil)
		size := []byte(nil)
		capture(opcode, nil, offset, size)

	case "storage_load_bytes32":
		if !checkArgs(32) || !checkOuts(32) {
			return
		}
		key := args[:32]
		value := outs[:32]
		capture(vm.SLOAD, nil, key)
		capture(vm.POP, nil, value)

	case "storage_cache_bytes32":
		if !checkArgs(32 + 32) {
			return
		}
		key := args[:32]
		value := args[32:64]
		capture(vm.SSTORE, nil, key, value)

	case "storage_flush_cache":
		// SSTORE is handled above by storage_cache_bytes32

	case "transient_load_bytes32":
		if !checkArgs(32) || !checkOuts(32) {
			return
		}
		key := args[:32]
		value := outs[:32]
		capture(vm.TLOAD, nil, key)
		capture(vm.POP, nil, value)

	case "transient_store_bytes32":
		if !checkArgs(32 + 32) {
			return
		}
		key := args[:32]
		value := args[32:64]
		capture(vm.TSTORE, nil, key, value)

	case "call_contract":
		if !checkArgs(20+8+32) || !checkOuts(4+1) {
			return
		}
		address := args[:20]
		gas := args[20:28]
		value := args[28:60]
		callArgs := args[60:]
		argsOffset := []byte(nil)
		argsSize := intToBytes(len(callArgs))
		retOffset := []byte(nil)
		retSize := []byte(nil)
		status := outs[4:5]
		capture(vm.CALL, callArgs, gas, address, value, argsOffset, argsSize, retOffset, retSize)
		capture(vm.POP, callArgs, status)

	case "delegate_call_contract", "static_call_contract":
		if !checkArgs(20+8) || !checkOuts(4+1) {
			return
		}
		address := args[:20]
		gas := args[20:28]
		callArgs := args[28:]
		argsOffset := []byte(nil)
		argsSize := intToBytes(len(callArgs))
		retOffset := []byte(nil)
		retSize := []byte(nil)
		status := outs[4:5]
		var opcode vm.OpCode
		if name == "delegate_call_contract" {
			opcode = vm.DELEGATECALL
		} else {
			opcode = vm.STATICCALL
		}
		capture(opcode, callArgs, gas, address, argsOffset, argsSize, retOffset, retSize)
		capture(vm.POP, callArgs, status)

	case "create1":
		if !checkArgs(32) || !checkOuts(20) {
			return
		}
		value := args[:32]
		code := args[32:]
		offset := []byte(nil)
		size := intToBytes(len(code))
		address := outs[:20]
		capture(vm.CREATE, code, value, offset, size)
		capture(vm.POP, code, address)

	case "create2":
		if !checkArgs(32+32) || !checkOuts(20) {
			return
		}
		value := args[:32]
		salt := args[32:64]
		code := args[64:]
		offset := []byte(nil)
		size := intToBytes(len(code))
		address := outs[:20]
		capture(vm.CREATE2, code, value, offset, size, salt)
		capture(vm.POP, code, address)

	case "read_return_data":
		if !checkArgs(8) {
			return
		}
		destOffset := []byte(nil)
		offset := args[:4]
		size := args[4:8]
		capture(vm.RETURNDATACOPY, outs, destOffset, offset, size)

	case "return_data_size":
		if !checkOuts(4) {
			return
		}
		size := outs[:4]
		capture(vm.RETURNDATASIZE, nil)
		capture(vm.POP, nil, size)

	case "emit_log":
		if !checkArgs(4) {
			return
		}
		numTopics := int(binary.BigEndian.Uint32(args[:4]))
		dataOffset := 4 + 32*numTopics
		if !checkArgs(dataOffset) {
			return
		}
		data := args[dataOffset:]
		offset := []byte(nil)
		size := intToBytes(len(data))
		opcode := vm.LOG0 + vm.OpCode(numTopics)
		stack := [][]byte{offset, size}
		for i := 0; i < numTopics; i++ {
			topic := args[4+32*i : 4+32*(i+1)]
			stack = append(stack, topic)
		}
		capture(opcode, data, stack...)

	case "account_balance":
		if !checkArgs(20) || !checkOuts(32) {
			return
		}
		address := args[:20]
		balance := outs[:32]
		capture(vm.BALANCE, nil, address)
		capture(vm.POP, nil, balance)

	case "account_code":
		if !checkArgs(20 + 4 + 4) {
			return
		}
		address := args[:20]
		destOffset := []byte(nil)
		offset := args[20:24]
		size := args[24:28]
		capture(vm.EXTCODECOPY, nil, address, destOffset, offset, size)

	case "account_code_size":
		if !checkArgs(20) || !checkOuts(4) {
			return
		}
		address := args[:20]
		size := outs[:4]
		capture(vm.EXTCODESIZE, nil, address)
		capture(vm.POP, nil, size)

	case "account_codehash":
		if !checkArgs(20) || !checkOuts(32) {
			return
		}
		address := args[:20]
		hash := outs[:32]
		capture(vm.EXTCODEHASH, nil, address)
		capture(vm.POP, nil, hash)

	case "block_basefee":
		if !checkOuts(32) {
			return
		}
		baseFee := outs[:32]
		capture(vm.BASEFEE, nil)
		capture(vm.POP, nil, baseFee)

	case "block_coinbase":
		if !checkOuts(20) {
			return
		}
		address := outs[:20]
		capture(vm.COINBASE, nil)
		capture(vm.POP, nil, address)

	case "block_gas_limit":
		if !checkOuts(8) {
			return
		}
		gasLimit := outs[:8]
		capture(vm.GASLIMIT, nil)
		capture(vm.POP, nil, gasLimit)

	case "block_number":
		if !checkOuts(8) {
			return
		}
		blockNumber := outs[:8]
		capture(vm.NUMBER, nil)
		capture(vm.POP, nil, blockNumber)

	case "block_timestamp":
		if !checkOuts(8) {
			return
		}
		timestamp := outs[:8]
		capture(vm.TIMESTAMP, nil)
		capture(vm.POP, nil, timestamp)

	case "chainid":
		if !checkOuts(8) {
			return
		}
		chainId := outs[:8]
		capture(vm.CHAINID, nil)
		capture(vm.POP, nil, chainId)

	case "contract_address":
		if !checkOuts(20) {
			return
		}
		address := outs[:20]
		capture(vm.ADDRESS, nil)
		capture(vm.POP, nil, address)

	case "evm_gas_left", "evm_ink_left":
		if !checkOuts(8) {
			return
		}
		gas := outs[:8]
		capture(vm.GAS, nil)
		capture(vm.POP, nil, gas)

	case "math_div":
		if !checkArgs(32+32) || !checkOuts(32) {
			return
		}
		a := args[:32]
		b := args[32:64]
		result := outs[:32]
		capture(vm.DIV, nil, a, b)
		capture(vm.POP, nil, result)

	case "math_mod":
		if !checkArgs(32+32) || !checkOuts(32) {
			return
		}
		a := args[:32]
		b := args[32:64]
		result := outs[:32]
		capture(vm.MOD, nil, a, b)
		capture(vm.POP, nil, result)

	case "math_pow":
		if !checkArgs(32+32) || !checkOuts(32) {
			return
		}
		a := args[:32]
		b := args[32:64]
		result := outs[:32]
		capture(vm.EXP, nil, a, b)
		capture(vm.POP, nil, result)

	case "math_add_mod":
		if !checkArgs(32+32+32) || !checkOuts(32) {
			return
		}
		a := args[:32]
		b := args[32:64]
		c := args[64:96]
		result := outs[:32]
		capture(vm.ADDMOD, nil, a, b, c)
		capture(vm.POP, nil, result)

	case "math_mul_mod":
		if !checkArgs(32+32+32) || !checkOuts(32) {
			return
		}
		a := args[:32]
		b := args[32:64]
		c := args[64:96]
		result := outs[:32]
		capture(vm.MULMOD, nil, a, b, c)
		capture(vm.POP, nil, result)

	case "msg_sender":
		if !checkOuts(20) {
			return
		}
		address := outs[:20]
		capture(vm.CALLER, nil)
		capture(vm.POP, nil, address)

	case "msg_value":
		if !checkOuts(32) {
			return
		}
		value := outs[:32]
		capture(vm.CALLVALUE, nil)
		capture(vm.POP, nil, value)

	case "native_keccak256":
		if !checkOuts(32) {
			return
		}
		offset := []byte(nil)
		size := intToBytes(len(args))
		hash := outs[:32]
		capture(vm.KECCAK256, args, offset, size)
		capture(vm.POP, args, hash)

	case "tx_gas_price":
		if !checkOuts(32) {
			return
		}
		price := outs[:32]
		capture(vm.GASPRICE, nil)
		capture(vm.POP, nil, price)

	case "tx_ink_price":
		if !checkOuts(4) {
			return
		}
		price := outs[:4]
		capture(vm.GASPRICE, nil)
		capture(vm.POP, nil, price)

	case "tx_origin":
		if !checkOuts(20) {
			return
		}
		address := outs[:20]
		capture(vm.ORIGIN, nil)
		capture(vm.POP, nil, address)

	case "user_entrypoint", "user_returned", "msg_reentrant", "write_result", "pay_for_memory_grow", "console_log_test", "console_log":
		// No EVM counterpart

	default:
		log.Warn("unhandled hostio trace", "name", name)
	}
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
