// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build !js
// +build !js

package programs

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#cgo LDFLAGS: ${SRCDIR}/../../target/lib/libstylus.a -ldl -lm
#include "arbitrator.h"

Bytes32 getBytes32Impl(size_t api, Bytes32 key, uint64_t * cost);
Bytes32 getBytes32Wrap(size_t api, Bytes32 key, uint64_t * cost) {
    return getBytes32Impl(api, key, cost);
}

uint8_t setBytes32Impl(size_t api, Bytes32 key, Bytes32 value, uint64_t * cost, RustVec * error);
uint8_t setBytes32Wrap(size_t api, Bytes32 key, Bytes32 value, uint64_t * cost, RustVec * error) {
    return setBytes32Impl(api, key, value, cost, error);
}

uint8_t callContractImpl(size_t api, Bytes20 contract, RustVec * data, uint64_t * gas, Bytes32 value);
uint8_t callContractWrap(size_t api, Bytes20 contract, RustVec * data, uint64_t * gas, Bytes32 value) {
    return callContractImpl(api, contract, data, gas, value);
}
*/
import "C"
import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
)

var apiClosures sync.Map
var apiIds int64 // atomic

type getBytes32Type func(key common.Hash) (value common.Hash, cost uint64)
type setBytes32Type func(key, value common.Hash) (cost uint64, err error)
type callContractType func(
	contract common.Address, input []byte, gas uint64, value *big.Int) (output []byte, gas_left uint64, err error,
)

type apiClosure struct {
	getBytes32   getBytes32Type
	setBytes32   setBytes32Type
	callContract callContractType
}

func newAPI(getBytes32 getBytes32Type, setBytes32 setBytes32Type, callContract callContractType) C.GoAPI {
	id := atomic.AddInt64(&apiIds, 1)
	apiClosures.Store(id, apiClosure{
		getBytes32:   getBytes32,
		setBytes32:   setBytes32,
		callContract: callContract,
	})
	return C.GoAPI{
		get_bytes32:   (*[0]byte)(C.getBytes32Wrap),
		set_bytes32:   (*[0]byte)(C.setBytes32Wrap),
		call_contract: (*[0]byte)(C.callContractWrap),
		id:            u64(id),
	}
}

func getAPI(api usize) (*apiClosure, error) {
	any, ok := apiClosures.Load(int64(api))
	if !ok {
		return nil, fmt.Errorf("failed to load stylus Go API %v", api)
	}
	closures, ok := any.(apiClosure)
	if !ok {
		return nil, errors.New("wrong type for stylus Go API")
	}
	return &closures, nil
}
