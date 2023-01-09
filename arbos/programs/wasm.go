// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build js
// +build js

package programs

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
)

type addr = common.Address

// rust types
type u8 = uint8
type u32 = uint32
type u64 = uint64
type usize = uintptr

// opaque types
type rustVec byte
type rustParams byte
type rustMachine byte

func compileUserWasmRustImpl(wasm []byte, params *rustParams) (status userStatus, machine *rustMachine, err *rustVec)
func callUserWasmRustImpl(machine *rustMachine, calldata []byte, params *rustParams, gas *u64) (status userStatus, out *rustVec)
func readRustVecImpl(vec *rustVec) (ptr *byte, len usize)
func freeRustVecImpl(vec *rustVec)
func rustParamsImpl(version, maxDepth, heapBound u32, wasmGasPrice, hostioCost u64) *rustParams

func compileUserWasm(db vm.StateDB, program addr, wasm []byte, params *goParams) error {
	_, err := compileMachine(db, program, wasm, params)
	return err
}

func callUserWasm(db vm.StateDB, program addr, calldata []byte, gas *uint64, params *goParams) ([]byte, error) {
	wasm, err := getWasm(db, program)
	if err != nil {
		log.Crit("failed to get wasm", "program", program, "err", err)
	}
	machine, err := compileMachine(db, program, wasm, params)
	if err != nil {
		log.Crit("failed to create machine", "program", program, "err", err)
	}
	return machine.call(calldata, params, gas)
}

func compileMachine(db vm.StateDB, program addr, wasm []byte, params *goParams) (*rustMachine, error) {
	status, machine, err := compileUserWasmRustImpl(wasm, params.encode())
	if status != userSuccess {
		return nil, errors.New(string(err.read()))
	}
	return machine, nil
}

func (m *rustMachine) call(calldata []byte, params *goParams, gas *u64) ([]byte, error) {
	status, output := callUserWasmRustImpl(m, calldata, params.encode(), gas)
	return status.output(output.read())
}

func (vec *rustVec) read() []byte {
	ptr, len := readRustVecImpl(vec)
	output := arbutil.PointerToSlice(ptr, int(len))
	freeRustVecImpl(vec)
	return output
}

func (p *goParams) encode() *rustParams {
	return rustParamsImpl(p.version, p.maxDepth, p.heapBound, p.wasmGasPrice, p.hostioCost)
}
