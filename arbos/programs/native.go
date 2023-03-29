// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build !js
// +build !js

package programs

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#cgo LDFLAGS: ${SRCDIR}/../../target/lib/libstylus.a -ldl -lm
#include "arbitrator.h"

typedef uint32_t u32;
typedef uint64_t u64;
typedef size_t usize;

Bytes32     getBytes32Wrap(usize api, Bytes32 key, u64 * cost);
GoApiStatus setBytes32Wrap(usize api, Bytes32 key, Bytes32 value, u64 * cost, RustVec * error);
GoApiStatus callContractWrap(usize api, Bytes20 contract, RustVec * calldata, u64 * gas, Bytes32 value, u32 * len);
void        getReturnDataWrap(usize api, RustVec * data);
*/
import "C"
import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
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
type rustVec = C.RustVec

func compileUserWasm(db vm.StateDB, program common.Address, wasm []byte, version uint32, debug bool) error {
	debugMode := 0
	if debug {
		debugMode = 1
	}

	output := &rustVec{}
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
	msg core.Message,
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
	callContract := func(contract common.Address, input []byte, gas uint64, value *big.Int) (uint32, uint64, error) {
		// This closure performs a contract call. The implementation should match that of the EVM.
		//
		// Note that while the Yellow Paper is authoritative, the following go-ethereum
		// functions provide a corresponding implementation in the vm package.
		//     - operations_acl.go makeCallVariantGasCallEIP2929()
		//     - gas_table.go      gasCall()
		//     - instructions.go   opCall()
		//

		// read-only calls are not payable (opCall)
		if readOnly && value.Sign() != 0 {
			return 0, 0, vm.ErrWriteProtection
		}

		evm := interpreter.Evm()
		startGas := gas

		// computes makeCallVariantGasCallEIP2929 and gasCall
		baseCost, err := vm.WasmCallCost(db, contract, value, startGas)
		if err != nil {
			return 0, 0, err
		}
		if gas < baseCost {
			return 0, 0, vm.ErrOutOfGas
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
		interpreter.SetReturnData(ret)
		cost := arbmath.SaturatingUSub(startGas, returnGas)
		return uint32(len(ret)), cost, err
	}
	getReturnData := func() []byte {
		data := interpreter.GetReturnData()
		if data == nil {
			return []byte{}
		}
		return data
	}

	output := &rustVec{}
	status := userStatus(C.stylus_call(
		goSlice(module),
		goSlice(calldata),
		stylusParams.encode(),
		newAPI(getBytes32, setBytes32, callContract, getReturnData),
		output,
		(*u64)(gas),
	))
	data, err := status.output(output.intoBytes())

	if status == userFailure {
		log.Debug("program failure", "err", string(data), "program", program)
	}
	return data, err
}

const apiSuccess C.GoApiStatus = C.GoApiStatus_Success
const apiFailure C.GoApiStatus = C.GoApiStatus_Failure

//export getBytes32Impl
func getBytes32Impl(api usize, key bytes32, cost *u64) bytes32 {
	closure := getAPI(api)
	value, gas := closure.getBytes32(key.toHash())
	*cost = u64(gas)
	return hashToBytes32(value)
}

//export setBytes32Impl
func setBytes32Impl(api usize, key, value bytes32, cost *u64, errVec *rustVec) C.GoApiStatus {
	closure := getAPI(api)

	gas, err := closure.setBytes32(key.toHash(), value.toHash())
	if err != nil {
		errVec.setString(err.Error())
		return apiFailure
	}
	*cost = u64(gas)
	return apiSuccess
}

//export callContractImpl
func callContractImpl(api usize, contract bytes20, data *rustVec, evmGas *u64, value bytes32, len *u32) C.GoApiStatus {
	closure := getAPI(api)

	ret_len, cost, err := closure.callContract(contract.toAddress(), data.read(), uint64(*evmGas), value.toBig())
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		return apiFailure
	}
	return apiSuccess
}

//export getReturnDataImpl
func getReturnDataImpl(api usize, output *rustVec) {
	closure := getAPI(api)
	return_data := closure.getReturnData()
	output.setBytes(return_data)
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

func (vec *rustVec) read() []byte {
	return arbutil.PointerToSlice((*byte)(vec.ptr), int(vec.len))
}

func (vec *rustVec) intoBytes() []byte {
	slice := vec.read()
	C.stylus_free(*vec)
	return slice
}

func (vec *rustVec) setString(data string) {
	vec.setBytes([]byte(data))
}

func (vec *rustVec) setBytes(data []byte) {
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
