package util

import (
	"encoding/binary"

	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	emptyCommit = HistoryCommitment{}
)

// HistoryCommitment defines a Merkle accumulator over a list of leaves, which
// are understood to be state roots in the protocol. A history commitment contains
// a "height" value, which can refer to a height of an assertion in the assertions
// tree, or a "step" of WAVM states in a big step or small step subchallenge.
// A commitment contains a Merkle root over the list of leaves, and can optionally
// provide a proof that the last leaf in the accumulator Merkleizes into the
// specified root hash, which is required when verifying challenge creation invariants.
type HistoryCommitment struct {
	Height         uint64
	Range          uint64
	Merkle         common.Hash
	LastLeafProof  []common.Hash
	LastLeaf       common.Hash
	LastLeafPrefix Option[HistoryCommitment]
}

// Hash of a HistoryCommitment encompasses its height value and its Merkle root.
func (comm HistoryCommitment) Hash() common.Hash {
	return crypto.Keccak256Hash(
		binary.BigEndian.AppendUint64([]byte{}, comm.Height),
		comm.Merkle.Bytes(),
	)
}

// CommitOpt defines a functional option for constructing HistoryCommitments.
type CommitOpt func(c *HistoryCommitment) error

// WithLastElementProof allows HistoryCommitment creation to optionally
// include a prefix proof of the last element in the commitment. This is useful
// for asserting the "last leaf" of a commitment verifies agianst the Merkle
// root contained within the commitment.
//
// It requires specifying the height of the penultimate element and the
// slice of leaves as function arguments.
func WithLastElementProof(
	leaves []common.Hash,
) CommitOpt {
	return func(c *HistoryCommitment) error {
		if len(leaves) == 0 {
			return errors.New("must commit to at least one leaf")
		}
		elems := leaves[:c.Height]
		for _, el := range elems {
			fmt.Printf("%#x\n", el)
		}
		fmt.Println(c.Height - 1)
		lo := ExpansionFromLeaves(leaves[:c.Height])
		loCommit := HistoryCommitment{
			Height: c.Height - 1,
			Merkle: lo.Root(),
		}
		lastLeaf := leaves[len(leaves)-1]
		fmt.Printf("%#x\n", lastLeaf)

		proof := GeneratePrefixProof(c.Height-1, lo, []common.Hash{lastLeaf})
		if err := VerifyPrefixProof(loCommit, *c, proof); err != nil {
			return err
		}
		c.LastLeafProof = proof
		c.LastLeaf = lastLeaf
		c.LastLeafPrefix = Some(loCommit)
		return nil
	}
}

// NewHistoryCommitment constructs a commitment from a height and list of leaves.
func NewHistoryCommitment(
	height,
	heightRange uint64,
	leaves []common.Hash,
	opts ...CommitOpt,
) (HistoryCommitment, error) {
	if len(leaves) == 0 {
		return emptyCommit, errors.New("must commit to at least one leaf")
	}
	if height > uint64(len(leaves)) {
		return emptyCommit, errors.New("height out of range")
	}
	exp := ExpansionFromLeaves(leaves)
	h := HistoryCommitment{
		Merkle: exp.Root(),
		Height: height,
		Range:  heightRange,
	}
	for _, o := range opts {
		if err := o(&h); err != nil {
			return emptyCommit, err
		}
	}
	return h, nil
}
