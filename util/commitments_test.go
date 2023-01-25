package util

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestHistoryCommitment_LastLeafProof(t *testing.T) {
	hashes := []common.Hash{
		common.BytesToHash([]byte{10}),
		common.BytesToHash([]byte{11}),
		common.BytesToHash([]byte{12}),
	}
	commit, err := NewHistoryCommitment(
		12,
		hashes,
		WithLastElementProof(hashes),
	)
	require.NoError(t, err)

	err = VerifyPrefixProof(
		commit.LastLeafPrefix.Unwrap(),
		commit.Normalized().Unwrap(),
		commit.LastLeafProof,
	)
	require.NoError(t, err)
}

func TestHistoryCommitment(t *testing.T) {
	hashes := []common.Hash{
		common.BytesToHash([]byte{10}),
		common.BytesToHash([]byte{11}),
		common.BytesToHash([]byte{12}),
	}

	hiHeight := uint64(3)
	hiExp := ExpansionFromLeaves(hashes)
	hiCommit := HistoryCommitment{
		Height: hiHeight,
		Merkle: hiExp.Root(),
	}

	lo := uint64(len(hashes) - 1)
	lower := hashes[:lo]
	loExp := ExpansionFromLeaves(lower)
	loCommit := HistoryCommitment{
		Height: lo,
		Merkle: loExp.Root(),
	}
	lastElem := hashes[len(hashes)-1]
	proof := GeneratePrefixProof(lo, loExp, []common.Hash{lastElem})
	err := VerifyPrefixProof(loCommit, hiCommit, proof)
	require.NoError(t, err)
}
