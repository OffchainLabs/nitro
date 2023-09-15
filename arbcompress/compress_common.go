// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbcompress

const LEVEL_WELL = 11
const WINDOW_SIZE = 22 // BROTLI_DEFAULT_WINDOW

func compressedBufferSizeFor(length int) int {
	return length + (length>>10)*8 + 64 // actual limit is: length + (length >> 14) * 4 + 6
}

func CompressFast(input []byte, level int) ([]byte, error) {
	return compressLevel(input, level)
}
