// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build js
// +build js

package arbcompress

import (
	"fmt"
)

func brotliCompress(inBuf []byte, outBuf []byte, level, windowSize uint32) (outLen uint64, status BrotliStatus)

func brotliDecompress(inBuf []byte, outBuf []byte) (outLen uint64, status BrotliStatus)

func Decompress(input []byte, maxSize int) ([]byte, error) {
	outBuf := make([]byte, maxSize)
	outLen, status := brotliDecompress(input, outBuf)
	if status != BrotliSuccess {
		return nil, fmt.Errorf("failed decompression")
	}
	return outBuf[:outLen], nil
}

func compressLevel(input []byte, level uint32) ([]byte, error) {
	maxOutSize := compressedBufferSizeFor(len(input))
	outBuf := make([]byte, maxOutSize)
	outLen, status := brotliCompress(input, outBuf, level, WINDOW_SIZE)
	if status != BrotliSuccess {
		return nil, fmt.Errorf("failed compression")
	}
	return outBuf[:outLen], nil
}
