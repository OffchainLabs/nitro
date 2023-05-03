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
	"github.com/offchainlabs/nitro/util/colors"
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

	evmApi, id := newApi(interpreter, tracingInfo, scope)
	defer dropApi(id)

	output := &rustVec{}
	status := userStatus(C.stylus_call(
		goSlice(module),
		goSlice(calldata),
		stylusParams.encode(),
		evmApi,
		evmData.encode(),
		output,
		(*u64)(&contract.Gas),
	))
	returnData := output.intoBytes()
	data, err := status.output(returnData)

	if status == userFailure {
		log.Debug("program failure", "err", string(data), "program", actingAddress, "returnData", colors.Uncolor(arbutil.ToStringOrHex(returnData)))
	}
	return data, err
}

type apiStatus = C.EvmApiStatus

const apiSuccess C.EvmApiStatus = C.EvmApiStatus_Success
const apiFailure C.EvmApiStatus = C.EvmApiStatus_Failure

//export addressBalanceImpl
func addressBalanceImpl(api usize, address bytes20, cost *u64) bytes32 {
	closures := getApi(api)
	value, gas := closures.addressBalance(address.toAddress())
	*cost = u64(gas)
	return hashToBytes32(value)
}

//export addressCodeHashImpl
func addressCodeHashImpl(api usize, address bytes20, cost *u64) bytes32 {
	closures := getApi(api)
	value, gas := closures.addressCodeHash(address.toAddress())
	*cost = u64(gas)
	return hashToBytes32(value)
}

//export evmBlockHashImpl
func evmBlockHashImpl(api usize, block bytes32, cost *u64) bytes32 {
	closures := getApi(api)
	value, gas := closures.evmBlockHash(block.toHash())
	*cost = u64(gas)
	return hashToBytes32(value)
}

//export getBytes32Impl
func getBytes32Impl(api usize, key bytes32, cost *u64) bytes32 {
	closures := getApi(api)
	value, gas := closures.getBytes32(key.toHash())
	*cost = u64(gas)
	return hashToBytes32(value)
}

//export setBytes32Impl
func setBytes32Impl(api usize, key, value bytes32, cost *u64, errVec *rustVec) apiStatus {
	closures := getApi(api)

	gas, err := closures.setBytes32(key.toHash(), value.toHash())
	if err != nil {
		errVec.setString(err.Error())
		return apiFailure
	}
	*cost = u64(gas)
	return apiSuccess
}

//export contractCallImpl
func contractCallImpl(api usize, contract bytes20, data *rustVec, evmGas *u64, value bytes32, len *u32) apiStatus {
	closures := getApi(api)
	defer data.drop()

	ret_len, cost, err := closures.contractCall(contract.toAddress(), data.read(), uint64(*evmGas), value.toBig())
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		return apiFailure
	}
	return apiSuccess
}

//export delegateCallImpl
func delegateCallImpl(api usize, contract bytes20, data *rustVec, evmGas *u64, len *u32) apiStatus {
	closures := getApi(api)
	defer data.drop()

	ret_len, cost, err := closures.delegateCall(contract.toAddress(), data.read(), uint64(*evmGas))
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		return apiFailure
	}
	return apiSuccess
}

//export staticCallImpl
func staticCallImpl(api usize, contract bytes20, data *rustVec, evmGas *u64, len *u32) apiStatus {
	closures := getApi(api)
	defer data.drop()

	ret_len, cost, err := closures.staticCall(contract.toAddress(), data.read(), uint64(*evmGas))
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		return apiFailure
	}
	return apiSuccess
}

//export create1Impl
func create1Impl(api usize, code *rustVec, endowment bytes32, evmGas *u64, len *u32) apiStatus {
	closures := getApi(api)
	addr, ret_len, cost, err := closures.create1(code.read(), endowment.toBig(), uint64(*evmGas))
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
	closures := getApi(api)
	addr, ret_len, cost, err := closures.create2(code.read(), endowment.toBig(), salt.toBig(), uint64(*evmGas))
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
	closures := getApi(api)
	return_data := closures.getReturnData()
	output.setBytes(return_data)
}

//export emitLogImpl
func emitLogImpl(api usize, data *rustVec, topics u32) apiStatus {
	closures := getApi(api)
	err := closures.emitLog(data.read(), uint32(topics))
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

func bigToBytes32(big *big.Int) bytes32 {
	return hashToBytes32(common.BigToHash(big))
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
		block_basefee:    bigToBytes32(data.block_basefee),
		block_chainid:    bigToBytes32(data.block_chainid),
		block_coinbase:   addressToBytes20(data.block_coinbase),
		block_difficulty: bigToBytes32(data.block_difficulty),
		block_gas_limit:  C.uint64_t(data.block_gas_limit),
		block_number:     bigToBytes32(data.block_number),
		block_timestamp:  bigToBytes32(data.block_timestamp),
		contract_address: addressToBytes20(data.contract_address),
		msg_sender:       addressToBytes20(data.msg_sender),
		msg_value:        bigToBytes32(data.msg_value),
		gas_price:        bigToBytes32(data.gas_price),
		origin:           addressToBytes20(data.origin),
	}
}
