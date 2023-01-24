package util

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/prysmaticlabs/prysm/v3/container/trie"
	"github.com/stretchr/testify/require"
	"math"
)

var nullHash = common.Hash{}

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

func TestMerkleExpansion_AgainstPrysmMerkleTrie(t *testing.T) {
	hashes := []common.Hash{
		common.BytesToHash([]byte{1}),
		common.BytesToHash([]byte{2}),
	}
	hashesBytes := make([][]byte, len(hashes))
	for i, h := range hashes {
		var tmp [32]byte
		copy(tmp[:], h[:])
		hashed := crypto.Keccak256Hash(tmp[:])
		hashesBytes[i] = hashed[:]
	}
	for _, h := range hashesBytes {
		t.Logf("%#x got hashes bytes", h)
	}
	resulting := crypto.Keccak256Hash(hashesBytes[0], hashesBytes[1])
	t.Logf("%#x resulting", resulting)

	depth := uint64(math.Ceil(math.Log2(float64(len(hashes)))))

	tr, err := trie.GenerateTrieFromItems(hashesBytes, depth)
	require.NoError(t, err)

	lastIdx := len(hashesBytes) - 1
	proof, err := tr.MerkleProof(lastIdx)
	require.NoError(t, err)

	proofHashes := make([]common.Hash, len(proof))
	for i := 0; i < len(proof); i++ {
		proofHashes[i] = common.BytesToHash(proof[i][:])
	}

	want, err := tr.HashTreeRoot()
	require.NoError(t, err)
	t.Logf("HTR gives %#x", want)

	exp := ExpansionFromLeaves(hashes)
	t.Logf("%#x and %#x\n", exp[0], exp[1])

	got := exp.Root()
	require.Equal(t, want[:], got[:], "mismatch expansion root")
}

func compUncompTest(t *testing.T, me MerkleExpansion) {
	t.Helper()
	comp, compSz := me.Compact()
	me2, _ := MerkleExpansionFromCompact(comp, compSz)
	require.Equal(t, me.Root(), me2.Root())
}

func TestMerkleProof(t *testing.T) {
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
		require.NoError(t, err, c.lo, c.hi)
	}
}

func TestMerkleProofBackend(t *testing.T) {
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

func hashesForUints(lo, hi uint64) []common.Hash {
	ret := []common.Hash{}
	for i := lo; i < hi; i++ {
		ret = append(ret, HashForUint(i))
	}
	return ret
}
