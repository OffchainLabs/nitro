package arbutil

import (
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
)

// slotAddress pads each argument to 32 bytes, concatenates and returns
// keccak256 hashe of the result.
func slotAddress(args ...[]byte) []byte {
	hash := sha3.NewLegacyKeccak256()
	for _, arg := range args {
		// fmt.Printf("%x\n", common.BytesToHash(arg).Bytes())
		hash.Write(common.BytesToHash(arg).Bytes())
	}
	return hash.Sum(nil)
}
