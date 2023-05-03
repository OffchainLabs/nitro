// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build !js
// +build !js

package programs

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#cgo LDFLAGS: ${SRCDIR}/../../target/lib/libstylus.a -ldl -lm
#include "arbitrator.h"

typedef uint32_t u32;
typedef uint64_t u64;
typedef size_t usize;

Bytes32 addressBalanceImpl(usize api, Bytes20 address, u64 * cost);
Bytes32 addressBalanceWrap(usize api, Bytes20 address, u64 * cost) {
    return addressBalanceImpl(api, address, cost);
}

Bytes32 addressCodeHashImpl(usize api, Bytes20 address, u64 * cost);
Bytes32 addressCodeHashWrap(usize api, Bytes20 address, u64 * cost) {
    return addressCodeHashImpl(api, address, cost);
}

Bytes32 blockHashImpl(usize api, Bytes32 block, u64 * cost);
Bytes32 blockHashWrap(usize api, Bytes32 block, u64 * cost) {
    return blockHashImpl(api, block, cost);
}

Bytes32 getBytes32Impl(usize api, Bytes32 key, u64 * cost);
Bytes32 getBytes32Wrap(usize api, Bytes32 key, u64 * cost) {
    return getBytes32Impl(api, key, cost);
}

EvmApiStatus setBytes32Impl(usize api, Bytes32 key, Bytes32 value, u64 * cost, RustVec * error);
EvmApiStatus setBytes32Wrap(usize api, Bytes32 key, Bytes32 value, u64 * cost, RustVec * error) {
    return setBytes32Impl(api, key, value, cost, error);
}

EvmApiStatus contractCallImpl(usize api, Bytes20 contract, RustVec * calldata, u64 * gas, Bytes32 value, u32 * len);
EvmApiStatus contractCallWrap(usize api, Bytes20 contract, RustVec * calldata, u64 * gas, Bytes32 value, u32 * len) {
    return contractCallImpl(api, contract, calldata, gas, value, len);
}

EvmApiStatus delegateCallImpl(usize api, Bytes20 contract, RustVec * calldata, u64 * gas, u32 * len);
EvmApiStatus delegateCallWrap(usize api, Bytes20 contract, RustVec * calldata, u64 * gas, u32 * len) {
    return delegateCallImpl(api, contract, calldata, gas, len);
}

EvmApiStatus staticCallImpl(usize api, Bytes20 contract, RustVec * calldata, u64 * gas, u32 * len);
EvmApiStatus staticCallWrap(usize api, Bytes20 contract, RustVec * calldata, u64 * gas, u32 * len) {
    return staticCallImpl(api, contract, calldata, gas, len);
}

EvmApiStatus create1Impl(usize api, RustVec * code, Bytes32 endowment, u64 * gas, u32 * len);
EvmApiStatus create1Wrap(usize api, RustVec * code, Bytes32 endowment, u64 * gas, u32 * len) {
    return create1Impl(api, code, endowment, gas, len);
}

EvmApiStatus create2Impl(usize api, RustVec * code, Bytes32 endowment, Bytes32 salt, u64 * gas, u32 * len);
EvmApiStatus create2Wrap(usize api, RustVec * code, Bytes32 endowment, Bytes32 salt, u64 * gas, u32 * len) {
    return create2Impl(api, code, endowment, salt, gas, len);
}

void getReturnDataImpl(usize api, RustVec * data);
void getReturnDataWrap(usize api, RustVec * data) {
    return getReturnDataImpl(api, data);
}

EvmApiStatus emitLogImpl(usize api, RustVec * data, usize topics);
EvmApiStatus emitLogWrap(usize api, RustVec * data, usize topics) {
    return emitLogImpl(api, data, topics);
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
var apiIds uintptr // atomic

func newApi(
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	scope *vm.ScopeContext,
) (C.GoEvmApi, usize) {
	closures := newApiClosures(interpreter, tracingInfo, scope)
	apiId := atomic.AddUintptr(&apiIds, 1)
	apiClosures.Store(apiId, closures)
	id := usize(apiId)
	return C.GoEvmApi{
		address_balance:   (*[0]byte)(C.addressBalanceWrap),
		address_code_hash: (*[0]byte)(C.addressCodeHashWrap),
		block_hash:        (*[0]byte)(C.blockHashWrap),
		get_bytes32:       (*[0]byte)(C.getBytes32Wrap),
		set_bytes32:       (*[0]byte)(C.setBytes32Wrap),
		contract_call:     (*[0]byte)(C.contractCallWrap),
		delegate_call:     (*[0]byte)(C.delegateCallWrap),
		static_call:       (*[0]byte)(C.staticCallWrap),
		create1:           (*[0]byte)(C.create1Wrap),
		create2:           (*[0]byte)(C.create2Wrap),
		get_return_data:   (*[0]byte)(C.getReturnDataWrap),
		emit_log:          (*[0]byte)(C.emitLogWrap),
		id:                id,
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
