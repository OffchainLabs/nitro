// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build wasm

package programs

import (
	"errors"
	"unsafe"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type addr = common.Address
type hash = common.Hash

// rust types
type u8 = uint8
type u16 = uint16
type u32 = uint32
type u64 = uint64
type usize = uintptr

// opaque types
type rustVec byte
type rustConfig byte
type rustModule byte
type rustEvmData byte

//go:wasmimport programs activate_v2
func programActivate(
	wasm_ptr unsafe.Pointer,
	wasm_size uint32,
	pages_ptr unsafe.Pointer,
	asm_estimation_ptr unsafe.Pointer,
	init_gas_ptr unsafe.Pointer,
	cached_init_gas_ptr unsafe.Pointer,
	stylusVersion uint32,
	arbosVersion uint64,
	debug uint32,
	codehash unsafe.Pointer,
	module_hash_ptr unsafe.Pointer,
	gas_ptr unsafe.Pointer,
	err_buf unsafe.Pointer,
	err_buf_len uint32,
) uint32

func activateProgram(
	db vm.StateDB,
	program addr,
	codehash common.Hash,
	wasm []byte,
	pageLimit u16,
	stylusVersion u16,
	arbosVersion uint64,
	debug bool,
	burner burn.Burner,
	runCtx *core.MessageRunContext,
) (*activationInfo, error) {
	errBuf := make([]byte, 1024)
	debugMode := arbmath.BoolToUint32(debug)
	moduleHash := common.Hash{}
	gasSupplied := burner.GasLeft()
	gasLeft := burner.GasLeft()
	asmEstimate := uint32(0)
	initGas := uint16(0)
	cachedInitGas := uint16(0)

	footprint := uint16(pageLimit)
	errLen := programActivate(
		arbutil.SliceToUnsafePointer(wasm),
		uint32(len(wasm)),
		unsafe.Pointer(&footprint),
		unsafe.Pointer(&asmEstimate),
		unsafe.Pointer(&initGas),
		unsafe.Pointer(&cachedInitGas),
		uint32(stylusVersion),
		arbosVersion,
		debugMode,
		arbutil.SliceToUnsafePointer(codehash[:]),
		arbutil.SliceToUnsafePointer(moduleHash[:]),
		unsafe.Pointer(&gasLeft),
		arbutil.SliceToUnsafePointer(errBuf),
		uint32(len(errBuf)),
	)
	burner.Burn(multigas.ResourceKindComputation, gasSupplied-gasLeft)
	if errLen != 0 {
		err := errors.New(string(errBuf[:errLen]))
		return nil, err
	}
	return &activationInfo{moduleHash, initGas, cachedInitGas, asmEstimate, footprint}, nil
}

// stub any non-consensus, Rust-side caching updates
func cacheProgram(db vm.StateDB, module common.Hash, program Program, addressForLogging common.Address, code []byte, codeHash common.Hash, params *StylusParams, debug bool, time uint64, runCtx *core.MessageRunContext) {
}
func evictProgram(db vm.StateDB, module common.Hash, version uint16, debug bool, runCtx *core.MessageRunContext, forever bool) {
}

//go:wasmimport programs new_program
func newProgram(
	hashPtr unsafe.Pointer,
	callDataPtr unsafe.Pointer,
	callDataSize uint32,
	configHandler stylusConfigHandler,
	evmHandler evmDataHandler,
	gas uint64,
) uint32

//go:wasmimport programs pop
func popProgram()

//go:wasmimport programs set_response
func setResponse(id uint32, gas uint64, result unsafe.Pointer, result_len uint32, raw_data unsafe.Pointer, raw_data_len uint32)

//go:wasmimport programs get_request
func getRequest(id uint32, reqLen unsafe.Pointer) uint32

//go:wasmimport programs get_request_data
func getRequestData(id uint32, dataPtr unsafe.Pointer)

//go:wasmimport programs start_program
func startProgram(module uint32) uint32

//go:wasmimport programs send_response
func sendResponse(req_id uint32) uint32

func getCompiledProgram(statedb vm.StateDB, moduleHash common.Hash, addressForLogging common.Address, code []byte, codeHash common.Hash, maxWasmSize uint32, pagelimit uint16, time uint64, debugMode bool, program Program, runCtx *core.MessageRunContext) (map[rawdb.WasmTarget][]byte, error) {
	// we need to return asm map with an entry for local target to make checks for local target work
	return map[rawdb.WasmTarget][]byte{rawdb.LocalTarget(): {}}, nil
}

func callProgram(
	address common.Address,
	moduleHash common.Hash,
	_localAsm []byte,
	scope *vm.ScopeContext,
	evm *vm.EVM,
	tracingInfo *util.TracingInfo,
	calldata []byte,
	evmData *EvmData,
	params *ProgParams,
	memoryModel *MemoryModel,
	runCtx *core.MessageRunContext,
) ([]byte, error) {
	reqHandler := newApiClosures(evm, tracingInfo, scope, memoryModel)
	gasLeft, retData, err := CallProgramLoop(moduleHash, calldata, scope.Contract.Gas, evmData, params, reqHandler)
	scope.Contract.Gas = gasLeft
	return retData, err
}

func GetWasmLruCacheMetrics() {}

func CallProgramLoop(
	moduleHash common.Hash,
	calldata []byte,
	gas uint64,
	evmData *EvmData,
	params *ProgParams,
	reqHandler RequestHandler) (uint64, []byte, error) {
	configHandler := params.createHandler()
	dataHandler := evmData.createHandler()
	debug := params.DebugMode

	module := newProgram(
		unsafe.Pointer(&moduleHash[0]),
		arbutil.SliceToUnsafePointer(calldata),
		uint32(len(calldata)),
		configHandler,
		dataHandler,
		gas,
	)
	reqId := startProgram(module)
	for {
		var reqLen uint32
		reqTypeId := getRequest(reqId, unsafe.Pointer(&reqLen))
		reqData := make([]byte, reqLen)
		getRequestData(reqId, arbutil.SliceToUnsafePointer(reqData))
		if reqTypeId < EvmApiMethodReqOffset {
			popProgram()
			status := userStatus(reqTypeId)
			gasLeft := arbmath.BytesToUint(reqData[:8])
			data, msg, err := status.toResult(reqData[8:], debug)
			if status == userFailure && debug {
				log.Warn("program failure", "err", err, "msg", msg, "moduleHash", moduleHash)
			}
			return gasLeft, data, err
		}

		reqType := RequestType(reqTypeId - EvmApiMethodReqOffset)
		result, rawData, cost := reqHandler(reqType, reqData)
		setResponse(
			reqId,
			cost,
			arbutil.SliceToUnsafePointer(result), uint32(len(result)),
			arbutil.SliceToUnsafePointer(rawData), uint32(len(rawData)),
		)
		reqId = sendResponse(reqId)
	}
}
