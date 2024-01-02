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
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
)

type u8 = C.uint8_t
type u16 = C.uint16_t
type u32 = C.uint32_t
type u64 = C.uint64_t
type usize = C.size_t
type cbool = C._Bool
type bytes20 = C.Bytes20
type bytes32 = C.Bytes32
type rustBytes = C.RustBytes
type rustSlice = C.RustSlice

func activateProgram(
	db vm.StateDB,
	program common.Address,
	wasm []byte,
	page_limit uint16,
	version uint16,
	debug bool,
	burner burn.Burner,
) (common.Hash, uint16, error) {
	output := &rustBytes{}
	asmLen := usize(0)
	moduleHash := &bytes32{}
	footprint := uint16(math.MaxUint16)

	status := userStatus(C.stylus_activate(
		goSlice(wasm),
		u16(page_limit),
		u16(version),
		cbool(debug),
		output,
		&asmLen,
		moduleHash,
		(*u16)(&footprint),
		(*u64)(burner.GasLeft()),
	))

	data, msg, err := status.toResult(output.intoBytes(), debug)
	if err != nil {
		if debug {
			log.Warn("activation failed", "err", err, "msg", msg, "program", program)
		}
		if errors.Is(err, vm.ErrExecutionReverted) {
			return common.Hash{}, footprint, fmt.Errorf("%w: %s", ErrProgramActivation, msg)
		}
		return common.Hash{}, footprint, err
	}

	hash := moduleHash.toHash()
	split := int(asmLen)
	asm := data[:split]
	module := data[split:]

	db.ActivateWasm(hash, asm, module)
	return hash, footprint, err
}

func callProgram(
	address common.Address,
	moduleHash common.Hash,
	scope *vm.ScopeContext,
	db vm.StateDB,
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	calldata []byte,
	evmData *evmData,
	stylusParams *goParams,
	memoryModel *MemoryModel,
) ([]byte, error) {
	if db, ok := db.(*state.StateDB); ok {
		db.RecordProgram(moduleHash)
	}
	asm := db.GetActivatedAsm(moduleHash)

	evmApi, id := newApi(interpreter, tracingInfo, scope, memoryModel)
	defer dropApi(id)

	output := &rustBytes{}
	status := userStatus(C.stylus_call(
		goSlice(asm),
		goSlice(calldata),
		stylusParams.encode(),
		evmApi,
		evmData.encode(),
		u32(stylusParams.debugMode),
		output,
		(*u64)(&scope.Contract.Gas),
	))

	depth := interpreter.Depth()
	debug := stylusParams.debugMode != 0
	data, msg, err := status.toResult(output.intoBytes(), debug)
	if status == userFailure && debug {
		log.Warn("program failure", "err", err, "msg", msg, "program", address, "depth", depth)
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
func setBytes32Impl(api usize, key, value bytes32, cost *u64, errVec *rustBytes) apiStatus {
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
func contractCallImpl(api usize, contract bytes20, data *rustSlice, evmGas *u64, value bytes32, len *u32) apiStatus {
	closures := getApi(api)
	ret_len, cost, err := closures.contractCall(contract.toAddress(), data.read(), uint64(*evmGas), value.toBig())
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		return apiFailure
	}
	return apiSuccess
}

//export delegateCallImpl
func delegateCallImpl(api usize, contract bytes20, data *rustSlice, evmGas *u64, len *u32) apiStatus {
	closures := getApi(api)
	ret_len, cost, err := closures.delegateCall(contract.toAddress(), data.read(), uint64(*evmGas))
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		return apiFailure
	}
	return apiSuccess
}

//export staticCallImpl
func staticCallImpl(api usize, contract bytes20, data *rustSlice, evmGas *u64, len *u32) apiStatus {
	closures := getApi(api)
	ret_len, cost, err := closures.staticCall(contract.toAddress(), data.read(), uint64(*evmGas))
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		return apiFailure
	}
	return apiSuccess
}

//export create1Impl
func create1Impl(api usize, code *rustBytes, endowment bytes32, evmGas *u64, len *u32) apiStatus {
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
func create2Impl(api usize, code *rustBytes, endowment, salt bytes32, evmGas *u64, len *u32) apiStatus {
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
func getReturnDataImpl(api usize, output *rustBytes, offset u32, size u32) {
	closures := getApi(api)
	returnData := closures.getReturnData(uint32(offset), uint32(size))
	output.setBytes(returnData)
}

//export emitLogImpl
func emitLogImpl(api usize, data *rustBytes, topics u32) apiStatus {
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

//export accountCodeImpl
func accountCodeImpl(api usize, output *rustBytes, address bytes20, offset u32, size u32, cost *u64) {
	closures := getApi(api)
	code, gas := closures.accountCode(address.toAddress())
	if int(offset) < len(code) {
		end := int(offset + size)
		if len(code) < end {
			end = len(code)
		}
		output.setBytes(code[offset:end])
	}
	*cost = u64(gas)
}

//export accountCodeHashImpl
func accountCodeHashImpl(api usize, address bytes20, cost *u64) bytes32 {
	closures := getApi(api)
	codehash, gas := closures.accountCodeHash(address.toAddress())
	*cost = u64(gas)
	return hashToBytes32(codehash)
}

//export accountCodeSizeImpl
func accountCodeSizeImpl(api usize, address bytes20, cost *u64) u32 {
	closures := getApi(api)
	size, gas := closures.accountCodeSize(address.toAddress())
	*cost = u64(gas)
	return u32(size)
}

//export addPagesImpl
func addPagesImpl(api usize, pages u16) u64 {
	closures := getApi(api)
	cost := closures.addPages(uint16(pages))
	return u64(cost)
}

//export captureHostioImpl
func captureHostioImpl(api usize, name *rustSlice, args *rustSlice, outs *rustSlice, startInk, endInk u64) {
	closures := getApi(api)
	closures.captureHostio(string(name.read()), args.read(), outs.read(), uint64(startInk), uint64(endInk))
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

func (slice *rustSlice) read() []byte {
	return arbutil.PointerToSlice((*byte)(slice.ptr), int(slice.len))
}

func (vec *rustBytes) read() []byte {
	return arbutil.PointerToSlice((*byte)(vec.ptr), int(vec.len))
}

func (vec *rustBytes) intoBytes() []byte {
	slice := vec.read()
	vec.drop()
	return slice
}

func (vec *rustBytes) drop() {
	C.stylus_drop_vec(*vec)
}

func (vec *rustBytes) setString(data string) {
	vec.setBytes([]byte(data))
}

func (vec *rustBytes) setBytes(data []byte) {
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
		ink_price: u32(params.inkPrice.ToUint32()),
	}
	return C.StylusConfig{
		version:   u16(params.version),
		max_depth: u32(params.maxDepth),
		pricing:   pricing,
	}
}

func (data *evmData) encode() C.EvmData {
	return C.EvmData{
		block_basefee:    hashToBytes32(data.blockBasefee),
		chainid:          u64(data.chainId),
		block_coinbase:   addressToBytes20(data.blockCoinbase),
		block_gas_limit:  u64(data.blockGasLimit),
		block_number:     u64(data.blockNumber),
		block_timestamp:  u64(data.blockTimestamp),
		contract_address: addressToBytes20(data.contractAddress),
		msg_sender:       addressToBytes20(data.msgSender),
		msg_value:        hashToBytes32(data.msgValue),
		tx_gas_price:     hashToBytes32(data.txGasPrice),
		tx_origin:        addressToBytes20(data.txOrigin),
		reentrant:        u32(data.reentrant),
		return_data_len:  0,
		tracing:          cbool(data.tracing),
	}
}
