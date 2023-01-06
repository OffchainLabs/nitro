// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build !js
// +build !js

package programs

//#cgo CFLAGS: -g -Wall
//#cgo LDFLAGS: ${SRCDIR}/../../target/lib/libstylus.a -ldl -lm
//#include <stdint.h>
//
// typedef struct GoParams {
//   uint32_t version;
//   uint32_t max_depth;
//   uint32_t heap_bound;
//   uint64_t wasm_gas_price;
//   uint64_t hostio_cost;
// } GoParams;
//
// typedef struct GoSlice {
//   const uint8_t * ptr;
//   const size_t len;
// } GoSlice;
//
// typedef struct RustVec {
//   uint8_t * const * ptr;
//   size_t * len;
//   size_t * cap;
// } RustVec;
//
// extern uint8_t stylus_compile(GoSlice wasm, GoParams params, RustVec output);
// extern uint8_t stylus_call(GoSlice module, GoSlice calldata, GoParams params, RustVec output, uint64_t * evm_gas);
// extern void    stylus_free(RustVec vec);
//
import "C"
import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
)

type u8 = C.uint8_t
type u32 = C.uint32_t
type u64 = C.uint64_t
type usize = C.size_t

const (
	Success u8 = iota
	Failure
	OutOfGas
)

func compileUserWasm(db vm.StateDB, program common.Address, wasm []byte, params *goParams) error {
	output := rustVec()
	status := C.stylus_compile(
		goSlice(wasm),
		params.encode(),
		output,
	)
	result := output.read()

	if status != Success {
		return errors.New(string(result))
	}
	db.AddUserModule(params.version, program, result)
	return nil
}

func callUserWasm(
	db vm.StateDB, program common.Address, calldata []byte, gas *uint64, params *goParams,
) (uint32, []byte, error) {

	if db, ok := db.(*state.StateDB); ok {
		db.RecordProgram(program)
	}

	module, err := db.GetUserModule(1, program)
	if err != nil {
		log.Crit("machine does not exist")
	}

	output := rustVec()
	status := C.stylus_call(
		goSlice(module),
		goSlice(calldata),
		params.encode(),
		output,
		(*u64)(gas),
	)
	if status == Failure {
		return 0, nil, errors.New(string(output.read()))
	}
	if status == OutOfGas {
		return 0, nil, vm.ErrOutOfGas
	}
	return uint32(status), output.read(), nil
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

func goSlice(slice []byte) C.GoSlice {
	return C.GoSlice{
		ptr: (*u8)(arbutil.SliceToPointer(slice)),
		len: usize(len(slice)),
	}
}

func (params *goParams) encode() C.GoParams {
	return C.GoParams{
		version:        u32(params.version),
		max_depth:      u32(params.max_depth),
		heap_bound:     u32(params.heap_bound),
		wasm_gas_price: u64(params.wasm_gas_price),
		hostio_cost:    u64(params.hostio_cost),
	}
}
