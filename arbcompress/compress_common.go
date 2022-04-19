// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbcompress

const LEVEL_FAST = 0
const LEVEL_WELL = 11
const WINDOW_SIZE = 22 // BROTLI_DEFAULT_WINDOW

func compressedBufferSizeFor(len int) int {
	return len + (len>>10)*8 + 64 // actual limit is: len + (len >> 14) * 4 + 6
}

func CompressFast(input []byte) ([]byte, error) {
	return compressLevel(input, LEVEL_FAST)
}
