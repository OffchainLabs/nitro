// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build !wasm
// +build !wasm

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
) (*activationInfo, error) {
	output := &rustBytes{}
	asmLen := usize(0)
	moduleHash := &bytes32{}
	stylusData := &C.StylusData{}

	status := userStatus(C.stylus_activate(
		goSlice(wasm),
		u16(page_limit),
		u16(version),
		cbool(debug),
		output,
		&asmLen,
		moduleHash,
		stylusData,
		(*u64)(burner.GasLeft()),
	))

	data, msg, err := status.toResult(output.intoBytes(), debug)
	if err != nil {
		if debug {
			log.Warn("activation failed", "err", err, "msg", msg, "program", program)
		}
		if errors.Is(err, vm.ErrExecutionReverted) {
			return nil, fmt.Errorf("%w: %s", ErrProgramActivation, msg)
		}
		return nil, err
	}

	hash := moduleHash.toHash()
	split := int(asmLen)
	asm := data[:split]
	module := data[split:]

	info := &activationInfo{
		moduleHash:  hash,
		initGas:     uint32(stylusData.init_gas),
		asmEstimate: uint32(stylusData.asm_estimate),
		footprint:   uint16(stylusData.footprint),
	}
	db.ActivateWasm(hash, asm, module)
	return info, err
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

//export handleReqImpl
func handleReqImpl(api usize, req_type u32, data *rustBytes, costPtr *u64, output *rustBytes) apiStatus {
	closure := getApi(api)
	reqData := data.read()
	reqType := RequestType(req_type - 0x10000000)
	res, cost := closure(reqType, reqData)
	*costPtr = u64(cost)
	output.setBytes(res)
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
