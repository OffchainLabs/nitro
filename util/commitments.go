package util

import (
	"encoding/binary"

	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	emptyCommit = HistoryCommitment{}
)

// StateCommitment is a type used to represent the state commitment of an assertion.
type StateCommitment struct {
	Height    uint64      `json:"height"`
	StateRoot common.Hash `json:"state_root"`
}

// Hash returns the hash of the state commitment.
func (comm StateCommitment) Hash() common.Hash {
	return crypto.Keccak256Hash(binary.BigEndian.AppendUint64([]byte{}, comm.Height), comm.StateRoot.Bytes())
}

// HistoryCommitment defines a Merkle accumulator over a list of leaves, which
// are understood to be state roots in the goimpl. A history commitment contains
// a "height" value, which can refer to a height of an assertion in the assertions
// tree, or a "step" of WAVM states in a big step or small step subchallenge.
// A commitment contains a Merkle root over the list of leaves, and can optionally
// provide a proof that the last leaf in the accumulator Merkleizes into the
// specified root hash, which is required when verifying challenge creation invariants.
type HistoryCommitment struct {
	Height         uint64
	Range          uint64
	Merkle         common.Hash
	FirstLeaf      common.Hash
	LastLeafProof  []common.Hash
	FirstLeafProof []common.Hash
	LastLeaf       common.Hash
}

// Hash of a HistoryCommitment encompasses its height value and its Merkle root.
func (comm HistoryCommitment) Hash() common.Hash {
	return crypto.Keccak256Hash(
		binary.BigEndian.AppendUint64([]byte{}, comm.Height),
		comm.Merkle.Bytes(),
	)
}

// NewHistoryCommitment constructs a commitment from a height and list of leaves.
func NewHistoryCommitment(
	height uint64,
	leaves []common.Hash,
) (HistoryCommitment, error) {
	if len(leaves) == 0 {
		return emptyCommit, errors.New("must commit to at least one leaf")
	}
	tree := ComputeMerkleTree(leaves)
	firstLeafProof, err := GenerateMerkleProof(0, tree)
	if err != nil {
		return emptyCommit, err
	}
	lastLeafProof, err := GenerateMerkleProof(uint64(len(leaves))-1, tree)
	if err != nil {
		return emptyCommit, err
	}
	root, err := MerkleRoot(tree)
	if err != nil {
		return emptyCommit, err
	}
	return HistoryCommitment{
		Merkle:         root,
		Height:         height,
		FirstLeaf:      leaves[0],
		LastLeaf:       leaves[len(leaves)-1],
		FirstLeafProof: firstLeafProof,
		LastLeafProof:  lastLeafProof,
	}, nil
}
