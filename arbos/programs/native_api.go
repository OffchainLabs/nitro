// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build !js
// +build !js

package programs

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#cgo LDFLAGS: ${SRCDIR}/../../target/lib/libstylus.a -ldl -lm
#include "arbitrator.h"

extern Bytes32 getBytes32API(size_t api, Bytes32 key, uint64_t * cost);
extern size_t  setBytes32API(size_t api, Bytes32 key, Bytes32 value, uint64_t * cost);

Bytes32 getBytes32WrapperC(size_t api, Bytes32 key, uint64_t * cost) {
    return getBytes32API(api, key, cost);
}
size_t setBytes32WrapperC(size_t api, Bytes32 key, Bytes32 value, uint64_t * cost) {
    return setBytes32API(api, key, value, cost);
}
*/
import "C"
import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
)

var apiClosures sync.Map
var apiIds int64 // atomic

type getBytes32Type func(key common.Hash) (common.Hash, uint64)
type setBytes32Type func(key, value common.Hash) (uint64, error)

type apiClosure struct {
	getBytes32 getBytes32Type
	setBytes32 setBytes32Type
}

func newAPI(getBytes32 getBytes32Type, setBytes32 setBytes32Type) C.GoAPI {
	id := atomic.AddInt64(&apiIds, 1)
	apiClosures.Store(id, apiClosure{
		getBytes32: getBytes32,
		setBytes32: setBytes32,
	})
	return C.GoAPI{
		get_bytes32: (*[0]byte)(C.getBytes32WrapperC),
		set_bytes32: (*[0]byte)(C.setBytes32WrapperC),
		id:          u64(id),
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
