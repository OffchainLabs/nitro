// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build wasm
// +build wasm

package programs

import (
	"encoding/binary"
	"errors"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
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

//go:wasmimport programs activate
func programActivate(
	wasm_ptr unsafe.Pointer,
	wasm_size uint32,
	pages_ptr unsafe.Pointer,
	asm_estimation_ptr unsafe.Pointer,
	init_gas_ptr unsafe.Pointer,
	version uint32,
	debug uint32,
	module_hash_ptr unsafe.Pointer,
	gas_ptr unsafe.Pointer,
	err_buf unsafe.Pointer,
	err_buf_len uint32,
) uint32

func activateProgram(
	db vm.StateDB,
	program addr,
	wasm []byte,
	pageLimit u16,
	version u16,
	debug bool,
	burner burn.Burner,
) (*activationInfo, error) {
	errBuf := make([]byte, 1024)
	debugMode := arbmath.BoolToUint32(debug)
	moduleHash := common.Hash{}
	gasPtr := burner.GasLeft()
	asmEstimate := uint32(0)
	initGas := uint32(0)

	footprint := uint16(pageLimit)
	errLen := programActivate(
		arbutil.SliceToUnsafePointer(wasm),
		uint32(len(wasm)),
		unsafe.Pointer(&footprint),
		unsafe.Pointer(&asmEstimate),
		unsafe.Pointer(&initGas),
		uint32(version),
		debugMode,
		arbutil.SliceToUnsafePointer(moduleHash[:]),
		unsafe.Pointer(gasPtr),
		arbutil.SliceToUnsafePointer(errBuf),
		uint32(len(errBuf)),
	)
	if errLen != 0 {
		err := errors.New(string(errBuf[:errLen]))
		return nil, err
	}
	return &activationInfo{moduleHash, initGas, asmEstimate, footprint}, nil
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
func setResponse(id uint32, gas uint64, response unsafe.Pointer, response_len uint32)

//go:wasmimport programs get_request
func getRequest(id uint32, reqLen unsafe.Pointer) uint32

//go:wasmimport programs get_request_data
func getRequestData(id uint32, dataPtr unsafe.Pointer)

//go:wasmimport programs start_program
func startProgram(module uint32) uint32

//go:wasmimport programs send_response
func sendResponse(req_id uint32) uint32

func callProgram(
	address common.Address,
	moduleHash common.Hash,
	scope *vm.ScopeContext,
	db vm.StateDB,
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	calldata []byte,
	evmData *evmData,
	params *goParams,
	memoryModel *MemoryModel,
) ([]byte, error) {
	// debug := arbmath.UintToBool(params.debugMode)
	reqHandler := newApiClosures(interpreter, tracingInfo, scope, memoryModel)

	configHandler := params.createHandler()
	dataHandler := evmData.createHandler()

	module := newProgram(
		unsafe.Pointer(&moduleHash[0]),
		arbutil.SliceToUnsafePointer(calldata),
		uint32(len(calldata)),
		configHandler,
		dataHandler,
		scope.Contract.Gas,
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
			gasLeft := binary.BigEndian.Uint64(reqData[:8])
			scope.Contract.Gas = gasLeft
			data, msg, err := status.toResult(reqData[8:], params.debugMode != 0)
			if status == userFailure && params.debugMode != 0 {
				log.Warn("program failure", "err", err, "msg", msg, "program", address)
			}
			return data, err
		}
		reqType := RequestType(reqTypeId - EvmApiMethodReqOffset)
		response, cost := reqHandler(reqType, reqData)
		setResponse(reqId, cost, arbutil.SliceToUnsafePointer(response), uint32(len(response)))
		reqId = sendResponse(reqId)
	}
}
