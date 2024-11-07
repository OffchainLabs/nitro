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
	Tracer       vm.EVMLogger
	Scenario     TracingScenario
	Contract     *vm.Contract
	Depth        int
	storageCache *storageCache
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
		Tracer:       evm.Config.Tracer,
		Scenario:     scenario,
		Contract:     vm.NewContract(addressHolder{to}, addressHolder{from}, uint256.NewInt(0), 0),
		Depth:        evm.Depth(),
		storageCache: newStorageCache(),
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

func (info *TracingInfo) CaptureEVMTraceForHostio(name string, args, outs []byte, startInk, endInk uint64) {
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
		gas := endInk / inkToGas
		var cost uint64
		if firstOpcode {
			cost = (startInk - endInk) / inkToGas
			firstOpcode = false
		} else {
			// When capturing multiple opcodes, usually the first one is the relevant
			// action and the following ones just pop the result values from the stack.
			cost = 0
		}
		info.captureState(op, gas, cost, memory, stackValues...)
	}

	switch name {
	case "read_args":
		destOffset := []byte(nil)
		offset := []byte(nil)
		size := lenToBytes(outs)
		capture(vm.CALLDATACOPY, outs, destOffset, offset, size)

	case "storage_load_bytes32":
		if !checkArgs(32) || !checkOuts(32) {
			return
		}
		key := args[:32]
		value := outs[:32]
		if info.storageCache.Load(common.Hash(key), common.Hash(value)) {
			capture(vm.SLOAD, nil, key)
			capture(vm.POP, nil, value)
		}

	case "storage_cache_bytes32":
		if !checkArgs(32 + 32) {
			return
		}
		key := args[:32]
		value := args[32:64]
		info.storageCache.Store(common.Hash(key), common.Hash(value))

	case "storage_flush_cache":
		if !checkArgs(1) {
			return
		}
		toClear := args[0] != 0
		for _, store := range info.storageCache.Flush() {
			capture(vm.SSTORE, nil, store.Key.Bytes(), store.Value.Bytes())
		}
		if toClear {
			info.storageCache.Clear()
		}

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

	case "create1":
		if !checkArgs(32) || !checkOuts(20) {
			return
		}
		value := args[:32]
		code := args[32:]
		offset := []byte(nil)
		size := lenToBytes(code)
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
		size := lenToBytes(code)
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
		size := lenToBytes(data)
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
		size := lenToBytes(args)
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

	case "call_contract", "delegate_call_contract", "static_call_contract":
		// The API receives the CaptureHostIO after the EVM call is done but we want to
		// capture the opcde before it. So, we capture the state in CaptureStylusCall.

	case "write_result", "exit_early":
		// These calls are handled on CaptureStylusExit to also cover the normal exit case.

	case "user_entrypoint", "user_returned", "msg_reentrant", "pay_for_memory_grow", "console_log_text", "console_log":
		// No EVM counterpart

	default:
		log.Warn("unhandled hostio trace", "name", name)
	}
}

func (info *TracingInfo) CaptureStylusCall(opCode vm.OpCode, contract common.Address, value *uint256.Int, input []byte, gas, startGas, baseCost uint64) {
	var stack [][]byte
	stack = append(stack, intToBytes(gas))  // gas
	stack = append(stack, contract.Bytes()) // address
	if opCode == vm.CALL {
		stack = append(stack, value.Bytes()) // call value
	}
	stack = append(stack, []byte(nil))       // memory offset
	stack = append(stack, lenToBytes(input)) // memory length
	stack = append(stack, []byte(nil))       // return offset
	stack = append(stack, []byte(nil))       // return size
	info.captureState(opCode, startGas, baseCost+gas, input, stack...)
}

func (info *TracingInfo) CaptureStylusExit(status uint8, data []byte, err error, gas uint64) {
	var opCode vm.OpCode
	if status == 0 {
		if len(data) == 0 {
			info.captureState(vm.STOP, gas, 0, nil)
			return
		}
		opCode = vm.RETURN
	} else {
		opCode = vm.REVERT
		if data == nil {
			data = []byte(err.Error())
		}
	}
	offset := []byte(nil)
	size := lenToBytes(data)
	info.captureState(opCode, gas, 0, data, offset, size)
}

func (info *TracingInfo) captureState(op vm.OpCode, gas uint64, cost uint64, memory []byte, stackValues ...[]byte) {
	stack := []uint256.Int{}
	for _, value := range stackValues {
		stack = append(stack, *uint256.NewInt(0).SetBytes(value))
	}
	scope := &vm.ScopeContext{
		Memory:   TracingMemoryFromBytes(memory),
		Stack:    TracingStackFromArgs(stack...),
		Contract: info.Contract,
	}
	info.Tracer.CaptureState(0, op, gas, cost, scope, []byte{}, info.Depth, nil)
}

func lenToBytes(data []byte) []byte {
	return intToBytes(uint64(len(data)))
}

func intToBytes(v uint64) []byte {
	return binary.BigEndian.AppendUint64(nil, v)
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
