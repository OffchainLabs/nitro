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
	"io"
	"runtime"
	"unsafe"
)

type u8 = C.uint8_t
type u32 = C.uint32_t
type usize = C.size_t

type brotliBuffer = C.BrotliBuffer

func CompressWell(input []byte) ([]byte, error) {
	return Compress(input, LEVEL_WELL, EmptyDictionary)
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

// Writer implements io.Writer using streaming brotli compression via the C library.
// It provides a drop-in replacement for github.com/andybalholm/brotli.Writer.
type Writer struct {
	dst    io.Writer
	state  unsafe.Pointer
	level  int
	buffer []byte
	err    error
}

var (
	errEncode       = errors.New("brotli: encode error")
	errWriterClosed = errors.New("brotli: Writer is closed")
)

// NewWriterLevel creates a new Writer with the specified compression level.
// The compression level can be 0-11, where 0 is fastest and 11 is best compression.
func NewWriterLevel(dst io.Writer, level int) *Writer {
	w := &Writer{
		level:  level,
		buffer: make([]byte, 32768), // 32KB output buffer
	}
	w.Reset(dst)
	return w
}

// Reset discards the Writer's state and makes it equivalent to the result of
// its original state from NewWriter or NewWriterLevel, but writing to dst instead.
func (w *Writer) Reset(dst io.Writer) {
	if w.state != nil {
		C.BrotliEncoderDestroyInstance(w.state)
		w.state = nil
	}

	w.dst = dst
	w.err = nil

	// Create the encoder instance
	w.state = C.BrotliEncoderCreateInstance(nil, nil, nil)
	if w.state == nil {
		w.err = errEncode
		return
	}

	// Set quality parameter
	if C.BrotliEncoderSetParameter(w.state, C.BrotliEncoderParameter_Quality, C.uint32_t(w.level)) == 0 {
		w.err = errEncode
		return
	}

	// Set window size parameter
	if C.BrotliEncoderSetParameter(w.state, C.BrotliEncoderParameter_WindowSize, C.uint32_t(WINDOW_SIZE)) == 0 {
		w.err = errEncode
		return
	}
}

// Write implements io.Writer by compressing p and writing the compressed data to the underlying writer.
func (w *Writer) Write(p []byte) (n int, err error) {
	if w.dst == nil {
		return 0, errWriterClosed
	}
	if w.err != nil {
		return 0, w.err
	}
	if len(p) == 0 {
		return 0, nil
	}

	return w.compress(p, C.BrotliEncoderOperation_Process)
}

// Flush outputs any buffered compressed data.
func (w *Writer) Flush() error {
	if w.dst == nil {
		return errWriterClosed
	}
	if w.err != nil {
		return w.err
	}

	_, err := w.compress(nil, C.BrotliEncoderOperation_Flush)
	return err
}

// Close flushes remaining data and finalizes the compressed stream.
func (w *Writer) Close() error {
	if w.dst == nil {
		return errWriterClosed
	}
	if w.err != nil {
		return w.err
	}

	_, err := w.compress(nil, C.BrotliEncoderOperation_Finish)
	if w.state != nil {
		C.BrotliEncoderDestroyInstance(w.state)
		w.state = nil
	}
	w.dst = nil
	return err
}

// compress handles the actual compression operation
func (w *Writer) compress(p []byte, op C.BrotliEncoderOperation) (n int, err error) {
	// Pin the Go memory to prevent it from being moved by the GC while C code is using it
	var pinner runtime.Pinner
	defer pinner.Unpin()

	inputLen := usize(len(p))
	inputPtr := (*u8)(nil)
	if len(p) > 0 {
		pinner.Pin(&p[0])
		inputPtr = (*u8)(unsafe.Pointer(&p[0]))
	}

	// Pin the output buffer
	pinner.Pin(&w.buffer[0])

	for {
		availableIn := inputLen
		nextIn := inputPtr
		availableOut := usize(len(w.buffer))
		nextOut := (*u8)(unsafe.Pointer(&w.buffer[0]))
		totalOut := usize(0)

		// Note: BrotliEncoderCompressStream expects enum BrotliEncoderOperation
		// which CGO represents as C.BrotliEncoderOperation (a uint32 underneath)
		opValue := uint32(op)
		//nolint:gocritic // False positive - linter confused by CGO code
		success := C.BrotliEncoderCompressStream(
			w.state,
			opValue,
			&availableIn,
			&nextIn,
			&availableOut,
			&nextOut,
			&totalOut,
		)

		if success == 0 {
			w.err = errEncode
			return n, w.err
		}

		bytesConsumed := int(inputLen - availableIn)
		n += bytesConsumed
		inputLen = availableIn
		if nextIn != nil && bytesConsumed > 0 {
			inputPtr = (*u8)(unsafe.Pointer(uintptr(unsafe.Pointer(nextIn))))
		}

		// Calculate how many bytes were written to the output buffer
		// by checking how much space is left
		outputSize := len(w.buffer) - int(availableOut)
		if outputSize > 0 {
			written, writeErr := w.dst.Write(w.buffer[:outputSize])
			if writeErr != nil {
				w.err = writeErr
				return n, w.err
			}
			if written != outputSize {
				w.err = io.ErrShortWrite
				return n, w.err
			}
		}

		if inputLen == 0 {
			if op == C.BrotliEncoderOperation_Finish && C.BrotliEncoderIsFinished(w.state) == 0 {
				continue
			}
			break
		}
	}

	return n, nil
}
