package inclusionproofs

import (
	"fmt"
	"testing"

	prefixproofs "github.com/OffchainLabs/challenge-protocol-v2/state-commitments/prefix-proofs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
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
		index = uint64(len(leaves) - 1)
		proof, err = GenerateInclusionProof(leaves, index)
		require.NoError(t, err)
		require.Equal(t, true, len(proof) > 0)
		computedRoot, err = CalculateRootFromProof(proof, index, leaves[index])
		require.NoError(t, err)
		require.Equal(t, root, computedRoot)
	})
	t.Run("Invalid inputs", func(t *testing.T) {
		// Empty tree should fail to generate a proof.
		_, err := GenerateInclusionProof([]common.Hash{}, 0)
		require.Equal(t, ErrInvalidLeaves, err)

		// Index greater than the number of leaves should fail to generate a proof.
		_, err = GenerateInclusionProof(leaves, uint64(len(leaves)))
		require.Equal(t, ErrInvalidLeaves, err)

		// Proof with more than 256 elements should fail to calculate a root...
		_, err = CalculateRootFromProof(make([]common.Hash, 257), 0, common.Hash{})
		require.Equal(t, ErrProofTooLong, err)

		// ... but proof with exactly 256 elements should be OK.
		_, err = CalculateRootFromProof(make([]common.Hash, 256), 0, common.Hash{})
		require.NotEqual(t, ErrProofTooLong, err)
	})
}
