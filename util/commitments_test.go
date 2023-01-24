package util

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestHistoryCommitment(t *testing.T) {
	hashes := []common.Hash{
		common.BytesToHash([]byte{1}),
		common.BytesToHash([]byte{2}),
		common.BytesToHash([]byte{3}),
	}

	hiHeight := uint64(3)
	hi := ExpansionFromLeaves(hashes[:hiHeight])
	hiCommit := HistoryCommitment{
		Height: hiHeight,
		Merkle: hi.Root(),
	}

	loHeight := uint64(2)
	lo := ExpansionFromLeaves(hashes[:loHeight])
	loCommit := HistoryCommitment{
		Height: 2,
		Merkle: lo.Root(),
	}
	lastElem := hashes[len(hashes)-1]
	proof := GeneratePrefixProof(loHeight, lo, []common.Hash{lastElem})
	err := VerifyPrefixProof(loCommit, hiCommit, proof)
	require.NoError(t, err)

	constructedCommit, err := NewHistoryCommitment(
		hiHeight,
		hashes,
		WithLastElementProof(
			loHeight,
			hashes,
		),
	)
	require.NoError(t, err)
	require.False(t, constructedCommit.LastLeafPrefix.IsNone())
	err = VerifyPrefixProof(
		constructedCommit.LastLeafPrefix.Unwrap(),
		constructedCommit,
		constructedCommit.LastLeafProof,
	)
	require.NoError(t, err)
}
