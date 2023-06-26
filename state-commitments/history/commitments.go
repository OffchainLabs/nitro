package commitments

import (
	"errors"

	inclusionproofs "github.com/OffchainLabs/challenge-protocol-v2/state-commitments/inclusion-proofs"
	prefixproofs "github.com/OffchainLabs/challenge-protocol-v2/state-commitments/prefix-proofs"
	"github.com/ethereum/go-ethereum/common"
)

var (
	emptyCommit = History{}
)

// History defines a Merkle accumulator over a list of leaves, which
// are understood to be state roots in the goimpl. A history commitment contains
// a "height" value, which can refer to a height of an assertion in the assertions
// tree, or a "step" of WAVM states in a big step or small step subchallenge.
// A commitment contains a Merkle root over the list of leaves, and can optionally
// provide a proof that the last leaf in the accumulator Merkleizes into the
// specified root hash, which is required when verifying challenge creation invariants.
type History struct {
	Height         uint64
	Merkle         common.Hash
	FirstLeaf      common.Hash
	LastLeafProof  []common.Hash
	FirstLeafProof []common.Hash
	LastLeaf       common.Hash
}

// New constructs a commitment from a list of leaves.
func New(leaves []common.Hash) (History, error) {
	if len(leaves) == 0 {
		return emptyCommit, errors.New("must commit to at least one leaf")
	}
	firstLeafProof, err := inclusionproofs.GenerateInclusionProof(leaves, 0)
	if err != nil {
		return emptyCommit, err
	}
	lastLeafProof, err := inclusionproofs.GenerateInclusionProof(leaves, uint64(len(leaves))-1)
	if err != nil {
		return emptyCommit, err
	}
	exp := prefixproofs.NewEmptyMerkleExpansion()
	for _, r := range leaves {
		exp, err = prefixproofs.AppendLeaf(exp, r)
		if err != nil {
			return emptyCommit, err
		}
	}
	root, err := prefixproofs.Root(exp)
	if err != nil {
		return emptyCommit, err
	}
	return History{
		Merkle:         root,
		Height:         uint64(len(leaves) - 1),
		FirstLeaf:      leaves[0],
		LastLeaf:       leaves[len(leaves)-1],
		FirstLeafProof: firstLeafProof,
		LastLeafProof:  lastLeafProof,
	}, nil
}
