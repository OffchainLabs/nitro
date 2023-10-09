package arbutil

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// PaddedKeccak256 pads each argument to 32 bytes, concatenates and returns
// keccak256 hash of the result.
func PaddedKeccak256(args ...[]byte) []byte {
	var data []byte
	for _, arg := range args {
		data = append(data, common.BytesToHash(arg).Bytes()...)
	}
	return crypto.Keccak256(data)
}

// SumBytes sums two byte slices and returns the result.
// If the sum of bytes are over 32 bytes, it return last 32.
func SumBytes(a, b []byte) []byte {
	// Normalize lengths to hash length.
	a = common.BytesToHash(a).Bytes()
	b = common.BytesToHash(b).Bytes()

	sum := make([]byte, common.HashLength)
	c := 0
	for i := common.HashLength - 1; i >= 0; i-- {
		tmp := int(a[i]) + int(b[i]) + c
		sum[i] = byte(tmp & 0xFF)
		c = tmp >> 8
	}
	return sum
}
