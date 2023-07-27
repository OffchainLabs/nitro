// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package prefixproofs

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestMerkleExpansion(t *testing.T) {
	me := NewEmptyMerkleExpansion()

	h0 := crypto.Keccak256Hash([]byte{0})
	me, err := AppendCompleteSubTree(me, 0, h0)
	require.NoError(t, err)
	root, err := Root(me)
	require.NoError(t, err)
	require.Equal(t, h0, root)
	compUncompTest(t, me)

	h1 := crypto.Keccak256Hash([]byte{1})
	me, err = AppendCompleteSubTree(me, 0, h1)
	require.NoError(t, err)
	root, err = Root(me)
	require.NoError(t, err)
	require.Equal(t, crypto.Keccak256Hash(h0.Bytes(), h1.Bytes()), root)
	compUncompTest(t, me)

	me2 := me.Clone()
	h2 := crypto.Keccak256Hash([]byte{2})
	h3 := crypto.Keccak256Hash([]byte{2})
	h23 := crypto.Keccak256Hash(h2.Bytes(), h3.Bytes())
	me, err = AppendCompleteSubTree(me, 1, h23)
	require.NoError(t, err)
	root, err = Root(me)
	require.NoError(t, err)
	root2, err := Root(me2)
	require.NoError(t, err)
	require.Equal(t, crypto.Keccak256Hash(root2.Bytes(), h23.Bytes()), root)
	compUncompTest(t, me)

	me4 := me.Clone()
	me, err = AppendCompleteSubTree(me2, 0, h2)
	require.NoError(t, err)
	me, err = AppendCompleteSubTree(me, 0, h3)
	require.NoError(t, err)
	root, err = Root(me)
	require.NoError(t, err)
	root4, err := Root(me4)
	require.NoError(t, err)
	require.Equal(t, root, root4)
	compUncompTest(t, me)
}

func compUncompTest(t *testing.T, me MerkleExpansion) {
	t.Helper()
	comp, compSz := me.Compact()
	me2, _ := MerkleExpansionFromCompact(comp, compSz)
	root, err := Root(me)
	require.NoError(t, err)
	root2, err := Root(me2)
	require.NoError(t, err)
	require.Equal(t, root, root2)
}
