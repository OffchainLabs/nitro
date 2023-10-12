package arbutil

import (
	"math/big"

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
	A := big.NewInt(0).SetBytes(a)
	B := big.NewInt(0).SetBytes(b)
	return common.BytesToHash((A.Add(A, B)).Bytes()).Bytes()
}
