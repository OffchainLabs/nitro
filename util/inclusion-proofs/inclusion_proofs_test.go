package inclusionproofs

import (
	"fmt"
	"github.com/OffchainLabs/challenge-protocol-v2/util/prefix-proofs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMerkleProof(t *testing.T) {
	leaves := make([]common.Hash, 10)
	for i := 0; i < len(leaves); i++ {
		leaves[i] = common.BytesToHash([]byte(fmt.Sprintf("%d", i)))
	}
	var tree [][]common.Hash
	t.Run("pads to power of two", func(t *testing.T) {
		tree = ComputeMerkleTree(leaves)
		require.Equal(t, 16, len(tree[0]))
	})
	t.Run("generates a tree with the correct height", func(t *testing.T) {
		// 4 levels + the root.
		require.Equal(t, 5, len(tree))
	})
	index := uint64(3)
	proof, err := GenerateMerkleProof(index, tree)
	require.NoError(t, err)
	require.Equal(t, true, len(proof) > 0)
	computedRoot, err := CalculateRootFromProof(proof, index, leaves[index])
	require.NoError(t, err)
	t.Run("proof verifies", func(t *testing.T) {
		root, err2 := MerkleRoot(tree)
		require.NoError(t, err2)
		require.Equal(t, root, computedRoot)
	})
	t.Run("first leaf proof", func(t *testing.T) {
		index = uint64(0)
		proof, err = GenerateMerkleProof(index, tree)
		require.NoError(t, err)
		require.Equal(t, true, len(proof) > 0)
		computedRoot, err = CalculateRootFromProof(proof, index, leaves[index])
		require.NoError(t, err)
		root, err3 := MerkleRoot(tree)
		require.NoError(t, err3)
		require.Equal(t, root, computedRoot)
	})
	t.Run("last leaf proof", func(t *testing.T) {
		index = uint64(len(leaves) - 1)
		proof, err = GenerateMerkleProof(index, tree)
		require.NoError(t, err)
		require.Equal(t, true, len(proof) > 0)
		computedRoot, err = CalculateRootFromProof(proof, index, leaves[index])
		require.NoError(t, err)
		root, err := MerkleRoot(tree)
		require.NoError(t, err)
		require.Equal(t, root, computedRoot)
	})
}

func TestMerkleProofExpansionEquivalence(t *testing.T) {
	leaves := make([]common.Hash, 4)
	for i := 0; i < len(leaves); i++ {
		leaves[i] = common.BytesToHash([]byte(fmt.Sprintf("%d", i)))
	}
	tree := ComputeMerkleTree(leaves)
	index := uint64(0)
	proof, err := GenerateMerkleProof(index, tree)
	require.NoError(t, err)
	computedRoot, err := CalculateRootFromProof(proof, index, leaves[index])
	require.NoError(t, err)
	root, err := MerkleRoot(tree)
	require.NoError(t, err)
	require.Equal(t, root, computedRoot)

	exp, err := prefixproofs.ExpansionFromLeaves(leaves)
	require.NoError(t, err)
	require.Equal(t, root, prefixproofs.Root(exp))
}

func Test_isPowerOfTwo(t *testing.T) {
	for _, tt := range []struct {
		num  uint64
		want bool
	}{
		{0, false},
		{1, true},
		{2, true},
		{3, false},
		{4, true},
		{100, false},
		{1 << 32, true},
		{1<<32 + 1, false},
	} {
		require.Equal(t, tt.want, isPowerOfTwo(tt.num))
	}
}
