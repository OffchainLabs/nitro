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

EvmApiStatus handleReqImpl(usize api, u32 req_type, RustBytes *data, u64 * cost, GoSliceData *out1, GoSliceData *out2);
EvmApiStatus handleReqWrap(usize api, u32 req_type, RustBytes *data, u64 * cost, GoSliceData *out1, GoSliceData *out2) {
    return handleReqImpl(api, req_type, data, cost, out1, out2);
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
) (NativeApi, usize) {
	handler := newApiClosures(interpreter, tracingInfo, scope, memoryModel)
	apiId := atomic.AddUintptr(&apiIds, 1)
	id := usize(apiId)
	api := NativeApi{
		handler: handler,
		cNative: C.NativeRequestHandler{
			handle_request: (*[0]byte)(C.handleReqWrap),
			id:             id,
		},
		// TODO: doesn't seem like pinner needs to be initialized?
	}
	api.pinner.Pin(&api)
	apiObjects.Store(apiId, api)
	return api, id
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

func dropApi(id usize) {
	uid := uintptr(id)
	api := getApi(id)
	api.pinner.Unpin()
	apiObjects.Delete(uid)
}
