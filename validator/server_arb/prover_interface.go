// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package server_arb

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#cgo LDFLAGS: ${SRCDIR}/../../target/lib/libprover.a -ldl -lm
#include "arbitrator.h"
#include <stdlib.h>

char **PrepareStringList(intptr_t num) {
	char** res = malloc(sizeof(char*) * num);
	if (! res) {
		return 0;
	}
	return res;
}

void AddToStringList(char** list, int index, char* val) {
	list[index] = val;
}
*/
import "C"
import (
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/validator"
)

func CreateCByteArray(input []byte) C.CByteArray {
	return C.CByteArray{
		// Warning: CBytes uses malloc internally, so this must be freed later
		ptr: (*C.uint8_t)(C.CBytes(input)),
		len: C.uintptr_t(len(input)),
	}
}

func DestroyCByteArray(cbyte C.CByteArray) {
	C.free(unsafe.Pointer(cbyte.ptr))
}

func GlobalStateToC(gsIn validator.GoGlobalState) C.GlobalState {
	gs := C.GlobalState{}
	gs.u64_vals[0] = C.uint64_t(gsIn.Batch)
	gs.u64_vals[1] = C.uint64_t(gsIn.PosInBatch)
	for i, b := range gsIn.BlockHash {
		gs.bytes32_vals[0].bytes[i] = C.uint8_t(b)
	}
	for i, b := range gsIn.SendRoot {
		gs.bytes32_vals[1].bytes[i] = C.uint8_t(b)
	}
	return gs
}

func GlobalStateFromC(gs C.GlobalState) validator.GoGlobalState {
	var blockHash common.Hash
	for i := range blockHash {
		blockHash[i] = byte(gs.bytes32_vals[0].bytes[i])
	}
	var sendRoot common.Hash
	for i := range sendRoot {
		sendRoot[i] = byte(gs.bytes32_vals[1].bytes[i])
	}
	return validator.GoGlobalState{
		Batch:      uint64(gs.u64_vals[0]),
		PosInBatch: uint64(gs.u64_vals[1]),
		BlockHash:  blockHash,
		SendRoot:   sendRoot,
	}
}

// CreateCStringList creates a list of strings, does take ownership, should be freed
func CreateCStringList(input []string) **C.char {
	res := C.PrepareStringList(C.intptr_t(len(input)))
	for i, str := range input {
		C.AddToStringList(res, C.int(i), C.CString(str))
	}
	return res
}

func FreeCStringList(arrPtr **C.char, size int) {
	arr := (*[1 << 30]*C.char)(unsafe.Pointer(arrPtr))[:size:size]
	for _, ptr := range arr {
		C.free(unsafe.Pointer(ptr))
	}
	C.free(unsafe.Pointer(arrPtr))
}
