package util

import (
	"testing"

	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

var nullHash = common.Hash{}

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
	proof := GenerateMerkleProof(index, tree)
	require.Equal(t, true, len(proof) > 0)
	computedRoot, err := CalculateRootFromProof(proof, index, leaves[index])
	require.NoError(t, err)
	t.Run("proof verifies", func(t *testing.T) {
		root, err := MerkleRoot(tree)
		require.NoError(t, err)
		require.Equal(t, root, computedRoot)
	})
	t.Run("first leaf proof", func(t *testing.T) {
		index = uint64(0)
		proof = GenerateMerkleProof(index, tree)
		require.Equal(t, true, len(proof) > 0)
		computedRoot, err = CalculateRootFromProof(proof, index, leaves[index])
		require.NoError(t, err)
		root, err := MerkleRoot(tree)
		require.NoError(t, err)
		require.Equal(t, root, computedRoot)
	})
	t.Run("last leaf proof", func(t *testing.T) {
		index = uint64(len(leaves) - 1)
		proof = GenerateMerkleProof(index, tree)
		require.Equal(t, true, len(proof) > 0)
		computedRoot, err = CalculateRootFromProof(proof, index, leaves[index])
		require.NoError(t, err)
		root, err := MerkleRoot(tree)
		require.NoError(t, err)
		require.Equal(t, root, computedRoot)
	})
}

func TestMerkleExpansion(t *testing.T) {
	me := NewEmptyMerkleExpansion()
	require.Equal(t, nullHash, me.Root())
	compUncompTest(t, me)

	h0 := crypto.Keccak256Hash([]byte{0})
	me, err := me.AppendCompleteSubtree(0, h0)
	require.NoError(t, err)
	require.Equal(t, h0, me.Root())
	compUncompTest(t, me)

	h1 := crypto.Keccak256Hash([]byte{1})
	me, err = me.AppendCompleteSubtree(0, h1)
	require.NoError(t, err)
	require.Equal(t, crypto.Keccak256Hash(h0.Bytes(), h1.Bytes()), me.Root())
	compUncompTest(t, me)

	me2 := me.Clone()
	h2 := crypto.Keccak256Hash([]byte{2})
	h3 := crypto.Keccak256Hash([]byte{2})
	h23 := crypto.Keccak256Hash(h2.Bytes(), h3.Bytes())
	me, err = me.AppendCompleteSubtree(1, h23)
	require.NoError(t, err)
	require.Equal(t, crypto.Keccak256Hash(me2.Root().Bytes(), h23.Bytes()), me.Root())
	compUncompTest(t, me)

	me4 := me.Clone()
	me, err = me2.AppendCompleteSubtree(0, h2)
	require.NoError(t, err)
	me, err = me.AppendCompleteSubtree(0, h3)
	require.NoError(t, err)
	require.Equal(t, me.Root(), me4.Root())
	compUncompTest(t, me)

	me2Compact, _ := me2.Compact()
	err = VerifyProof(
		HistoryCommitment{
			Height: 2,
			Merkle: me2.Root(),
		},
		HistoryCommitment{
			Height: 4,
			Merkle: me4.Root(),
		},
		me2Compact,
		h23,
	)
	require.NoError(t, err)
}

func compUncompTest(t *testing.T, me MerkleExpansion) {
	t.Helper()
	comp, compSz := me.Compact()
	me2, _ := MerkleExpansionFromCompact(comp, compSz)
	require.Equal(t, me.Root(), me2.Root())
}

func TestPrefixProofs(t *testing.T) {
	t.Skip("Prefix proofs tested elsewhere, need to investigate off by one")
	for _, c := range []struct {
		lo uint64
		hi uint64
	}{
		{0, 1},
		{0, 2},
		{1, 2},
		{1, 3},
		{1, 13},
		{17, 39820},
		{23, 39820},
		{20, 39823},
	} {
		leaves := hashesForUints(0, c.hi)
		loExp := ExpansionFromLeaves(leaves[:c.lo])
		hiExp := ExpansionFromLeaves(leaves[:c.hi])
		proof := GeneratePrefixProof(c.lo, loExp, leaves[c.lo:c.hi])
		err := VerifyPrefixProof(
			HistoryCommitment{
				Height: c.lo,
				Merkle: loExp.Root(),
			},
			HistoryCommitment{
				Height: c.hi,
				Merkle: hiExp.Root(),
			},
			proof,
		)
		require.NoError(t, err)
	}
}

func TestPrefixProofBackend(t *testing.T) {
	t.Skip("Prefix proofs tested elsewhere, need to investigate off by one")
	for _, c := range []struct {
		lo uint64
		hi uint64
	}{
		{0, 1},
		{0, 2},
		{1, 2},
		{1, 3},
		{1, 13},
		{17, 39820},
		{23, 39820},
		{20, 39823},
	} {
		leaves := hashesForUints(0, c.hi)
		loExp := ExpansionFromLeaves(leaves[:c.lo])
		hiExp := ExpansionFromLeaves(leaves[:c.hi])
		proof := GeneratePrefixProofBackend(
			c.lo,
			loExp,
			c.hi,
			func(lo uint64, hi uint64) (common.Hash, error) {
				return ExpansionFromLeaves(leaves[lo : hi+1]).Root(), nil
			})
		err := VerifyPrefixProof(
			HistoryCommitment{
				Height: c.lo,
				Merkle: loExp.Root(),
			},
			HistoryCommitment{
				Height: c.hi,
				Merkle: hiExp.Root(),
			},
			proof,
		)
		require.NoError(t, err, c.lo, c.hi)
	}
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

//nolint:unused
func hashesForUints(lo, hi uint64) []common.Hash {
	ret := []common.Hash{}
	for i := lo; i < hi; i++ {
		ret = append(ret, HashForUint(i))
	}
	return ret
}
