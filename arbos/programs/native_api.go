// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build !wasm
// +build !wasm

package programs

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#cgo LDFLAGS: ${SRCDIR}/../../target/lib/libstylus.a -ldl -lm
#include "arbitrator.h"

typedef uint16_t u16;
typedef uint32_t u32;
typedef uint64_t u64;
typedef size_t usize;

EvmApiStatus handleReqImpl(usize api, u32 req_type, RustSlice *data, u64 *out_cost, GoSliceData *out_result, GoSliceData *out_raw_data);
EvmApiStatus handleReqWrap(usize api, u32 req_type, RustSlice *data, u64 *out_cost, GoSliceData *out_result, GoSliceData *out_raw_data) {
    return handleReqImpl(api, req_type, data, out_cost, out_result, out_raw_data);
}
*/
import "C"
import (
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
)

var apiObjects sync.Map
var apiIds uintptr // atomic and sequential

type NativeApi struct {
	handler RequestHandler
	cNative C.NativeRequestHandler
	pinner  runtime.Pinner
}

func newApi(
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	scope *vm.ScopeContext,
	memoryModel *MemoryModel,
) NativeApi {
	handler := newApiClosures(interpreter, tracingInfo, scope, memoryModel)
	apiId := atomic.AddUintptr(&apiIds, 1)
	id := usize(apiId)
	api := NativeApi{
		handler: handler,
		cNative: C.NativeRequestHandler{
			handle_request_fptr: (*[0]byte)(C.handleReqWrap),
			id:                  id,
		},
		pinner: runtime.Pinner{},
	}
	api.pinner.Pin(&api)
	apiObjects.Store(apiId, api)
	return api
}

func getApi(id usize) NativeApi {
	any, ok := apiObjects.Load(uintptr(id))
	if !ok {
		log.Crit("failed to load stylus Go API", "id", id)
	}
	api, ok := any.(NativeApi)
	if !ok {
		log.Crit("wrong type for stylus Go API", "id", id)
	}
	return api
}

// Free the API object, and any saved request payloads.
func (api *NativeApi) drop() {
	api.pinner.Unpin()
	apiObjects.Delete(uintptr(api.cNative.id))
}

// Pins a slice until program exit during the call to `drop`.
func (api *NativeApi) pinAndRef(data []byte, goSlice *C.GoSliceData) {
	if len(data) > 0 {
		dataPointer := arbutil.SliceToPointer(data)
		api.pinner.Pin(dataPointer)
		goSlice.ptr = (*u8)(dataPointer)
	} else {
		goSlice.ptr = (*u8)(nil)
	}
	goSlice.len = usize(len(data))
}
