// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package history

import (
	"fmt"
	"testing"

	inclusionproofs "github.com/OffchainLabs/bold/state-commitments/inclusion-proofs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestHistoryCommitment_LeafProofs(t *testing.T) {
	leaves := make([]common.Hash, 8)
	for i := 0; i < len(leaves); i++ {
		leaves[i] = common.BytesToHash([]byte(fmt.Sprintf("%d", i)))
	}
	history, err := New(leaves)
	require.NoError(t, err)
	require.Equal(t, history.FirstLeaf, leaves[0])
	require.Equal(t, history.LastLeaf, leaves[len(leaves)-1])

	computed, err := inclusionproofs.CalculateRootFromProof(history.LastLeafProof, history.Height, history.LastLeaf)
	require.NoError(t, err)
	require.Equal(t, history.Merkle, computed)
	computed, err = inclusionproofs.CalculateRootFromProof(history.FirstLeafProof, 0, history.FirstLeaf)
	require.NoError(t, err)
	require.Equal(t, history.Merkle, computed)
}
