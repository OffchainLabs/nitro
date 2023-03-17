package util

import (
	"fmt"
	"github.com/OffchainLabs/challenge-protocol-v2/util/inclusion-proofs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHistoryCommitment_LeafProofs(t *testing.T) {
	leaves := make([]common.Hash, 8)
	for i := 0; i < len(leaves); i++ {
		leaves[i] = common.BytesToHash([]byte(fmt.Sprintf("%d", i)))
	}
	height := uint64(8)
	history, err := NewHistoryCommitment(height, leaves)
	require.NoError(t, err)
	require.Equal(t, history.FirstLeaf, leaves[0])
	require.Equal(t, history.LastLeaf, leaves[height-1])

	computed, err := inclusionproofs.CalculateRootFromProof(history.LastLeafProof, history.Height-1, history.LastLeaf)
	require.NoError(t, err)
	require.Equal(t, history.Merkle, computed)
	computed, err = inclusionproofs.CalculateRootFromProof(history.FirstLeafProof, 0, history.FirstLeaf)
	require.NoError(t, err)
	require.Equal(t, history.Merkle, computed)
}
