// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !wasm

package arbcompress

/*
#cgo CFLAGS: -g -I${SRCDIR}/../target/include/
#cgo LDFLAGS: ${SRCDIR}/../target/lib/libstylus.a -lm
#include "arbitrator.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

type u8 = C.uint8_t
type u32 = C.uint32_t
type usize = C.size_t

type brotliBuffer = C.BrotliBuffer

type StreamingCompressor = unsafe.Pointer

func CompressWell(input []byte) ([]byte, error) {
	return Compress(input, LEVEL_WELL, EmptyDictionary)
}

func CreateStreamingCompressor(level uint32) StreamingCompressor {
	return C.brotli_create_compressing_writer(u32(level))
}

func WriteToStreamingCompressor(state StreamingCompressor, input []byte, output []byte) int {
	outbuf := sliceToBuffer(output)
	inbuf := sliceToBuffer(input)
	return int(C.brotli_write_to_stream(state, inbuf, outbuf))
}

func FlushStreamingCompressor(state StreamingCompressor, output []byte) {
	outbuf := sliceToBuffer(output)
	C.brotli_flush_stream(state, outbuf)
}

func CloseStreamingCompressor(state StreamingCompressor, output []byte) {
	outbuf := sliceToBuffer(output)
	C.brotli_close_stream(state, outbuf)
}

func Compress(input []byte, level uint32, dictionary Dictionary) ([]byte, error) {
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

var ErrOutputWontFit = errors.New("output won't fit in maxsize")

func Decompress(input []byte, maxSize int) ([]byte, error) {
	return DecompressWithDictionary(input, maxSize, EmptyDictionary)
}

func DecompressWithDictionary(input []byte, maxSize int, dictionary Dictionary) ([]byte, error) {
	output := make([]byte, maxSize)
	outbuf := sliceToBuffer(output)
	inbuf := sliceToBuffer(input)

	status := C.brotli_decompress(inbuf, outbuf, C.Dictionary(dictionary))
	if status == C.BrotliStatus_NeedsMoreOutput {
		return nil, ErrOutputWontFit
	}
	if status != C.BrotliStatus_Success {
		return nil, fmt.Errorf("failed decompression: %d", status)
	}
	if *outbuf.len > usize(maxSize) {
		return nil, fmt.Errorf("failed decompression: result too large: %d, wanted: < %d", *outbuf.len, maxSize)
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
