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

EvmApiStatus handleReqImpl(usize api, u32 req_type, RustBytes *data, u64 * cost, RustBytes * output);
EvmApiStatus handleReqWrap(usize api, u32 req_type, RustBytes *data, u64 * cost, RustBytes * output) {
    return handleReqImpl(api, req_type, data, cost, output);
}
*/
import "C"
import (
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos/util"
)

var apiClosures sync.Map
var apiIds uintptr // atomic and sequential

func newApi(
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	scope *vm.ScopeContext,
	memoryModel *MemoryModel,
) (C.NativeRequestHandler, usize) {
	closures := newApiClosures(interpreter, tracingInfo, scope, memoryModel)
	apiId := atomic.AddUintptr(&apiIds, 1)
	apiClosures.Store(apiId, closures)
	id := usize(apiId)
	return C.NativeRequestHandler{
		handle_request: (*[0]byte)(C.handleReqWrap),
		id:             id,
	}, id
}

func getApi(id usize) RequestHandler {
	any, ok := apiClosures.Load(uintptr(id))
	if !ok {
		log.Crit("failed to load stylus Go API", "id", id)
	}
	closures, ok := any.(RequestHandler)
	if !ok {
		log.Crit("wrong type for stylus Go API", "id", id)
	}
	return closures
}

func dropApi(id usize) {
	apiClosures.Delete(uintptr(id))
}
