// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build !js
// +build !js

package programs

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#cgo LDFLAGS: ${SRCDIR}/../../target/lib/libstylus.a -ldl -lm
#include "arbitrator.h"

typedef uint8_t u8;
typedef uint16_t u16;
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
)

type u8 = C.uint8_t
type u16 = C.uint16_t
type u32 = C.uint32_t
type u64 = C.uint64_t
type usize = C.size_t
type bytes20 = C.Bytes20
type bytes32 = C.Bytes32
type rustVec = C.RustVec

func compileUserWasm(db vm.StateDB, program common.Address, wasm []byte, version uint32, debug bool) (uint16, error) {
	open, _ := db.GetStylusPages()
	pageLimit := arbmath.SaturatingUSub(initialMachinePageLimit, *open)
	footprint := uint16(0)

	output := &rustVec{}
	status := userStatus(C.stylus_compile(
		goSlice(wasm),
		u32(version),
		u16(pageLimit),
		(*u16)(&footprint),
		output,
		usize(arbmath.BoolToUint32(debug)),
	))
	data := output.intoBytes()
	result, err := status.output(data)
	if err == nil {
		db.SetCompiledWasmCode(program, result, version)
	} else {
		data := arbutil.ToStringOrHex(data)
		log.Debug("compile failure", "err", err.Error(), "data", data, "program", program)
	}
	return footprint, err
}

func callUserWasm(
	program Program,
	scope *vm.ScopeContext,
	db vm.StateDB,
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	calldata []byte,
	evmData *evmData,
	stylusParams *goParams,
) ([]byte, error) {
	if db, ok := db.(*state.StateDB); ok {
		db.RecordProgram(program.address, stylusParams.version)
	}
	module := db.GetCompiledWasmCode(program.address, stylusParams.version)

	evmApi, id := newApi(interpreter, tracingInfo, scope)
	defer dropApi(id)

	output := &rustVec{}
	status := userStatus(C.stylus_call(
		goSlice(module),
		goSlice(calldata),
		stylusParams.encode(),
		evmApi,
		evmData.encode(),
		u32(stylusParams.debugMode),
		output,
		(*u64)(&scope.Contract.Gas),
	))
	returnData := output.intoBytes()
	data, err := status.output(returnData)

	if status == userFailure {
		str := arbutil.ToStringOrHex(returnData)
		log.Debug("program failure", "err", string(data), "program", program.address, "returnData", str)
	}
	return data, err
}

type apiStatus = C.EvmApiStatus

const apiSuccess C.EvmApiStatus = C.EvmApiStatus_Success
const apiFailure C.EvmApiStatus = C.EvmApiStatus_Failure

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

//export accountBalanceImpl
func accountBalanceImpl(api usize, address bytes20, cost *u64) bytes32 {
	closures := getApi(api)
	balance, gas := closures.accountBalance(address.toAddress())
	*cost = u64(gas)
	return hashToBytes32(balance)
}

//export accountCodeHashImpl
func accountCodeHashImpl(api usize, address bytes20, cost *u64) bytes32 {
	closures := getApi(api)
	codehash, gas := closures.accountCodeHash(address.toAddress())
	*cost = u64(gas)
	return hashToBytes32(codehash)
}

//export evmBlockHashImpl
func evmBlockHashImpl(api usize, block bytes32) bytes32 {
	closures := getApi(api)
	hash := closures.evmBlockHash(block.toHash())
	return hashToBytes32(hash)
}

//export addPagesImpl
func addPagesImpl(api usize, pages u16, open *u16, ever *u16) {
	closures := getApi(api)
	openPages, everPages := closures.addPages(uint16(pages))
	*open = u16(openPages)
	*ever = u16(everPages)
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

func (params *goParams) encode() C.StylusConfig {
	pricing := C.PricingParams{
		ink_price:    u64(params.inkPrice),
		hostio_ink:   u64(params.hostioInk),
		memory_model: params.memoryModel.encode(),
	}
	return C.StylusConfig{
		version:   u32(params.version),
		max_depth: u32(params.maxDepth),
		pricing:   pricing,
	}
}

func (model *goMemoryModel) encode() C.MemoryModel {
	return C.MemoryModel{
		free_pages: u16(model.freePages),
		page_gas:   u32(model.pageGas),
		page_ramp:  u32(model.pageRamp),
	}
}

func (data *evmData) encode() C.EvmData {
	return C.EvmData{
		block_basefee:    hashToBytes32(data.blockBasefee),
		block_chainid:    hashToBytes32(data.blockChainId),
		block_coinbase:   addressToBytes20(data.blockCoinbase),
		block_difficulty: hashToBytes32(data.blockDifficulty),
		block_gas_limit:  u64(data.blockGasLimit),
		block_number:     hashToBytes32(data.blockNumber),
		block_timestamp:  u64(data.blockTimestamp),
		contract_address: addressToBytes20(data.contractAddress),
		msg_sender:       addressToBytes20(data.msgSender),
		msg_value:        hashToBytes32(data.msgValue),
		tx_gas_price:     hashToBytes32(data.txGasPrice),
		tx_origin:        addressToBytes20(data.txOrigin),
		footprint:        u16(data.footprint),
		return_data_len:  0,
	}
}
