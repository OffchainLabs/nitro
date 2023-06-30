// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

//go:build js
// +build js

package arbcompress

import (
	"fmt"
)

func brotliCompress(inBuf []byte, outBuf []byte, level int, windowSize int) int64

func brotliDecompress(inBuf []byte, outBuf []byte) int64

func Decompress(input []byte, maxSize int) ([]byte, error) {
	outBuf := make([]byte, maxSize)
	outLen := brotliDecompress(input, outBuf)
	if outLen < 0 {
		return nil, fmt.Errorf("failed decompression")
	}
	return outBuf[:outLen], nil
}

func compressLevel(input []byte, level int) ([]byte, error) {
	maxOutSize := compressedBufferSizeFor(len(input))
	outBuf := make([]byte, maxOutSize)
	outLen := brotliCompress(input, outBuf, level, WINDOW_SIZE)
	if outLen < 0 {
		return nil, fmt.Errorf("failed compression")
	}
	return outBuf[:outLen], nil
}
