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

type u8 = C.uint8_t
type usize = C.size_t

type brotliBool = uint32

const (
	brotliFalse brotliBool = iota
	brotliTrue
)

const (
	rawSharedDictionary        C.BrotliSharedDictionaryType = iota // LZ77 prefix dictionary
	serializedSharedDictionary                                     // Serialized dictionary
)

func (d Dictionary) data() []byte {
	return []byte{}
}

func Decompress(input []byte, maxSize int) ([]byte, error) {
	return DecompressWithDictionary(input, maxSize, EmptyDictionary)
}

func DecompressWithDictionary(input []byte, maxSize int, dictionary Dictionary) ([]byte, error) {
	state := C.BrotliDecoderCreateInstance(nil, nil, nil)
	defer C.BrotliDecoderDestroyInstance(state)

	if dictionary != EmptyDictionary {
		data := dictionary.data()
		attached := C.BrotliDecoderAttachDictionary(
			state,
			rawSharedDictionary,
			usize(len(data)),
			sliceToPointer(data),
		)
		if uint32(attached) != brotliTrue {
			return nil, fmt.Errorf("failed decompression: failed to attach dictionary")
		}
	}

	inLen := usize(len(input))
	inPtr := sliceToPointer(input)
	output := make([]byte, maxSize)
	outLen := usize(maxSize)
	outLeft := usize(len(output))
	outPtr := sliceToPointer(output)

	status := C.BrotliDecoderDecompressStream(
		state,
		&inLen,
		&inPtr,
		&outLeft,
		&outPtr,
		&outLen, //nolint:gocritic
	)
	if uint32(status) != brotliSuccess {
		return nil, fmt.Errorf("failed decompression: failed streaming: %d", status)
	}
	if int(outLen) > maxSize {
		return nil, fmt.Errorf("failed decompression: result too large: %d", outLen)
	}
	return output[:outLen], nil
}

func compressLevel(input []byte, level int) ([]byte, error) {
	maxOutSize := compressedBufferSizeFor(len(input))
	outbuf := make([]byte, maxOutSize)
	outSize := C.size_t(maxOutSize)
	inputPtr := sliceToPointer(input)
	outPtr := sliceToPointer(outbuf)

	res := C.BrotliEncoderCompress(
		C.int(level), C.BROTLI_DEFAULT_WINDOW, C.BROTLI_MODE_GENERIC,
		C.size_t(len(input)), inputPtr, &outSize, outPtr,
	)
	if uint32(res) != brotliSuccess {
		return nil, fmt.Errorf("failed compression: %d", res)
	}
	return outbuf[:outSize], nil
}

func CompressWell(input []byte) ([]byte, error) {
	return compressLevel(input, LEVEL_WELL)
}

func sliceToPointer(slice []byte) *u8 {
	if len(slice) == 0 {
		slice = []byte{0x00} // ensures pointer is not null (shouldn't be necessary, but brotli docs are picky about NULL)
	}
	return (*u8)(&slice[0])
}
