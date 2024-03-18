// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build wasm
// +build wasm

package arbcompress

import (
	"fmt"
	"unsafe"

	"github.com/offchainlabs/nitro/arbutil"
)

//go:wasmimport arbcompress brotli_compress
func brotliCompress(inBuf unsafe.Pointer, inBufLen uint32, outBuf unsafe.Pointer, outBufLen unsafe.Pointer, level, windowSize uint32) brotliStatus

//go:wasmimport arbcompress brotli_decompress
func brotliDecompress(inBuf unsafe.Pointer, inBufLen uint32, outBuf unsafe.Pointer, outBufLen unsafe.Pointer, dictionary Dictionary) brotliStatus

func Decompress(input []byte, maxSize int) ([]byte, error) {
	return DecompressWithDictionary(input, maxSize, EmptyDictionary)
}

func DecompressWithDictionary(input []byte, maxSize int, dictionary Dictionary) ([]byte, error) {
	outBuf := make([]byte, maxSize)
	outLen := uint32(len(outBuf))
	status := brotliDecompress(
		arbutil.SliceToUnsafePointer(input),
		uint32(len(input)),
		arbutil.SliceToUnsafePointer(outBuf),
		unsafe.Pointer(&outLen),
		dictionary,
	)
	if status != brotliSuccess {
		return nil, fmt.Errorf("failed decompression")
	}
	return outBuf[:outLen], nil
}

func compressLevel(input []byte, level uint32) ([]byte, error) {
	maxOutSize := compressedBufferSizeFor(len(input))
	outBuf := make([]byte, maxOutSize)
	outLen := uint32(len(outBuf))
	status := brotliCompress(
		arbutil.SliceToUnsafePointer(input), uint32(len(input)),
		arbutil.SliceToUnsafePointer(outBuf), unsafe.Pointer(&outLen),
		level,
		WINDOW_SIZE,
	)
	if status != brotliSuccess {
		return nil, fmt.Errorf("failed compression")
	}
	return outBuf[:outLen], nil
}
