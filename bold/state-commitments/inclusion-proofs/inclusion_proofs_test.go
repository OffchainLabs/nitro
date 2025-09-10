// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package inclusionproofs

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	prefixproofs "github.com/offchainlabs/nitro/bold/state-commitments/prefix-proofs"
	"github.com/offchainlabs/nitro/bold/testing/casttest"
)

func TestInclusionProof(t *testing.T) {
	leaves := make([]common.Hash, 8)
	for i := 0; i < len(leaves); i++ {
		leaves[i] = common.BytesToHash([]byte(fmt.Sprintf("%d", i)))
	}
	index := uint64(0)
	proof, err := GenerateInclusionProof(leaves, index)
	require.NoError(t, err)
	require.Equal(t, true, len(proof) > 0)

	computedRoot, err := CalculateRootFromProof(proof, index, leaves[index])
	require.NoError(t, err)

	exp := prefixproofs.NewEmptyMerkleExpansion()
	for _, r := range leaves {
		exp, err = prefixproofs.AppendLeaf(exp, r)
		require.NoError(t, err)
	}

	root, err := prefixproofs.Root(exp)
	require.NoError(t, err)

	t.Run("proof verifies", func(t *testing.T) {
		require.Equal(t, root, computedRoot)
	})
	t.Run("first leaf proof", func(t *testing.T) {
		index = uint64(0)
		proof, err = GenerateInclusionProof(leaves, index)
		require.NoError(t, err)
		require.Equal(t, true, len(proof) > 0)
		computedRoot, err = CalculateRootFromProof(proof, index, leaves[index])
		require.NoError(t, err)
		require.Equal(t, root, computedRoot)
	})
	t.Run("last leaf proof", func(t *testing.T) {
		index = casttest.ToUint64(t, len(leaves)-1)
		proof, err = GenerateInclusionProof(leaves, index)
		require.NoError(t, err)
		require.Equal(t, true, len(proof) > 0)
		computedRoot, err = CalculateRootFromProof(proof, index, leaves[index])
		require.NoError(t, err)
		require.Equal(t, root, computedRoot)
	})
	t.Run("Invalid inputs", func(t *testing.T) {
		// Empty tree should not generate a proof.
		_, err := GenerateInclusionProof([]common.Hash{}, 0)
		require.Equal(t, ErrInvalidLeaves, err)

		// Index greater than the number of leaves should not generate a proof.
		_, err = GenerateInclusionProof(leaves, uint64(len(leaves)))
		require.Equal(t, ErrInvalidLeaves, err)

		// Proof with more than 256 elements should not calculate a root...
		_, err = CalculateRootFromProof(make([]common.Hash, 257), 0, common.Hash{})
		require.Equal(t, ErrProofTooLong, err)

		// ... but proof with exactly 256 elements should be OK.
		_, err = CalculateRootFromProof(make([]common.Hash, 256), 0, common.Hash{})
		require.NotEqual(t, ErrProofTooLong, err)
	})
}
