// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build js
// +build js

package programs

import (
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
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
type rustMachine byte
type rustEvmData byte

func compileUserWasmRustImpl(
	wasm []byte, version, debugMode u32, pageLimit u16,
) (machine *rustMachine, footprint u16, err *rustVec)

func callUserWasmRustImpl(
	machine *rustMachine, calldata []byte, params *rustConfig, evmApi []byte,
	evmData *rustEvmData, gas *u64, root *hash,
) (status userStatus, out *rustVec)

func readRustVecLenImpl(vec *rustVec) (len u32)
func rustVecIntoSliceImpl(vec *rustVec, ptr *byte)
func rustConfigImpl(version, maxDepth u32, inkPrice, hostioInk u64, debugMode u32) *rustConfig
func rustEvmDataImpl(
	blockBasefee *hash,
	blockChainId *hash,
	blockCoinbase *addr,
	blockGasLimit u64,
	blockNumber *hash,
	blockTimestamp u64,
	contractAddress *addr,
	msgSender *addr,
	msgValue *hash,
	txGasPrice *hash,
	txOrigin *addr,
) *rustEvmData

func compileUserWasm(db vm.StateDB, program addr, wasm []byte, pageLimit u16, version u32, debug bool) (u16, error) {
	debugMode := arbmath.BoolToUint32(debug)
	_, footprint, err := compileMachine(db, program, wasm, pageLimit, version, debugMode)
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
	params *goParams,
	memoryModel *MemoryModel,
) ([]byte, error) {
	// since the program has previously passed compilation, don't limit memory
	pageLimit := uint16(math.MaxUint16)

	wasm, err := getWasm(db, program.address)
	if err != nil {
		log.Crit("failed to get wasm", "program", program, "err", err)
	}
	machine, _, err := compileMachine(db, program.address, wasm, pageLimit, params.version, params.debugMode)
	if err != nil {
		log.Crit("failed to create machine", "program", program, "err", err)
	}

	root := db.NoncanonicalProgramHash(program.address, params.version)
	evmApi := newApi(interpreter, tracingInfo, scope, memoryModel)
	defer evmApi.drop()

	status, output := callUserWasmRustImpl(
		machine,
		calldata,
		params.encode(),
		evmApi.funcs,
		evmData.encode(),
		&scope.Contract.Gas,
		&root,
	)
	result := output.intoSlice()
	return status.output(result)
}

func compileMachine(
	db vm.StateDB, program addr, wasm []byte, pageLimit u16, version, debugMode u32,
) (*rustMachine, u16, error) {
	machine, footprint, err := compileUserWasmRustImpl(wasm, version, debugMode, pageLimit)
	if err != nil {
		_, err := userFailure.output(err.intoSlice())
		return nil, footprint, err
	}
	return machine, footprint, nil
}

func (vec *rustVec) intoSlice() []byte {
	len := readRustVecLenImpl(vec)
	slice := make([]byte, len)
	rustVecIntoSliceImpl(vec, arbutil.SliceToPointer(slice))
	return slice
}

func (p *goParams) encode() *rustConfig {
	return rustConfigImpl(p.version, p.maxDepth, p.inkPrice, p.hostioInk, p.debugMode)
}

func (d *evmData) encode() *rustEvmData {
	return rustEvmDataImpl(
		&d.blockBasefee,
		&d.blockChainId,
		&d.blockCoinbase,
		u64(d.blockGasLimit),
		&d.blockNumber,
		u64(d.blockTimestamp),
		&d.contractAddress,
		&d.msgSender,
		&d.msgValue,
		&d.txGasPrice,
		&d.txOrigin,
	)
}
