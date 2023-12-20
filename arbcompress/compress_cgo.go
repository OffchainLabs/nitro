// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build !wasm
// +build !wasm

package arbcompress

/*
#cgo CFLAGS: -g -Wall -I${SRCDIR}/../target/include/
#cgo LDFLAGS: ${SRCDIR}/../target/lib/libbrotlidec-static.a ${SRCDIR}/../target/lib/libbrotlienc-static.a ${SRCDIR}/../target/lib/libbrotlicommon-static.a -lm
#include "brotli/encode.h"
#include "brotli/decode.h"
*/
import "C"
import (
	"fmt"
)

func Decompress(input []byte, maxSize int) ([]byte, error) {
	outbuf := make([]byte, maxSize)
	outsize := C.size_t(maxSize)
	var ptr *C.uint8_t
	if len(input) > 0 {
		ptr = (*C.uint8_t)(&input[0])
	}
	res := C.BrotliDecoderDecompress(C.size_t(len(input)), ptr, &outsize, (*C.uint8_t)(&outbuf[0]))
	if uint32(res) != BrotliSuccess {
		return nil, fmt.Errorf("failed decompression: %d", res)
	}
	if int(outsize) > maxSize {
		return nil, fmt.Errorf("result too large: %d", outsize)
	}
	return outbuf[:outsize], nil
}

func compressLevel(input []byte, level int) ([]byte, error) {
	maxOutSize := compressedBufferSizeFor(len(input))
	outbuf := make([]byte, maxOutSize)
	outSize := C.size_t(maxOutSize)
	var inputPtr *C.uint8_t
	if len(input) > 0 {
		inputPtr = (*C.uint8_t)(&input[0])
	}
	res := C.BrotliEncoderCompress(
		C.int(level), C.BROTLI_DEFAULT_WINDOW, C.BROTLI_MODE_GENERIC,
		C.size_t(len(input)), inputPtr, &outSize, (*C.uint8_t)(&outbuf[0]),
	)
	if uint32(res) != BrotliSuccess {
		return nil, fmt.Errorf("failed compression: %d", res)
	}
	return outbuf[:outSize], nil
}

func CompressWell(input []byte) ([]byte, error) {
	return compressLevel(input, LEVEL_WELL)
}
