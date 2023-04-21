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
GoApiStatus create1Wrap(usize api, RustVec * code, Bytes32 endowment,               u64 * gas, u32 * len);
GoApiStatus create2Wrap(usize api, RustVec * code, Bytes32 endowment, Bytes32 salt, u64 * gas, u32 * len);
void        getReturnDataWrap(usize api, RustVec * data);
GoApiStatus emitLogWrap(usize api, RustVec * data, usize topics);
*/
import "C"
import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
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
	output := &rustVec{}
	status := userStatus(C.stylus_compile(
		goSlice(wasm),
		u32(version),
		usize(arbmath.BoolToUint32(debug)),
		output,
	))
	data := output.intoBytes()
	result, err := status.output(data)
	if err == nil {
		db.SetCompiledWasmCode(program, result, version)
	} else {
		log.Debug("program failure", "err", err.Error(), "data", string(data), "program", program)
	}
	return err
}

func callUserWasm(
	scope *vm.ScopeContext,
	db vm.StateDB,
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	calldata []byte,
	evmData *evmData,
	stylusParams *goParams,
) ([]byte, error) {
	contract := scope.Contract
	actingAddress := contract.Address() // not necessarily WASM
	program := actingAddress
	if contract.CodeAddr != nil {
		program = *contract.CodeAddr
	}
	if db, ok := db.(*state.StateDB); ok {
		db.RecordProgram(program, stylusParams.version)
	}
	module := db.GetCompiledWasmCode(program, stylusParams.version)

	api, id := wrapGoApi(newApi(interpreter, tracingInfo, scope))
	defer dropApi(id)

	output := &rustVec{}
	status := userStatus(C.stylus_call(
		goSlice(module),
		goSlice(calldata),
		stylusParams.encode(),
		api,
		evmData.encode(),
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
	closure := getApi(api)
	value, gas := closure.getBytes32(key.toHash())
	*cost = u64(gas)
	return hashToBytes32(value)
}

//export setBytes32Impl
func setBytes32Impl(api usize, key, value bytes32, cost *u64, errVec *rustVec) apiStatus {
	closure := getApi(api)

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
	closure := getApi(api)
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
	closure := getApi(api)
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
	closure := getApi(api)
	defer data.drop()

	ret_len, cost, err := closure.staticCall(contract.toAddress(), data.read(), uint64(*evmGas))
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		return apiFailure
	}
	return apiSuccess
}

//export create1Impl
func create1Impl(api usize, code *rustVec, endowment bytes32, evmGas *u64, len *u32) apiStatus {
	closure := getApi(api)
	addr, ret_len, cost, err := closure.create1(code.read(), endowment.toBig(), uint64(*evmGas))
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		code.setString(err.Error())
		return apiFailure
	}
	code.setBytes(addr.Bytes())
	return apiSuccess
}

//export create2Impl
func create2Impl(api usize, code *rustVec, endowment, salt bytes32, evmGas *u64, len *u32) apiStatus {
	closure := getApi(api)
	addr, ret_len, cost, err := closure.create2(code.read(), endowment.toBig(), salt.toBig(), uint64(*evmGas))
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		code.setString(err.Error())
		return apiFailure
	}
	code.setBytes(addr.Bytes())
	return apiSuccess
}

//export getReturnDataImpl
func getReturnDataImpl(api usize, output *rustVec) {
	closure := getApi(api)
	return_data := closure.getReturnData()
	output.setBytes(return_data)
}

//export emitLogImpl
func emitLogImpl(api usize, data *rustVec, topics usize) apiStatus {
	closure := getApi(api)
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

func addressToBytes20(addr common.Address) bytes20 {
	value := bytes20{}
	for index, b := range addr.Bytes() {
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
		version:    u32(params.version),
		max_depth:  u32(params.maxDepth),
		ink_price:  u64(params.inkPrice),
		hostio_ink: u64(params.hostioInk),
		debug_mode: u32(params.debugMode),
	}
}

func (data *evmData) encode() C.EvmData {
	return C.EvmData{
		origin: addressToBytes20(data.origin),
	}
}
