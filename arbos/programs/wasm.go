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
type u32 = uint32
type u64 = uint64
type usize = uintptr

// opaque types
type rustVec byte
type rustConfig byte
type rustMachine byte
type rustEvmData byte

func compileUserWasmRustImpl(wasm []byte, version, debugMode u32) (machine *rustMachine, err *rustVec)
func callUserWasmRustImpl(
	machine *rustMachine, calldata []byte, params *rustConfig, evmApi []byte,
	evmData *rustEvmData, gas *u64, root *hash,
) (status userStatus, out *rustVec)

func readRustVecLenImpl(vec *rustVec) (len u32)
func rustVecIntoSliceImpl(vec *rustVec, ptr *byte)
func rustConfigImpl(version, maxDepth u32, inkPrice, hostioInk u64, debugMode u32) *rustConfig
func rustEvmDataImpl(
	block_basefee *byte,
	block_chainid *byte,
	block_coinbase *byte,
	block_difficulty *byte,
	block_gas_limit *uint64,
	block_number *byte,
	block_timestamp *byte,
	contract_address *byte,
	msg_sender *byte,
	msg_value *byte,
	tx_gas_price *byte,
	tx_origin *byte,
) *rustEvmData

func compileUserWasm(db vm.StateDB, program addr, wasm []byte, version uint32, debug bool) error {
	debugMode := arbmath.BoolToUint32(debug)
	_, err := compileMachine(db, program, wasm, version, debugMode)
	if err != nil {
		println("COMPILE:", debug, err.Error())
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
	params *goParams,
) ([]byte, error) {
	contract := scope.Contract
	actingAddress := contract.Address() // not necessarily WASM
	program := actingAddress
	if contract.CodeAddr != nil {
		program = *contract.CodeAddr
	}

	wasm, err := getWasm(db, program)
	if err != nil {
		log.Crit("failed to get wasm", "program", program, "err", err)
	}
	machine, err := compileMachine(db, program, wasm, params.version, params.debugMode)
	if err != nil {
		log.Crit("failed to create machine", "program", program, "err", err)
	}

	root := db.NoncanonicalProgramHash(program, params.version)
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

func compileMachine(db vm.StateDB, program addr, wasm []byte, version, debugMode u32) (*rustMachine, error) {
	machine, err := compileUserWasmRustImpl(wasm, version, debugMode)
	if err != nil {
		return nil, errors.New(string(err.intoSlice()))
	}
	return machine, nil
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
		arbutil.SliceToPointer(d.block_basefee.Bytes()),
		arbutil.SliceToPointer(d.block_chainid.Bytes()),
		arbutil.SliceToPointer(d.block_coinbase.Bytes()),
		arbutil.SliceToPointer(d.block_difficulty.Bytes()),
		&d.block_gas_limit,
		arbutil.SliceToPointer(d.block_number.Bytes()),
		arbutil.SliceToPointer(d.block_timestamp.Bytes()),
		arbutil.SliceToPointer(d.contract_address.Bytes()),
		arbutil.SliceToPointer(d.msg_sender.Bytes()),
		arbutil.SliceToPointer(d.msg_value.Bytes()),
		arbutil.SliceToPointer(d.tx_gas_price.Bytes()),
		arbutil.SliceToPointer(d.tx_origin.Bytes()),
	)
}
