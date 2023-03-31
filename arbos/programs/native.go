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
GoApiStatus contractCallWrap(usize api, Bytes20 contract, RustVec * data, u64 * gas, Bytes32 value, u32 * len);
GoApiStatus delegateCallWrap(usize api, Bytes20 contract, RustVec * data, u64 * gas,                u32 * len);
GoApiStatus staticCallWrap  (usize api, Bytes20 contract, RustVec * data, u64 * gas,                u32 * len);
void        getReturnDataWrap(usize api, RustVec * data);
void        emitLogWrap(usize api, RustVec * data, usize topics);
*/
import "C"
import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
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
	stylusParams *goParams,
) ([]byte, error) {
	contract := scope.Contract
	readOnly := interpreter.ReadOnly()
	evm := interpreter.Evm()

	actingAddress := contract.Address() // not necessarily WASM
	program := actingAddress
	if contract.CodeAddr != nil {
		program = *contract.CodeAddr
	}
	if db, ok := db.(*state.StateDB); ok {
		db.RecordProgram(program, stylusParams.version)
	}
	module := db.GetCompiledWasmCode(program, stylusParams.version)

	// closures so Rust can call back into Go
	getBytes32 := func(key common.Hash) (common.Hash, uint64) {
		if tracingInfo != nil {
			tracingInfo.RecordStorageGet(key)
		}
		cost := vm.WasmStateLoadCost(db, actingAddress, key)
		return db.GetState(actingAddress, key), cost
	}
	setBytes32 := func(key, value common.Hash) (uint64, error) {
		if tracingInfo != nil {
			tracingInfo.RecordStorageSet(key, value)
		}
		if readOnly {
			return 0, vm.ErrWriteProtection
		}
		cost := vm.WasmStateStoreCost(db, actingAddress, key, value)
		db.SetState(actingAddress, key, value)
		return cost, nil
	}
	doCall := func(
		contract common.Address, opcode vm.OpCode, input []byte, gas uint64, value *big.Int,
	) (uint32, uint64, error) {
		// This closure can perform each kind of contract call based on the opcode passed in.
		// The implementation for each should match that of the EVM.
		//
		// Note that while the Yellow Paper is authoritative, the following go-ethereum
		// functions provide corresponding implementations in the vm package.
		//     - operations_acl.go makeCallVariantGasCallEIP2929()
		//     - gas_table.go      gasCall() gasDelegateCall() gasStaticCall()
		//     - instructions.go   opCall()  opDelegateCall()  opStaticCall()
		//

		// read-only calls are not payable (opCall)
		if readOnly && value.Sign() != 0 {
			return 0, 0, vm.ErrWriteProtection
		}

		startGas := gas

		// computes makeCallVariantGasCallEIP2929 and gasCall/gasDelegateCall/gasStaticCall
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
			tracingInfo.Tracer.CaptureState(0, opcode, startGas-gas, startGas, scope, []byte{}, depth, nil)
		}

		// EVM rule: calls that pay get a stipend (opCall)
		if value.Sign() != 0 {
			gas = arbmath.SaturatingUAdd(gas, params.CallStipend)
		}

		var ret []byte
		var returnGas uint64

		switch opcode {
		case vm.CALL:
			ret, returnGas, err = evm.Call(scope.Contract, contract, input, gas, value)
		case vm.DELEGATECALL:
			ret, returnGas, err = evm.DelegateCall(scope.Contract, contract, input, gas)
		case vm.STATICCALL:
			ret, returnGas, err = evm.StaticCall(scope.Contract, contract, input, gas)
		default:
			log.Crit("unsupported call type", "opcode", opcode)
		}

		interpreter.SetReturnData(ret)
		cost := arbmath.SaturatingUSub(startGas, returnGas)
		return uint32(len(ret)), cost, err
	}
	contractCall := func(contract common.Address, input []byte, gas uint64, value *big.Int) (uint32, uint64, error) {
		return doCall(contract, vm.CALL, input, gas, value)
	}
	delegateCall := func(contract common.Address, input []byte, gas uint64) (uint32, uint64, error) {
		return doCall(contract, vm.DELEGATECALL, input, gas, common.Big0)
	}
	staticCall := func(contract common.Address, input []byte, gas uint64) (uint32, uint64, error) {
		return doCall(contract, vm.STATICCALL, input, gas, common.Big0)
	}
	getReturnData := func() []byte {
		data := interpreter.GetReturnData()
		if data == nil {
			return []byte{}
		}
		return data
	}
	emitLog := func(data []byte, topics int) error {
		if readOnly {
			return vm.ErrWriteProtection
		}
		hashes := make([]common.Hash, topics)
		for i := 0; i < topics; i++ {
			hashes[i] = common.BytesToHash(data[:(i+1)*32])
		}
		event := &types.Log{
			Address:     actingAddress,
			Topics:      hashes,
			Data:        data[32*topics:],
			BlockNumber: evm.Context.BlockNumber.Uint64(),
			// Geth will set other fields
		}
		db.AddLog(event)
		return nil
	}

	output := &rustVec{}
	status := userStatus(C.stylus_call(
		goSlice(module),
		goSlice(calldata),
		stylusParams.encode(),
		newAPI(getBytes32, setBytes32, contractCall, delegateCall, staticCall, getReturnData, emitLog),
		output,
		(*u64)(&contract.Gas),
	))
	data, err := status.output(output.intoBytes())

	if status == userFailure {
		log.Debug("program failure", "err", string(data), "program", actingAddress)
	}

	return data, err
}

type apiStatus = C.GoApiStatus

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
func setBytes32Impl(api usize, key, value bytes32, cost *u64, errVec *rustVec) apiStatus {
	closure := getAPI(api)

	gas, err := closure.setBytes32(key.toHash(), value.toHash())
	if err != nil {
		errVec.setString(err.Error())
		return apiFailure
	}
	*cost = u64(gas)
	return apiSuccess
}

//export contractCallImpl
func contractCallImpl(api usize, contract bytes20, data *rustVec, evmGas *u64, value bytes32, len *u32) apiStatus {
	closure := getAPI(api)
	defer data.drop()

	ret_len, cost, err := closure.contractCall(contract.toAddress(), data.read(), uint64(*evmGas), value.toBig())
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		return apiFailure
	}
	return apiSuccess
}

//export delegateCallImpl
func delegateCallImpl(api usize, contract bytes20, data *rustVec, evmGas *u64, len *u32) apiStatus {
	closure := getAPI(api)
	defer data.drop()

	ret_len, cost, err := closure.delegateCall(contract.toAddress(), data.read(), uint64(*evmGas))
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		return apiFailure
	}
	return apiSuccess
}

//export staticCallImpl
func staticCallImpl(api usize, contract bytes20, data *rustVec, evmGas *u64, len *u32) apiStatus {
	closure := getAPI(api)
	defer data.drop()

	ret_len, cost, err := closure.staticCall(contract.toAddress(), data.read(), uint64(*evmGas))
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

//export emitLogImpl
func emitLogImpl(api usize, data *rustVec, topics usize) apiStatus {
	closure := getAPI(api)
	err := closure.emitLog(data.read(), int(topics))
	if err != nil {
		data.setString(err.Error())
		return apiFailure
	}
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

func (vec *rustVec) read() []byte {
	return arbutil.PointerToSlice((*byte)(vec.ptr), int(vec.len))
}

func (vec *rustVec) intoBytes() []byte {
	slice := vec.read()
	C.stylus_drop_vec(*vec)
	return slice
}

func (vec *rustVec) drop() {
	C.stylus_drop_vec(*vec)
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
