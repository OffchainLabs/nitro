package tree

import (
	"crypto/sha256"
	"hash"

	"github.com/ethereum/go-ethereum/common"
)

// customHasher embeds hash.Hash and includes a map for the hash-to-preimage mapping
type NmtPreimageHasher struct {
	hash.Hash
	record func(bytes32, []byte)
	data   []byte
}

// Need to make sure this is writting relevant data into the tree
// Override the Sum method to capture the preimage
func (h *NmtPreimageHasher) Sum(b []byte) []byte {
	hashed := h.Hash.Sum(nil)
	hashKey := common.BytesToHash(hashed)
	h.record(hashKey, append([]byte(nil), h.data...))
	return h.Hash.Sum(b)
}

func (h *NmtPreimageHasher) Write(p []byte) (n int, err error) {
	h.data = append(h.data[:0], p...) // Update the data slice with the new data
	return h.Hash.Write(p)
}

// Override the Reset method to clean the hash state and the data slice
func (h *NmtPreimageHasher) Reset() {
	h.Hash.Reset()
	h.data = h.data[:0] // Reset the data slice to be empty, but keep the underlying array
}

func newNmtPreimageHasher(record func(bytes32, []byte)) hash.Hash {
	return &NmtPreimageHasher{
		Hash:   sha256.New(),
		record: record,
	}
}
