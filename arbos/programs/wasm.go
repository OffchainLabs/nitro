// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build js
// +build js

package programs

import (
	"errors"

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
type rustStartPages byte

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
func rustStartPagesImpl(need, open, ever u16) *rustStartPages
func rustEvmDataImpl(
	blockBasefee *hash,
	blockChainId *hash,
	blockCoinbase *addr,
	blockDifficulty *hash,
	blockGasLimit u64,
	blockNumber *hash,
	blockTimestamp u64,
	contractAddress *addr,
	msgSender *addr,
	msgValue *hash,
	txGasPrice *hash,
	txOrigin *addr,
	startPages *rustStartPages,
) *rustEvmData

func compileUserWasm(db vm.StateDB, program addr, wasm []byte, version u32, debug bool) (u16, error) {
	debugMode := arbmath.BoolToUint32(debug)
	_, footprint, err := compileMachine(db, program, wasm, version, debugMode)
	println("FOOTPRINT", footprint)
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
) ([]byte, error) {
	wasm, err := getWasm(db, program.address)
	if err != nil {
		log.Crit("failed to get wasm", "program", program, "err", err)
	}
	machine, _, err := compileMachine(db, program.address, wasm, params.version, params.debugMode)
	if err != nil {
		log.Crit("failed to create machine", "program", program, "err", err)
	}

	root := db.NoncanonicalProgramHash(program.address, params.version)
	evmApi := newApi(interpreter, tracingInfo, scope)
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

func compileMachine(db vm.StateDB, program addr, wasm []byte, version, debugMode u32) (*rustMachine, u16, error) {
	open, _ := db.GetStylusPages()
	pageLimit := arbmath.SaturatingUSub(initialMachinePageLimit, *open)

	machine, footprint, err := compileUserWasmRustImpl(wasm, version, debugMode, pageLimit)
	if err != nil {
		return nil, footprint, errors.New(string(err.intoSlice()))
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
		&d.blockDifficulty,
		u64(d.blockGasLimit),
		&d.blockNumber,
		u64(d.blockTimestamp),
		&d.contractAddress,
		&d.msgSender,
		&d.msgValue,
		&d.txGasPrice,
		&d.txOrigin,
		d.startPages.encode(),
	)
}

func (d *startPages) encode() *rustStartPages {
	return rustStartPagesImpl(d.need, d.open, d.ever)
}
