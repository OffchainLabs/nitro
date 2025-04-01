// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbcompress

type Dictionary uint32

const (
	EmptyDictionary Dictionary = iota
	StylusProgramDictionary
)

const LEVEL_WELL = 11
const WINDOW_SIZE = 22 // BROTLI_DEFAULT_WINDOW

func compressedBufferSizeFor(length int) int {
	return length + (length>>10)*8 + 64 // actual limit is: length + (length >> 14) * 4 + 6
}

func CompressLevel(input []byte, level uint64) ([]byte, error) {
	// level is trusted and shouldn't be anything crazy
	// #nosec G115
	return Compress(input, uint32(level), EmptyDictionary)
}
