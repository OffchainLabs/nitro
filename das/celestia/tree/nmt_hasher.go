package tree

import (
	"crypto/sha256"
	"hash"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"
)

// customHasher embeds hash.Hash and includes a map for the hash-to-preimage mapping
type NmtPreimageHasher struct {
	hash.Hash
	record func(bytes32, []byte, arbutil.PreimageType)
	data   []byte
}

// Need to make sure this is writting relevant data into the tree
// Override the Sum method to capture the preimage
func (h *NmtPreimageHasher) Sum(b []byte) []byte {
	hashed := h.Hash.Sum(nil)
	hashKey := common.BytesToHash(hashed)
	h.record(hashKey, append([]byte(nil), h.data...), arbutil.Sha2_256PreimageType)
	return h.Hash.Sum(b)
}

func (h *NmtPreimageHasher) Write(p []byte) (n int, err error) {
	h.data = append(h.data, p...)
	return h.Hash.Write(p)
}

// Override the Reset method to clean the hash state and the data slice
func (h *NmtPreimageHasher) Reset() {
	h.Hash.Reset()
	h.data = h.data[:0] // Reset the data slice to be empty, but keep the underlying array
}

func newNmtPreimageHasher(record func(bytes32, []byte, arbutil.PreimageType)) hash.Hash {
	return &NmtPreimageHasher{
		Hash:   sha256.New(),
		record: record,
	}
}
