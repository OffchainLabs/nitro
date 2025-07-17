package execution_engine

import (
	"crypto"

	"github.com/ethereum/go-ethereum/common"
	geth_crypo "github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

// MessageHasher is an interface that defines the common hashing functionality
// for messages with metadata. It allows for different hashing algorithms to be
// used interchangeably.
type MessageHasher interface {
	HashMessageWithMetadata(msg *arbostypes.MessageWithMetadata) common.Hash
}

// StdLibCryptoHasher is a struct to wrap the crypto.Hash interface defined
// by the Go standard library. This allows for the substitution or swapping
// of the hashing algorithms provided by the Go standard library's crypto
// package.
type StdLibCryptoHasher struct {
	Hash crypto.Hash
}

// Compile time check to ensure that StdLibCryptoHasher implements the
// MessageHasher interface.
var _ MessageHasher = StdLibCryptoHasher{}

// NewStdLibHasher creates a new instance of StdLibCryptoHasher with the
// specified crypto.Hash algorithm.
func NewStdLibHasher(hash crypto.Hash) StdLibCryptoHasher {
	return StdLibCryptoHasher{Hash: hash}
}

// HashMessageWithMetadata implements MessageHasher
func (h StdLibCryptoHasher) HashMessageWithMetadata(msg *arbostypes.MessageWithMetadata) common.Hash {
	hasher := h.Hash.New()
	hasher.Write([]byte{msg.Message.Header.Kind})
	hasher.Write(msg.Message.L2msg)
	var hash common.Hash
	copy(hash[:], hasher.Sum(nil)[:common.HashLength])
	return hash
}

// KeccakHasher is a struct that is a place holder for the Keccak hashing
// algorithm provided by the go-ethereum library.
type KeccakHasher struct{}

// Compile time check to ensure that KeccakHasher implements the
// MessageHasher interface.
var _ MessageHasher = KeccakHasher{}

// HashMessageWithMetadata implements MessageHasher
func (h KeccakHasher) HashMessageWithMetadata(msg *arbostypes.MessageWithMetadata) common.Hash {
	keccak := geth_crypo.NewKeccakState()
	keccak.Write([]byte{msg.Message.Header.Kind})
	keccak.Write(msg.Message.L2msg)
	var hash common.Hash
	keccak.Read(hash[:])
	return hash
}

// DefaultMessageHasher is the default MessageHasher used by the
// MockExecutionEngine.
var DefaultMessageHasher MessageHasher = KeccakHasher{}
