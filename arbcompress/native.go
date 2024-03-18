// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build !wasm
// +build !wasm

package arbcompress

/*
#cgo CFLAGS: -g -Wall -I${SRCDIR}/../target/include/
#cgo LDFLAGS: ${SRCDIR}/../target/lib/libstylus.a -lm
#include "arbitrator.h"
*/
import "C"
import "fmt"

type u8 = C.uint8_t
type u32 = C.uint32_t
type usize = C.size_t

type brotliBool = uint32
type brotliBuffer = C.BrotliBuffer

const (
	brotliFalse brotliBool = iota
	brotliTrue
)

func Decompress(input []byte, maxSize int) ([]byte, error) {
	return DecompressWithDictionary(input, maxSize, EmptyDictionary)
}

func DecompressWithDictionary(input []byte, maxSize int, dictionary Dictionary) ([]byte, error) {
	output := make([]byte, maxSize)
	outbuf := sliceToBuffer(output)
	inbuf := sliceToBuffer(input)

	status := C.brotli_decompress(inbuf, outbuf, C.Dictionary(dictionary))
	if status != C.BrotliStatus_Success {
		return nil, fmt.Errorf("failed decompression: %d", status)
	}
	if *outbuf.len > usize(maxSize) {
		return nil, fmt.Errorf("failed decompression: result too large: %d", *outbuf.len)
	}
	output = output[:*outbuf.len]
	return output, nil
}

func CompressWell(input []byte) ([]byte, error) {
	return compressLevel(input, LEVEL_WELL, EmptyDictionary)
}

func compressLevel(input []byte, level int, dictionary Dictionary) ([]byte, error) {
	maxSize := compressedBufferSizeFor(len(input))
	output := make([]byte, maxSize)
	outbuf := sliceToBuffer(output)
	inbuf := sliceToBuffer(input)

	status := C.brotli_compress(inbuf, outbuf, C.Dictionary(dictionary), u32(level))
	if status != C.BrotliStatus_Success {
		return nil, fmt.Errorf("failed decompression: %d", status)
	}
	output = output[:*outbuf.len]
	return output, nil
}

func sliceToBuffer(slice []byte) brotliBuffer {
	count := usize(len(slice))
	if count == 0 {
		slice = []byte{0x00} // ensures pointer is not null (shouldn't be necessary, but brotli docs are picky about NULL)
	}
	return brotliBuffer{
		ptr: (*u8)(&slice[0]),
		len: &count,
	}
}
