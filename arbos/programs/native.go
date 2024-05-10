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
	"github.com/ethereum/go-ethereum/core/rawdb"
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
	codehash common.Hash,
	wasm []byte,
	page_limit uint16,
	version uint16,
	debug bool,
	burner burn.Burner,
) (*activationInfo, error) {
	info, asm, module, err := activateProgramInternal(db, program, codehash, wasm, page_limit, version, debug, burner.GasLeft())
	if err != nil {
		return nil, err
	}
	db.ActivateWasm(info.moduleHash, asm, module)
	return info, nil
}

func activateProgramInternal(
	db vm.StateDB,
	program common.Address,
	codehash common.Hash,
	wasm []byte,
	page_limit uint16,
	version uint16,
	debug bool,
	gasLeft *uint64,
) (*activationInfo, []byte, []byte, error) {
	output := &rustBytes{}
	asmLen := usize(0)
	moduleHash := &bytes32{}
	stylusData := &C.StylusData{}
	codeHash := hashToBytes32(codehash)

	status := userStatus(C.stylus_activate(
		goSlice(wasm),
		u16(page_limit),
		u16(version),
		cbool(debug),
		output,
		&asmLen,
		&codeHash,
		moduleHash,
		stylusData,
		(*u64)(gasLeft),
	))

	data, msg, err := status.toResult(output.intoBytes(), debug)
	if err != nil {
		if debug {
			log.Warn("activation failed", "err", err, "msg", msg, "program", program)
		}
		if errors.Is(err, vm.ErrExecutionReverted) {
			return nil, nil, nil, fmt.Errorf("%w: %s", ErrProgramActivation, msg)
		}
		return nil, nil, nil, err
	}

	hash := moduleHash.toHash()
	split := int(asmLen)
	asm := data[:split]
	module := data[split:]

	info := &activationInfo{
		moduleHash:    hash,
		initGas:       uint16(stylusData.init_cost),
		cachedInitGas: uint16(stylusData.cached_init_cost),
		asmEstimate:   uint32(stylusData.asm_estimate),
		footprint:     uint16(stylusData.footprint),
	}
	return info, asm, module, err
}

func getLocalAsm(statedb vm.StateDB, moduleHash common.Hash, address common.Address, pagelimit uint16, time uint64, debugMode bool, program Program) ([]byte, error) {
	localAsm, err := statedb.TryGetActivatedAsm(moduleHash)
	if err == nil && len(localAsm) > 0 {
		return localAsm, nil
	}

	codeHash := statedb.GetCodeHash(address)

	wasm, err := getWasm(statedb, address)
	if err != nil {
		log.Error("Failed to reactivate program: getWasm", "address", address, "expected moduleHash", moduleHash, "err", err)
		return nil, fmt.Errorf("failed to reactivate program address: %v err: %w", address, err)
	}

	unlimitedGas := uint64(0xffffffffffff)
	// we know program is activated, so it must be in correct version and not use too much memory
	info, asm, module, err := activateProgramInternal(statedb, address, codeHash, wasm, pagelimit, program.version, debugMode, &unlimitedGas)
	if err != nil {
		log.Error("failed to reactivate program", "address", address, "expected moduleHash", moduleHash, "err", err)
		return nil, fmt.Errorf("failed to reactivate program address: %v err: %w", address, err)
	}

	if info.moduleHash != moduleHash {
		log.Error("failed to reactivate program", "address", address, "expected moduleHash", moduleHash, "got", info.moduleHash)
		return nil, fmt.Errorf("failed to reactivate program. address: %v, expected ModuleHash: %v", address, moduleHash)
	}

	currentHoursSince := hoursSinceArbitrum(time)
	if currentHoursSince > program.activatedAt {
		// stylus program is active on-chain, and was activated in the past
		// so we store it directly to database
		batch := statedb.Database().WasmStore().NewBatch()
		rawdb.WriteActivation(batch, moduleHash, asm, module)
		if err := batch.Write(); err != nil {
			log.Error("failed writing re-activation to state", "address", address, "err", err)
		}
	} else {
		// program activated recently, possibly in this eth_call
		// store it to statedb. It will be stored to database if statedb is commited
		statedb.ActivateWasm(info.moduleHash, asm, module)
	}
	return asm, nil
}

func callProgram(
	address common.Address,
	moduleHash common.Hash,
	localAsm []byte,
	scope *vm.ScopeContext,
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	calldata []byte,
	evmData *evmData,
	stylusParams *goParams,
	memoryModel *MemoryModel,
) ([]byte, error) {
	db := interpreter.Evm().StateDB
	debug := stylusParams.debugMode

	if db, ok := db.(*state.StateDB); ok {
		db.RecordProgram(moduleHash)
	}

	evmApi := newApi(interpreter, tracingInfo, scope, memoryModel)
	defer evmApi.drop()

	output := &rustBytes{}
	status := userStatus(C.stylus_call(
		goSlice(localAsm),
		goSlice(calldata),
		stylusParams.encode(),
		evmApi.cNative,
		evmData.encode(),
		cbool(debug),
		output,
		(*u64)(&scope.Contract.Gas),
	))

	depth := interpreter.Depth()
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
		db.RecordCacheWasm(state.CacheWasm{ModuleHash: module, Version: version, Debug: debug})
	}
}

// Evicts a program in Rust. We write a record so that we can undo on revert, unless we don't need to (e.g. expired)
// For gas estimation and eth_call, we ignore permanent updates and rely on Rust's LRU.
func evictProgram(db vm.StateDB, module common.Hash, version uint16, debug bool, runMode core.MessageRunMode, forever bool) {
	if runMode == core.MessageCommitMode {
		state.EvictWasmRust(module, version, debug)
		if !forever {
			db.RecordEvictWasm(state.EvictWasm{ModuleHash: module, Version: version, Debug: debug})
		}
	}
}

func init() {
	state.CacheWasmRust = func(asm []byte, moduleHash common.Hash, version uint16, debug bool) {
		C.stylus_cache_module(goSlice(asm), hashToBytes32(moduleHash), u16(version), cbool(debug))
	}
	state.EvictWasmRust = func(moduleHash common.Hash, version uint16, debug bool) {
		C.stylus_evict_module(hashToBytes32(moduleHash), u16(version), cbool(debug))
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
		module_hash:      hashToBytes32(data.moduleHash),
		msg_sender:       addressToBytes20(data.msgSender),
		msg_value:        hashToBytes32(data.msgValue),
		tx_gas_price:     hashToBytes32(data.txGasPrice),
		tx_origin:        addressToBytes20(data.txOrigin),
		reentrant:        u32(data.reentrant),
		return_data_len:  0,
		cached:           cbool(data.cached),
		tracing:          cbool(data.tracing),
	}
}
