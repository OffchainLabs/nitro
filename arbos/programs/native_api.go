// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build !js
// +build !js

package programs

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#cgo LDFLAGS: ${SRCDIR}/../../target/lib/libstylus.a -ldl -lm
#include "arbitrator.h"

extern Bytes32 getBytes32API(size_t api, Bytes32 key);
extern void    setBytes32API(size_t api, Bytes32 key, Bytes32 value);

Bytes32 getBytes32WrapperC(size_t api, Bytes32 key) {
    return getBytes32API(api, key);
}
void setBytes32WrapperC(size_t api, Bytes32 key, Bytes32 value) {
    setBytes32API(api, key, value);
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

type getBytes32Type func(key common.Hash) common.Hash
type setBytes32Type func(key, value common.Hash)

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
