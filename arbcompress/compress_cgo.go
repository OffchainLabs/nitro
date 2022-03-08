//go:build !js
// +build !js

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
	res := C.BrotliDecoderDecompress(C.size_t(len(input)), (*C.uint8_t)(&input[0]), &outsize, (*C.uint8_t)(&outbuf[0]))
	if res != 1 {
		return nil, fmt.Errorf("failed decompression: %d", res)
	}
	if int(outsize) > maxSize {
		return nil, fmt.Errorf("result too large: %d", outsize)
	}
	return outbuf[:outsize], nil
}

func compressLevel(input []byte, level int) ([]byte, error) {
	maxOutSize := maxCompressedSize(len(input))
	outbuf := make([]byte, maxOutSize)
	outSize := C.size_t(maxOutSize)
	var inputPtr *C.uint8_t
	if len(input) > 0 {
		inputPtr = (*C.uint8_t)(&input[0])
	}
	res := C.BrotliEncoderCompress(C.int(level), C.BROTLI_DEFAULT_WINDOW, C.BROTLI_MODE_GENERIC,
		C.size_t(len(input)), inputPtr, &outSize, (*C.uint8_t)(&outbuf[0]))
	if res != 1 {
		return nil, fmt.Errorf("failed compression: %d", res)
	}
	return outbuf[:outSize], nil
}

func CompressWell(input []byte) ([]byte, error) {
	return compressLevel(input, LEVEL_WELL)
}
