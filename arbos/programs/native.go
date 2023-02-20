// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build !js
// +build !js

package programs

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#cgo LDFLAGS: ${SRCDIR}/../../target/lib/libstylus.a -ldl -lm
#include "arbitrator.h"

Bytes32  getBytes32WrapperC(size_t api, Bytes32 key, uint64_t * cost);
uint64_t setBytes32WrapperC(size_t api, Bytes32 key, Bytes32 value);
*/
import "C"
import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
)

type u8 = C.uint8_t
type u32 = C.uint32_t
type u64 = C.uint64_t
type usize = C.size_t
type bytes32 = C.Bytes32

func compileUserWasm(db vm.StateDB, program common.Address, wasm []byte, version uint32) error {
	output := rustVec()
	status := userStatus(C.stylus_compile(
		goSlice(wasm),
		u32(version),
		output,
	))
	result, err := status.output(output.read())
	if err == nil {
		db.SetCompiledWasmCode(program, result, version)
	}
	return err
}

func callUserWasm(
	db vm.StateDB,
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	program common.Address,
	calldata []byte,
	gas *uint64,
	params *goParams,
) ([]byte, error) {
	if db, ok := db.(*state.StateDB); ok {
		db.RecordProgram(program, params.version)
	}

	module := db.GetCompiledWasmCode(program, params.version)

	getBytes32 := func(key common.Hash) (common.Hash, uint64) {
		if tracingInfo != nil {
			tracingInfo.RecordStorageGet(key)
		}
		cost := vm.WasmStateLoadCost(db, program, key)
		return db.GetState(program, key), cost
	}
	setBytes32 := func(key, value common.Hash) (uint64, error) {
		if tracingInfo != nil {
			tracingInfo.RecordStorageSet(key, value)
		}
		if interpreter.ReadOnly() {
			return 0, vm.ErrWriteProtection
		}
		cost := vm.WasmStateStoreCost(db, program, key, value)
		db.SetState(program, key, value)
		return cost, nil
	}

	output := rustVec()
	status := userStatus(C.stylus_call(
		goSlice(module),
		goSlice(calldata),
		params.encode(),
		newAPI(getBytes32, setBytes32),
		output,
		(*u64)(gas),
	))
	data, err := status.output(output.read())
	if status == userFailure {
		log.Debug("program failure", "err", string(data), "program", program)
	}
	return data, err
}

//export getBytes32API
func getBytes32API(api usize, key bytes32, cost *uint64) bytes32 {
	closure, err := getAPI(api)
	if err != nil {
		log.Error(err.Error())
		return bytes32{}
	}
	value, gas := closure.getBytes32(key.toHash())
	*cost = gas
	return hashToBytes32(value)
}

//export setBytes32API
func setBytes32API(api usize, key, value bytes32, cost *uint64) usize {
	closure, err := getAPI(api)
	if err != nil {
		log.Error(err.Error())
		return 1
	}
	gas, err := closure.setBytes32(key.toHash(), value.toHash())
	if err != nil {
		return 1
	}
	*cost = gas
	return 0
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

func rustVec() C.RustVec {
	var ptr *u8
	var len usize
	var cap usize
	return C.RustVec{
		ptr: (**u8)(&ptr),
		len: (*usize)(&len),
		cap: (*usize)(&cap),
	}
}

func (vec C.RustVec) read() []byte {
	slice := arbutil.PointerToSlice((*byte)(*vec.ptr), int(*vec.len))
	C.stylus_free(vec)
	return slice
}

func goSlice(slice []byte) C.GoSliceData {
	return C.GoSliceData{
		ptr: (*u8)(arbutil.SliceToPointer(slice)),
		len: usize(len(slice)),
	}
}

func (params *goParams) encode() C.GoParams {
	return C.GoParams{
		version:        u32(params.version),
		max_depth:      u32(params.maxDepth),
		wasm_gas_price: u64(params.wasmGasPrice),
		hostio_cost:    u64(params.hostioCost),
	}
}
