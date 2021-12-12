package validator

/*
#cgo CFLAGS: -g -Wall -I../arbitrator/target/env/include/
#cgo LDFLAGS: ${SRCDIR}/../arbitrator/target/env/lib/libprover.a -ldl -lm
#include "arbitrator.h"
#include <stdlib.h>

// same as arbitrator defines, but without constant pointers
struct TempByteArray {
  uint8_t *ptr;
  uintptr_t len;
};

CByteArray CreateCByteArray(uint8_t* ptr, uintptr_t len) {
	CByteArray retval = {ptr, len};
	return retval;
}

void DestroyCByteArray(CByteArray arr) {
	free((void *)arr.ptr);
}

CMultipleByteArrays CreateMultipleCByteArrays(uintptr_t num) {
	CMultipleByteArrays retval = {malloc(sizeof(struct TempByteArray) * num), num};
	return retval;
}

int CopyCByteToMultiple(CMultipleByteArrays multiple, uintptr_t index, CByteArray cbyte) {
	if (multiple.len < index) {
		return -1;
	}
	if (!multiple.ptr) {
		return -2;
	}
	struct TempByteArray *tempPtr = (struct TempByteArray *)&multiple.ptr[index];
	tempPtr->ptr = (uint8_t *)cbyte.ptr;
	tempPtr->len = cbyte.len;
	return 0;
}

struct TempByteArray TempByteFromMultiple(CMultipleByteArrays multiple, uintptr_t index) {
	struct TempByteArray res;
	res.len = 0;

	if (multiple.len < index) {
		return res;
	}
	if (!multiple.ptr) {
		return res;
	}
	struct TempByteArray *tempPtr = (struct TempByteArray *)&multiple.ptr[index];
	res.ptr = tempPtr->ptr;
	res.len = tempPtr->len;
	return res;
}

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

int Bytes32ToGlobalState(struct GlobalState *gs, int idx, void *hash) {
	uint8_t *hashBytes = (uint8_t *)hash;
	if (idx >= GLOBAL_STATE_BYTES32_NUM) {
		return -1;
	}
	for (int i=0; i<32 ; i++) {
		gs->bytes32_vals[idx].bytes[i] = hashBytes[i];
	}
	return 0;
}

int Bytes32FromGlobalState(struct GlobalState *gs, int idx, void *hash) {
	uint8_t *hashBytes = (uint8_t *)hash;
	if (idx >= GLOBAL_STATE_BYTES32_NUM) {
		return -1;
	}
	for (int i=0; i<32 ; i++) {
		hashBytes[i] = gs->bytes32_vals[idx].bytes[i];
	}
	return 0;
}

struct CByteArray InboxReaderFunc(uint64_t context, uint64_t inbox_idx, uint64_t seq_num);

struct CByteArray InboxReaderWrapper(uint64_t context, uint64_t inbox_idx, uint64_t seq_num){
	return InboxReaderFunc(context, inbox_idx, seq_num);
}

*/
import "C"
import (
	"encoding/binary"
	"os"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
)

var cbyteError = C.CByteArray{ptr: nil, len: 1}

func AllocateMultipleCByteArrays(length int) C.CMultipleByteArrays {
	return C.CreateMultipleCByteArrays(C.uintptr_t(length))
}

// Does not clone / take ownership of data
func UpdateCByteArrayInMultiple(array C.CMultipleByteArrays, index int, cbyte C.CByteArray) {
	C.CopyCByteToMultiple(array, C.uintptr_t(index), cbyte)
}

func CreateCByteArray(input []byte) C.CByteArray {
	return C.CreateCByteArray((*C.uint8_t)(C.CBytes(input)), C.uintptr_t(len(input)))
}

func DestroyCByteArray(cbyte C.CByteArray) {
	C.DestroyCByteArray(cbyte)
}

func CreateGlobalState(batch uint64, posinBatch uint64, blockHash common.Hash) C.GlobalState {
	gs := C.GlobalState{}
	gs.u64_vals[0] = C.uint64_t(batch)
	gs.u64_vals[1] = C.uint64_t(posinBatch)
	C.Bytes32ToGlobalState(&gs, 0, unsafe.Pointer(&blockHash[0]))
	return gs
}

func ParseGlobalState(gs C.GlobalState) (batch uint64, posinBatch uint64, blockHash common.Hash) {
	batch = uint64(gs.u64_vals[0])
	posinBatch = uint64(gs.u64_vals[1])
	C.Bytes32FromGlobalState(&gs, 0, unsafe.Pointer(&blockHash[0]))
	return
}

// creates a list of strings, does take ownership, should be freed
func CreateCStringList(input []string) **C.char {
	res := C.PrepareStringList(C.intptr_t(len(input)))
	for i, str := range input {
		C.AddToStringList(res, C.int(i), C.CString(str))
	}
	return res
}

// single file with the binary blob
func CByteToFile(cData C.CByteArray, path string) error {
	data := C.GoBytes(unsafe.Pointer(cData.ptr), C.int(cData.len))
	return os.WriteFile(path, data, 0644)
}

// single file with multiple values, each prefixed with it's size
func CMultipleByteArrayToFile(cMulti C.CMultipleByteArrays, path string) error {
	bufNum := int(cMulti.len)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	for i := 0; i < bufNum; i++ {
		cTempByte := C.TempByteFromMultiple(cMulti, C.uintptr_t(i))
		data := C.GoBytes(unsafe.Pointer(cTempByte.ptr), C.int(cTempByte.len))
		lenbytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(lenbytes, uint64(cTempByte.len))
		_, err := file.Write(lenbytes)
		if err != nil {
			return err
		}
		_, err = file.Write(data)
		if err != nil {
			return err
		}
	}
	return nil
}
