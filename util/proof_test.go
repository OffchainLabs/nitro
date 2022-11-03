package util

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"testing"
)

var nullHash = common.Hash{}

func TestMerkleExpansion(t *testing.T) {
	me := NewEmptyMerkleExpansion()
	if me.Root() != nullHash {
		t.Fatal(me.Root())
	}
	compUncompTest(t, me)

	h0 := crypto.Keccak256Hash([]byte{0})
	me, err := me.AppendCompleteSubtree(0, h0)
	if err != nil {
		t.Fatal(err)
	}
	if me.Root() != h0 {
		t.Fatal(me.Root(), h0)
	}
	compUncompTest(t, me)

	h1 := crypto.Keccak256Hash([]byte{1})
	me, err = me.AppendCompleteSubtree(0, h1)
	if err != nil {
		t.Fatal(err)
	}
	if me.Root() != crypto.Keccak256Hash(h0.Bytes(), h1.Bytes()) {
		t.Fatal(me.Root(), h0)
	}
	compUncompTest(t, me)

	me2 := me.Clone()
	h2 := crypto.Keccak256Hash([]byte{2})
	h3 := crypto.Keccak256Hash([]byte{2})
	h23 := crypto.Keccak256Hash(h2.Bytes(), h3.Bytes())
	me, err = me.AppendCompleteSubtree(1, h23)
	if err != nil {
		t.Fatal(err)
	}
	if me.Root() != crypto.Keccak256Hash(me2.Root().Bytes(), h23.Bytes()) {
		t.Fatal(me.Root())
	}
	compUncompTest(t, me)

	me4 := me.Clone()
	me, err = me2.AppendCompleteSubtree(0, h2)
	if err != nil {
		t.Fatal(err)
	}
	me, err = me.AppendCompleteSubtree(0, h3)
	if err != nil {
		t.Fatal(err)
	}
	if me4.Root() != me.Root() {
		t.Fatal(me4.Root(), me.Root())
	}
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
	if err != nil {
		t.Fatal(err)
	}
}

func compUncompTest(t *testing.T, me MerkleExpansion) {
	t.Helper()
	comp, compSz := me.Compact()
	me2, sz := MerkleExpansionFromCompact(comp, compSz)
	if me.Root() != me2.Root() {
		t.Fatal(me.Root(), me2.Root())
	}
	_ = sz
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
		if err != nil {
			t.Fatal(c.lo, c.hi, err)
		}
	}
}

func hashesForUints(lo, hi uint64) []common.Hash {
	ret := []common.Hash{}
	for i := lo; i < hi; i++ {
		ret = append(ret, HashForUint(i))
	}
	return ret
}
