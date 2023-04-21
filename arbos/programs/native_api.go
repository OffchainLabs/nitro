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

Bytes32 getBytes32Impl(usize api, Bytes32 key, u64 * cost);
Bytes32 getBytes32Wrap(usize api, Bytes32 key, u64 * cost) {
    return getBytes32Impl(api, key, cost);
}

GoApiStatus setBytes32Impl(usize api, Bytes32 key, Bytes32 value, u64 * cost, RustVec * error);
GoApiStatus setBytes32Wrap(usize api, Bytes32 key, Bytes32 value, u64 * cost, RustVec * error) {
    return setBytes32Impl(api, key, value, cost, error);
}

GoApiStatus contractCallImpl(usize api, Bytes20 contract, RustVec * calldata, u64 * gas, Bytes32 value, u32 * len);
GoApiStatus contractCallWrap(usize api, Bytes20 contract, RustVec * calldata, u64 * gas, Bytes32 value, u32 * len) {
    return contractCallImpl(api, contract, calldata, gas, value, len);
}

GoApiStatus delegateCallImpl(usize api, Bytes20 contract, RustVec * calldata, u64 * gas, u32 * len);
GoApiStatus delegateCallWrap(usize api, Bytes20 contract, RustVec * calldata, u64 * gas, u32 * len) {
    return delegateCallImpl(api, contract, calldata, gas, len);
}

GoApiStatus staticCallImpl(usize api, Bytes20 contract, RustVec * calldata, u64 * gas, u32 * len);
GoApiStatus staticCallWrap(usize api, Bytes20 contract, RustVec * calldata, u64 * gas, u32 * len) {
    return staticCallImpl(api, contract, calldata, gas, len);
}

GoApiStatus create1Impl(usize api, RustVec * code, Bytes32 endowment, u64 * gas, u32 * len);
GoApiStatus create1Wrap(usize api, RustVec * code, Bytes32 endowment, u64 * gas, u32 * len) {
    return create1Impl(api, code, endowment, gas, len);
}

GoApiStatus create2Impl(usize api, RustVec * code, Bytes32 endowment, Bytes32 salt, u64 * gas, u32 * len);
GoApiStatus create2Wrap(usize api, RustVec * code, Bytes32 endowment, Bytes32 salt, u64 * gas, u32 * len) {
    return create2Impl(api, code, endowment, salt, gas, len);
}

void getReturnDataImpl(usize api, RustVec * data);
void getReturnDataWrap(usize api, RustVec * data) {
    return getReturnDataImpl(api, data);
}

GoApiStatus emitLogImpl(usize api, RustVec * data, usize topics);
GoApiStatus emitLogWrap(usize api, RustVec * data, usize topics) {
    return emitLogImpl(api, data, topics);
}
*/
import "C"

func wrapGoApi(id usize) (C.GoApi, usize) {
	return C.GoApi{
		get_bytes32:     (*[0]byte)(C.getBytes32Wrap),
		set_bytes32:     (*[0]byte)(C.setBytes32Wrap),
		contract_call:   (*[0]byte)(C.contractCallWrap),
		delegate_call:   (*[0]byte)(C.delegateCallWrap),
		static_call:     (*[0]byte)(C.staticCallWrap),
		create1:         (*[0]byte)(C.create1Wrap),
		create2:         (*[0]byte)(C.create2Wrap),
		get_return_data: (*[0]byte)(C.getReturnDataWrap),
		emit_log:        (*[0]byte)(C.emitLogWrap),
		id:              usize(id),
	}, id
}
