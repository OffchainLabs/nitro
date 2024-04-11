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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
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
		moduleHash:    hash,
		initGas:       uint16(stylusData.init_gas),
		cachedInitGas: uint16(stylusData.cached_init_gas),
		asmEstimate:   uint32(stylusData.asm_estimate),
		footprint:     uint16(stylusData.footprint),
	}
	db.ActivateWasm(hash, asm, module)
	return info, err
}

func callProgram(
	address common.Address,
	moduleHash common.Hash,
	scope *vm.ScopeContext,
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	calldata []byte,
	evmData *evmData,
	stylusParams *goParams,
	memoryModel *MemoryModel,
) ([]byte, error) {
	db := interpreter.Evm().StateDB
	asm := db.GetActivatedAsm(moduleHash)

	if db, ok := db.(*state.StateDB); ok {
		db.RecordProgram(moduleHash)
	}

	evmApi := newApi(interpreter, tracingInfo, scope, memoryModel)
	defer evmApi.drop()

	output := &rustBytes{}
	status := userStatus(C.stylus_call(
		goSlice(asm),
		goSlice(calldata),
		stylusParams.encode(),
		evmApi.cNative,
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

//export handleReqImpl
func handleReqImpl(apiId usize, req_type u32, data *rustSlice, costPtr *u64, out_response *C.GoSliceData, out_raw_data *C.GoSliceData) {
	api := getApi(apiId)
	reqData := data.read()
	reqType := RequestType(req_type - EvmApiMethodReqOffset)
	response, raw_data, cost := api.handler(reqType, reqData)
	*costPtr = u64(cost)
	api.pinAndRef(response, out_response)
	api.pinAndRef(raw_data, out_raw_data)
}

// Caches a program in Rust. We write a record so that we can undo on revert.
// For gas estimation and eth_call, we ignore permanent updates and rely on Rust's LRU.
func cacheProgram(db vm.StateDB, module common.Hash, version uint16, debug bool, runMode core.MessageRunMode) {
	if runMode == core.MessageCommitMode {
		asm := db.GetActivatedAsm(module)
		state.CacheWasmRust(asm, module, version, debug)
		db.RecordCacheWasm(state.CacheWasm{ModuleHash: module})
	}
}

// Evicts a program in Rust. We write a record so that we can undo on revert, unless we don't need to (e.g. expired)
// For gas estimation and eth_call, we ignore permanent updates and rely on Rust's LRU.
func evictProgram(db vm.StateDB, module common.Hash, version uint16, debug bool, runMode core.MessageRunMode, forever bool) {
	if runMode == core.MessageCommitMode {
		state.EvictWasmRust(module)
		if !forever {
			db.RecordEvictWasm(state.EvictWasm{
				ModuleHash: module,
				Version:    version,
				Debug:      debug,
			})
		}
	}
}

func init() {
	state.CacheWasmRust = func(asm []byte, moduleHash common.Hash, version uint16, debug bool) {
		C.stylus_cache_module(goSlice(asm), hashToBytes32(moduleHash), u16(version), cbool(debug))
	}
	state.EvictWasmRust = func(moduleHash common.Hash) {
		C.stylus_evict_module(hashToBytes32(moduleHash))
	}
}

func (value bytes32) toHash() common.Hash {
	hash := common.Hash{}
	for index, b := range value.bytes {
		hash[index] = byte(b)
	}
	return hash
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
