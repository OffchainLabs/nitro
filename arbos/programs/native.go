// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build !js
// +build !js

package programs

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#cgo LDFLAGS: ${SRCDIR}/../../target/lib/libstylus.a -ldl -lm
#include "arbitrator.h"

Bytes32 getBytes32Wrap(size_t api, Bytes32 key, uint64_t * cost);
uint8_t setBytes32Wrap(size_t api, Bytes32 key, Bytes32 value, uint64_t * cost, RustVec * error);
*/
import "C"
import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type u8 = C.uint8_t
type u32 = C.uint32_t
type u64 = C.uint64_t
type usize = C.size_t
type bytes20 = C.Bytes20
type bytes32 = C.Bytes32

func compileUserWasm(db vm.StateDB, program common.Address, wasm []byte, version uint32, debug bool) error {
	debugMode := 0
	if debug {
		debugMode = 1
	}

	output := &C.RustVec{}
	status := userStatus(C.stylus_compile(
		goSlice(wasm),
		u32(version),
		usize(debugMode),
		output,
	))
	result, err := status.output(output.intoBytes())
	if err == nil {
		db.SetCompiledWasmCode(program, result, version)
	}
	return err
}

func callUserWasm(
	scope *vm.ScopeContext,
	db vm.StateDB,
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	calldata []byte,
	gas *uint64,
	stylusParams *goParams,
) ([]byte, error) {
	program := scope.Contract.Address()

	if db, ok := db.(*state.StateDB); ok {
		db.RecordProgram(program, stylusParams.version)
	}

	module := db.GetCompiledWasmCode(program, stylusParams.version)
	readOnly := interpreter.ReadOnly()

	getBytes32 := func(key common.Hash) (common.Hash, uint64) {
		if tracingInfo != nil {
			tracingInfo.RecordStorageGet(key)
		}
		cost := vm.WasmStateLoadCost(db, program, key)
		return db.GetState(program, key), cost
	}
	setBytes32 := func(key, value common.Hash) (uint64, error) {
		if tracingInfo != nil {
			tracingInfo.RecordStorageSet(key, value)
		}
		if readOnly {
			return 0, vm.ErrWriteProtection
		}
		cost := vm.WasmStateStoreCost(db, program, key, value)
		db.SetState(program, key, value)
		return cost, nil
	}
	callContract := func(contract common.Address, input []byte, gas uint64, value *big.Int) ([]byte, uint64, error) {
		// This closure performs a contract call. The implementation should match that of the EVM.
		//
		// Note that while the Yellow Paper is authoritative, the following go-ethereum
		// functions provide a corresponding implementation in the vm package.
		//     - operations_acl.go makeCallVariantGasCallEIP2929()
		//     - gas_table.go      gasCall()
		//     - instructions.go   opCall()
		//
		// TODO: handle custom return calldata
		//

		// read-only calls are not payable (opCall)
		if readOnly && value.Sign() != 0 {
			return nil, 0, vm.ErrWriteProtection
		}

		evm := interpreter.Evm()
		startGas := gas

		// computes makeCallVariantGasCallEIP2929 and gasCall
		baseCost, err := vm.WasmCallCost(db, contract, value, startGas)
		if err != nil {
			return nil, 0, err
		}
		gas -= baseCost
		gas = gas - gas/64

		// Tracing: emit the call (value transfer is done later in evm.Call)
		if tracingInfo != nil {
			depth := evm.Depth()
			tracingInfo.Tracer.CaptureState(0, vm.CALL, startGas-gas, startGas, scope, []byte{}, depth, nil)
		}

		// EVM rule: calls that pay get a stipend (opCall)
		if value.Sign() != 0 {
			gas = arbmath.SaturatingUAdd(gas, params.CallStipend)
		}

		ret, returnGas, err := evm.Call(scope.Contract, contract, input, gas, value)
		if err != nil && errors.Is(err, vm.ErrExecutionReverted) {
			ret = []byte{}
		}

		cost := arbmath.SaturatingUSub(startGas, returnGas)
		return ret, cost, err
	}

	output := &C.RustVec{}
	status := userStatus(C.stylus_call(
		goSlice(module),
		goSlice(calldata),
		stylusParams.encode(),
		newAPI(getBytes32, setBytes32, callContract),
		output,
		(*u64)(gas),
	))
	data, err := status.output(output.intoBytes())
	if status == userFailure {
		log.Debug("program failure", "err", string(data), "program", program)
	}
	return data, err
}

const (
	apiSuccess u8 = iota
	apiFailure
)

//export getBytes32Impl
func getBytes32Impl(api usize, key bytes32, cost *u64) bytes32 {
	closure, err := getAPI(api)
	if err != nil {
		log.Error(err.Error())
		return bytes32{}
	}
	value, gas := closure.getBytes32(key.toHash())
	*cost = u64(gas)
	return hashToBytes32(value)
}

//export setBytes32Impl
func setBytes32Impl(api usize, key, value bytes32, cost *u64, vec *C.RustVec) u8 {
	closure, err := getAPI(api)
	if err != nil {
		vec.setString(err.Error())
		log.Error(err.Error())
		return apiFailure
	}
	gas, err := closure.setBytes32(key.toHash(), value.toHash())
	if err != nil {
		vec.setString(err.Error())
		return apiFailure
	}
	*cost = u64(gas)
	return apiSuccess
}

//export callContractImpl
func callContractImpl(api usize, contract bytes20, data *C.RustVec, evmGas *u64, value bytes32) u8 {
	closure, err := getAPI(api)
	if err != nil {
		log.Error(err.Error())
		return apiFailure
	}

	result, gasLeft, err := closure.callContract(contract.toAddress(), data.read(), uint64(*evmGas), value.toBig())
	data.setBytes(result)
	if err != nil {
		return apiFailure
	}
	*evmGas = u64(gasLeft)
	return apiSuccess
}

func (value bytes20) toAddress() common.Address {
	addr := common.Address{}
	for index, b := range value.bytes {
		addr[index] = byte(b)
	}
	return addr
}

func (value bytes32) toHash() common.Hash {
	hash := common.Hash{}
	for index, b := range value.bytes {
		hash[index] = byte(b)
	}
	return hash
}

func (value bytes32) toBig() *big.Int {
	return value.toHash().Big()
}

func hashToBytes32(hash common.Hash) bytes32 {
	value := bytes32{}
	for index, b := range hash.Bytes() {
		value.bytes[index] = u8(b)
	}
	return value
}

func (vec *C.RustVec) read() []byte {
	return arbutil.PointerToSlice((*byte)(vec.ptr), int(vec.len))
}

func (vec *C.RustVec) intoBytes() []byte {
	slice := vec.read()
	C.stylus_free(*vec)
	return slice
}

func (vec *C.RustVec) setString(data string) {
	vec.setBytes([]byte(data))
}

func (vec *C.RustVec) setBytes(data []byte) {
	C.stylus_vec_set_bytes(vec, goSlice(data))
}

func goSlice(slice []byte) C.GoSliceData {
	return C.GoSliceData{
		ptr: (*u8)(arbutil.SliceToPointer(slice)),
		len: usize(len(slice)),
	}
}

func (params *goParams) encode() C.GoParams {
	return C.GoParams{
		version:        u32(params.version),
		max_depth:      u32(params.maxDepth),
		wasm_gas_price: u64(params.wasmGasPrice),
		hostio_cost:    u64(params.hostioCost),
		debug_mode:     usize(params.debugMode),
	}
}
