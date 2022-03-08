package arbcompress

const LEVEL_FAST = 0
const LEVEL_WELL = 11
const WINDOW_SIZE = 22 // BROTLI_DEFAULT_WINDOW

func maxCompressedSize(len int) int {
	return len + (len >> 12) + 6
}

func CompressFast(input []byte) ([]byte, error) {
	return compressLevel(input, LEVEL_FAST)
}
