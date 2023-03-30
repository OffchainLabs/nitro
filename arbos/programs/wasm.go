// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build js
// +build js

package programs

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
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

func compileUserWasmRustImpl(wasm []byte, version u32) (machine *rustMachine, err *rustVec)
func callUserWasmRustImpl(machine *rustMachine, calldata []byte, params *rustConfig, gas *u64, root *hash) (status userStatus, out *rustVec)
func readRustVecLenImpl(vec *rustVec) (len u32)
func rustVecIntoSliceImpl(vec *rustVec, ptr *byte)
func rustConfigImpl(version, maxDepth u32, wasmGasPrice, hostioCost u64) *rustConfig

func compileUserWasm(db vm.StateDB, program addr, wasm []byte, version uint32, debug bool) error {
	_, err := compileMachine(db, program, wasm, version)
	return err
}

func callUserWasm(
	scope *vm.ScopeContext,
	db vm.StateDB,
	_ *vm.EVMInterpreter,
	_ *util.TracingInfo,
	_ core.Message,
	calldata []byte,
	params *goParams,
) ([]byte, error) {
	program := scope.Contract.Address()
	wasm, err := getWasm(db, program)
	if err != nil {
		log.Crit("failed to get wasm", "program", program, "err", err)
	}
	machine, err := compileMachine(db, program, wasm, params.version)
	if err != nil {
		log.Crit("failed to create machine", "program", program, "err", err)
	}
	root := db.NoncanonicalProgramHash(program, params.version)
	return machine.call(calldata, params, &scope.Contract.Gas, &root)
}

func compileMachine(db vm.StateDB, program addr, wasm []byte, version uint32) (*rustMachine, error) {
	machine, err := compileUserWasmRustImpl(wasm, version)
	if err != nil {
		return nil, errors.New(string(err.intoSlice()))
	}
	return machine, nil
}

func (m *rustMachine) call(calldata []byte, params *goParams, gas *u64, root *hash) ([]byte, error) {
	status, output := callUserWasmRustImpl(m, calldata, params.encode(), gas, root)
	result := output.intoSlice()
	return status.output(result)
}

func (vec *rustVec) intoSlice() []byte {
	len := readRustVecLenImpl(vec)
	slice := make([]byte, len)
	rustVecIntoSliceImpl(vec, arbutil.SliceToPointer(slice))
	return slice
}

func (p *goParams) encode() *rustConfig {
	return rustConfigImpl(p.version, p.maxDepth, p.wasmGasPrice, p.hostioCost)
}
