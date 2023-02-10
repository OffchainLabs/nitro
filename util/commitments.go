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
	LastLeaf       common.Hash
	LastLeafPrefix Option[HistoryCommitment]
	normalized     Option[HistoryCommitment]
}

// Hash of a HistoryCommitment encompasses its height value and its Merkle root.
func (comm HistoryCommitment) Hash() common.Hash {
	return crypto.Keccak256Hash(
		binary.BigEndian.AppendUint64([]byte{}, comm.Height),
		comm.Merkle.Bytes(),
	)
}

// Normalized returns a commitment that has its height
// and Merkle expansion normalized to the number of leaves it has
// rather than the absolute height. For example, if a commitment claims
// height 100, but only has 3 leaves, the normalized version will
// return a height of 3. This is useful for proving last leaf prefix
// proofs.
func (comm HistoryCommitment) Normalized() Option[HistoryCommitment] {
	return comm.normalized
}

// CommitOpt defines a functional option for constructing HistoryCommitments.
type CommitOpt func(c *HistoryCommitment) error

// WithLastElementProof allows HistoryCommitment creation to optionally
// include a prefix proof of the last element in the commitment. This is useful
// for asserting the "last leaf" of a commitment verifies against the Merkle
// root contained within the commitment.
func WithLastElementProof(
	leaves []common.Hash,
) CommitOpt {
	return func(c *HistoryCommitment) error {
		if len(leaves) == 0 {
			return errors.New("must commit to at least one leaf")
		}
		lo := uint64(len(leaves) - 1)
		loExp := ExpansionFromLeaves(leaves[:lo])
		loCommit := HistoryCommitment{
			Height: lo,
			Merkle: loExp.Root(),
		}
		lastLeaf := leaves[len(leaves)-1]
		proof := GeneratePrefixProof(lo, loExp, []common.Hash{lastLeaf})

		hi := uint64(len(leaves))
		hiExp := ExpansionFromLeaves(leaves)
		hiCommit := HistoryCommitment{
			Height: hi,
			Merkle: hiExp.Root(),
		}
		if err := VerifyPrefixProof(loCommit, hiCommit, proof); err != nil {
			return err
		}
		c.FirstLeaf = leaves[0]
		c.LastLeafProof = proof
		c.LastLeaf = lastLeaf
		c.LastLeafPrefix = Some(loCommit)
		c.normalized = Some(hiCommit)
		return nil
	}
}

// NewHistoryCommitment constructs a commitment from a height and list of leaves.
func NewHistoryCommitment(
	height uint64,
	leaves []common.Hash,
	opts ...CommitOpt,
) (HistoryCommitment, error) {
	if len(leaves) == 0 {
		return emptyCommit, errors.New("must commit to at least one leaf")
	}
	exp := ExpansionFromLeaves(leaves)
	h := HistoryCommitment{
		Merkle: exp.Root(),
		Height: height,
	}
	for _, o := range opts {
		if err := o(&h); err != nil {
			return emptyCommit, err
		}
	}
	return h, nil
}
