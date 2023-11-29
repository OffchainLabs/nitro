// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build !js
// +build !js

package programs

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#cgo LDFLAGS: ${SRCDIR}/../../target/lib/libstylus.a -ldl -lm
#include "arbitrator.h"

typedef uint16_t u16;
typedef uint32_t u32;
typedef uint64_t u64;
typedef size_t usize;

Bytes32 getBytes32Impl(usize api, Bytes32 key, u64 * cost);
Bytes32 getBytes32Wrap(usize api, Bytes32 key, u64 * cost) {
    return getBytes32Impl(api, key, cost);
}

EvmApiStatus setBytes32Impl(usize api, Bytes32 key, Bytes32 value, u64 * cost, RustBytes * error);
EvmApiStatus setBytes32Wrap(usize api, Bytes32 key, Bytes32 value, u64 * cost, RustBytes * error) {
    return setBytes32Impl(api, key, value, cost, error);
}

EvmApiStatus contractCallImpl(usize api, Bytes20 contract, RustSlice * calldata, u64 * gas, Bytes32 value, u32 * len);
EvmApiStatus contractCallWrap(usize api, Bytes20 contract, RustSlice * calldata, u64 * gas, Bytes32 value, u32 * len) {
    return contractCallImpl(api, contract, calldata, gas, value, len);
}

EvmApiStatus delegateCallImpl(usize api, Bytes20 contract, RustSlice * calldata, u64 * gas, u32 * len);
EvmApiStatus delegateCallWrap(usize api, Bytes20 contract, RustSlice * calldata, u64 * gas, u32 * len) {
    return delegateCallImpl(api, contract, calldata, gas, len);
}

EvmApiStatus staticCallImpl(usize api, Bytes20 contract, RustSlice * calldata, u64 * gas, u32 * len);
EvmApiStatus staticCallWrap(usize api, Bytes20 contract, RustSlice * calldata, u64 * gas, u32 * len) {
    return staticCallImpl(api, contract, calldata, gas, len);
}

EvmApiStatus create1Impl(usize api, RustBytes * code, Bytes32 endowment, u64 * gas, u32 * len);
EvmApiStatus create1Wrap(usize api, RustBytes * code, Bytes32 endowment, u64 * gas, u32 * len) {
    return create1Impl(api, code, endowment, gas, len);
}

EvmApiStatus create2Impl(usize api, RustBytes * code, Bytes32 endowment, Bytes32 salt, u64 * gas, u32 * len);
EvmApiStatus create2Wrap(usize api, RustBytes * code, Bytes32 endowment, Bytes32 salt, u64 * gas, u32 * len) {
    return create2Impl(api, code, endowment, salt, gas, len);
}

void getReturnDataImpl(usize api, RustBytes * data, u32 offset, u32 size);
void getReturnDataWrap(usize api, RustBytes * data, u32 offset, u32 size) {
    return getReturnDataImpl(api, data, offset, size);
}

EvmApiStatus emitLogImpl(usize api, RustBytes * data, usize topics);
EvmApiStatus emitLogWrap(usize api, RustBytes * data, usize topics) {
    return emitLogImpl(api, data, topics);
}

Bytes32 accountBalanceImpl(usize api, Bytes20 address, u64 * cost);
Bytes32 accountBalanceWrap(usize api, Bytes20 address, u64 * cost) {
    return accountBalanceImpl(api, address, cost);
}

Bytes32 accountCodeHashImpl(usize api, Bytes20 address, u64 * cost);
Bytes32 accountCodeHashWrap(usize api, Bytes20 address, u64 * cost) {
    return accountCodeHashImpl(api, address, cost);
}

u64 addPagesImpl(usize api, u16 pages);
u64 addPagesWrap(usize api, u16 pages) {
    return addPagesImpl(api, pages);
}

void captureHostioImpl(usize api, RustSlice * name, RustSlice * data, RustSlice * outs, u64 startInk, u64 endInk);
void captureHostioWrap(usize api, RustSlice * name, RustSlice * data, RustSlice * outs, u64 startInk, u64 endInk) {
    return captureHostioImpl(api, name, data, outs, startInk, endInk);
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
) (C.GoEvmApi, usize) {
	closures := newApiClosures(interpreter, tracingInfo, scope, memoryModel)
	apiId := atomic.AddUintptr(&apiIds, 1)
	apiClosures.Store(apiId, closures)
	id := usize(apiId)
	return C.GoEvmApi{
		get_bytes32:      (*[0]byte)(C.getBytes32Wrap),
		set_bytes32:      (*[0]byte)(C.setBytes32Wrap),
		contract_call:    (*[0]byte)(C.contractCallWrap),
		delegate_call:    (*[0]byte)(C.delegateCallWrap),
		static_call:      (*[0]byte)(C.staticCallWrap),
		create1:          (*[0]byte)(C.create1Wrap),
		create2:          (*[0]byte)(C.create2Wrap),
		get_return_data:  (*[0]byte)(C.getReturnDataWrap),
		emit_log:         (*[0]byte)(C.emitLogWrap),
		account_balance:  (*[0]byte)(C.accountBalanceWrap),
		account_codehash: (*[0]byte)(C.accountCodeHashWrap),
		add_pages:        (*[0]byte)(C.addPagesWrap),
		capture_hostio:   (*[0]byte)(C.captureHostioWrap),
		id:               id,
	}, id
}

func getApi(id usize) *goClosures {
	any, ok := apiClosures.Load(uintptr(id))
	if !ok {
		log.Crit("failed to load stylus Go API", "id", id)
	}
	closures, ok := any.(*goClosures)
	if !ok {
		log.Crit("wrong type for stylus Go API", "id", id)
	}
	return closures
}

func dropApi(id usize) {
	apiClosures.Delete(uintptr(id))
}
