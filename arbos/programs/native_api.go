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

GoApiStatus callContractImpl(usize api, Bytes20 contract, RustVec * calldata, u64 * gas, Bytes32 value, u32 * len);
GoApiStatus callContractWrap(usize api, Bytes20 contract, RustVec * calldata, u64 * gas, Bytes32 value, u32 * len) {
    return callContractImpl(api, contract, calldata, gas, value, len);
}

void getReturnDataImpl(usize api, RustVec * data);
void getReturnDataWrap(usize api, RustVec * data) {
    return getReturnDataImpl(api, data);
}
*/
import "C"
import (
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/colors"
)

var apiClosures sync.Map
var apiIds int64 // atomic

type getBytes32Type func(key common.Hash) (value common.Hash, cost uint64)
type setBytes32Type func(key, value common.Hash) (cost uint64, err error)
type callContractType func(
	contract common.Address, input []byte, gas uint64, value *big.Int) (
	retdata_len uint32, gas_left uint64, err error,
)
type getReturnDataType func() []byte

type apiClosure struct {
	getBytes32    getBytes32Type
	setBytes32    setBytes32Type
	callContract  callContractType
	getReturnData getReturnDataType
}

func newAPI(
	getBytes32 getBytes32Type,
	setBytes32 setBytes32Type,
	callContract callContractType,
	getReturnData getReturnDataType,
) C.GoApi {
	id := atomic.AddInt64(&apiIds, 1)
	apiClosures.Store(id, apiClosure{
		getBytes32:    getBytes32,
		setBytes32:    setBytes32,
		callContract:  callContract,
		getReturnData: getReturnData,
	})
	colors.PrintRed("Registered new API ", id)
	return C.GoApi{
		get_bytes32:     (*[0]byte)(C.getBytes32Wrap),
		set_bytes32:     (*[0]byte)(C.setBytes32Wrap),
		call_contract:   (*[0]byte)(C.callContractWrap),
		get_return_data: (*[0]byte)(C.getReturnDataWrap),
		id:              u64(id),
	}
}

func getAPI(api usize) *apiClosure {
	colors.PrintRed("Getting API ", api)
	any, ok := apiClosures.Load(int64(api))
	if !ok {
		log.Crit("failed to load stylus Go API", "id", api)
	}
	closures, ok := any.(apiClosure)
	if !ok {
		log.Crit("wrong type for stylus Go API", "id", api)
	}
	return &closures
}
