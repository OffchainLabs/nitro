package util

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type HistoryCommitment struct {
	// The "height" refers to a height of an assertion in the tree, or the "step"
	// in a big step or small step challenge.
	Height uint64
	// The root of a Merkle expansion of leaves.
	Merkle common.Hash
	// The history commitment optionally provides a proof of the last
	// leaf in the Merkle expansion, which can be verified using
	// prefix proofs and is required by the spec in challenge creation.
	LastLeafProof  []common.Hash
	LastLeaf       common.Hash
	LastLeafPrefix Option[HistoryCommitment]
}

func (comm HistoryCommitment) Hash() common.Hash {
	return crypto.Keccak256Hash(binary.BigEndian.AppendUint64([]byte{}, comm.Height), comm.Merkle.Bytes())
}

type CommitOpt func(c *HistoryCommitment)

func WithLastElementProof() CommitOpt {
	return func(c *HistoryCommitment) {

	}
}

func NewHistoryCommitment(
	height uint64,
	leaves []common.Hash,
	opts ...CommitOpt,
) HistoryCommitment {
	h := HistoryCommitment{
		Height: height,
	}
	for _, o := range opts {
		o(&h)
	}
	return h
}
