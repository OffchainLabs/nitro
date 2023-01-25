package util

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestYay(t *testing.T) {
	hashes := []common.Hash{
		common.BytesToHash([]byte{1}),
		common.BytesToHash([]byte{2}),
		common.BytesToHash([]byte{3}),
		common.BytesToHash([]byte{4}),
	}
	_, err := NewHistoryCommitment(
		3,
		hashes,
		WithLastElementProof(hashes),
	)
	require.NoError(t, err)
}

func TestHistoryCommitment(t *testing.T) {
	hashes := []common.Hash{
		common.BytesToHash([]byte{1}),
		common.BytesToHash([]byte{2}),
		common.BytesToHash([]byte{3}),
	}

	hiHeight := uint64(3)
	for _, h := range hashes {
		t.Logf("%#x", h)
	}
	hi := ExpansionFromLeaves(hashes)
	hiCommit := HistoryCommitment{
		Height: hiHeight,
		Merkle: hi.Root(),
	}

	loHeight := uint64(2)
	for _, h := range hashes[:len(hashes)-1] {
		t.Logf("%#x", h)
	}
	lo := ExpansionFromLeaves(hashes[:len(hashes)-1])
	loCommit := HistoryCommitment{
		Height: loHeight,
		Merkle: lo.Root(),
	}
	lastElem := hashes[len(hashes)-1]
	t.Logf("%#x", lastElem)
	proof := GeneratePrefixProof(loHeight, lo, []common.Hash{lastElem})
	err := VerifyPrefixProof(loCommit, hiCommit, proof)
	require.NoError(t, err)

	// constructedCommit, err := NewHistoryCommitment(
	// 	hiHeight,
	// 	hashes,
	// 	WithLastElementProof(hashes),
	// )
	// require.NoError(t, err)
	// require.False(t, constructedCommit.LastLeafPrefix.IsNone())
	// err = VerifyPrefixProof(
	// 	constructedCommit.LastLeafPrefix.Unwrap(),
	// 	constructedCommit,
	// 	constructedCommit.LastLeafProof,
	// )
	// require.NoError(t, err)
}
