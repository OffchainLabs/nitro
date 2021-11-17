//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package merkletree

import (
	"bytes"
	"encoding/binary"
	"github.com/offchainlabs/arbstate/arbos/merkleAccumulator"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestEmptyAccumulator(t *testing.T) {
	acc := initializedMerkleAccumulatorForTesting()
	if acc.Root() != (common.Hash{}) {
		t.Fatal()
	}
	mt := NewEmptyMerkleTree()
	if acc.Root() != mt.Hash() {
		t.Fatal()
	}
	testAllSummarySizes(mt, t)
	testSerDe(mt, t)
}

func TestAccumulator1(t *testing.T) {
	acc := initializedMerkleAccumulatorForTesting()
	if acc.Root() != (common.Hash{}) {
		t.Fatal()
	}
	mt := NewEmptyMerkleTree()

	itemHash := pseudorandomForTesting(0)
	_ = acc.Append(itemHash)
	if acc.Size() != 1 {
		t.Fatal()
	}
	mt = mt.Append(itemHash)
	if mt.Size() != 1 {
		t.Fatal(mt.Size())
	}
	if acc.Root() != itemHash {
		t.Fatal()
	}
	if acc.Root() != mt.Hash() {
		t.Fatal()
	}
	testAllSummarySizes(mt, t)
	testSerDe(mt, t)
}

func TestAccumulator3(t *testing.T) {
	acc := initializedMerkleAccumulatorForTesting()
	if acc.Root() != (common.Hash{}) {
		t.Fatal()
	}
	mt := NewEmptyMerkleTree()

	itemHash0 := pseudorandomForTesting(0)
	itemHash1 := pseudorandomForTesting(1)
	itemHash2 := pseudorandomForTesting(2)

	_ = acc.Append(itemHash0)
	mt = mt.Append(itemHash0)
	_ = acc.Append(itemHash1)
	mt = mt.Append(itemHash1)
	_ = acc.Append(itemHash2)
	mt = mt.Append(itemHash2)

	if acc.Size() != 3 {
		t.Fatal()
	}
	if mt.Size() != 3 {
		t.Fatal()
	}

	expectedHash := crypto.Keccak256(
		crypto.Keccak256(itemHash0.Bytes(), itemHash1.Bytes()),
		crypto.Keccak256(itemHash2.Bytes(), make([]byte, 32)),
	)
	if acc.Root() != common.BytesToHash(expectedHash) {
		t.Fatal()
	}
	if acc.Root() != mt.Hash() {
		t.Fatal()
	}
	testAllSummarySizes(mt, t)
	testSerDe(mt, t)
}

func TestAccumulator4(t *testing.T) {
	acc := initializedMerkleAccumulatorForTesting()
	if acc.Root() != (common.Hash{}) {
		t.Fatal()
	}
	mt := NewEmptyMerkleTree()

	itemHash0 := pseudorandomForTesting(0)
	itemHash1 := pseudorandomForTesting(1)
	itemHash2 := pseudorandomForTesting(2)
	itemHash3 := pseudorandomForTesting(3)

	_ = acc.Append(itemHash0)
	mt = mt.Append(itemHash0)
	_ = acc.Append(itemHash1)
	mt = mt.Append(itemHash1)
	_ = acc.Append(itemHash2)
	mt = mt.Append(itemHash2)
	_ = acc.Append(itemHash3)
	mt = mt.Append(itemHash3)

	if acc.Size() != 4 {
		t.Fatal()
	}
	if mt.Size() != 4 {
		t.Fatal()
	}

	expectedHash := crypto.Keccak256(
		crypto.Keccak256(itemHash0.Bytes(), itemHash1.Bytes()),
		crypto.Keccak256(itemHash2.Bytes(), itemHash3.Bytes()),
	)
	if acc.Root() != common.BytesToHash(expectedHash) {
		t.Fatal()
	}
	if acc.Root() != mt.Hash() {
		t.Fatal()
	}
	testAllSummarySizes(mt, t)
	testSerDe(mt, t)
}

const consistencyProofTestSize = 14

func TestConsistencyProofs(t *testing.T) {
	leaves := []common.Hash{ }
	trees := []MerkleTree{ NewEmptyMerkleTree() }
	accs := []*merkleAccumulator.MerkleAccumulator{ merkleAccumulator.NewNonpersistentMerkleAccumulator() }
	for i := 1; i < consistencyProofTestSize; i++ {
		newLeaf := pseudorandomForTesting(uint64(i))
		leaves = append(leaves, newLeaf)
		trees = append(trees, trees[i-1].Append(newLeaf))
		newAcc := accs[i-1].NonPersistentClone()
		newAcc.Append(newLeaf)
		accs = append(accs, newAcc)
	}
	finalTree := trees[consistencyProofTestSize-1]

	for i := 0; i < consistencyProofTestSize; i++ {
		for j := i+1; j < consistencyProofTestSize; j++ {
			proof := finalTree.ConsistencyProof(uint64(i), uint64(j))
			if ! accs[i].VerifyConsistencyProof(accs[j].Root(), proof) {
				t.Fatal(i, j, proof)
			}

			if finalTree.Truncate(uint64(i)).Hash() != trees[i].Hash() {
				t.Fatal(i)
			}
			if finalTree.Truncate(uint64(j)).Hash() != trees[j].Hash() {
				t.Fatal(j)
			}
			conciseProof := MakeConciseConsistencyProof(finalTree, uint64(i), uint64(j))
			if !conciseProof.Verify() {
				t.Fatal(i, j, conciseProof)
			}
		}
	}
}

func testAllSummarySizes(tree MerkleTree, t *testing.T) {
	for i := uint64(1); i <= tree.Size(); i++ {
		sum := tree.SummarizeUpTo(i)
		if tree.Hash() != sum.Hash() {
			t.Fatal()
		}
		if tree.Size() != sum.Size() {
			t.Fatal()
		}
		if tree.Capacity() != sum.Capacity() {
			t.Fatal()
		}
		testSerDe(sum, t)
	}
}

func testSerDe(tree MerkleTree, t *testing.T) {
	var wr bytes.Buffer
	if err := tree.Serialize(&wr); err != nil {
		t.Fatal(err)
	}
	rd := bytes.NewReader(wr.Bytes())
	result, err := NewMerkleTreeFromReader(rd)
	if err != nil {
		t.Fatal(err)
	}
	if tree.Hash() != result.Hash() {
		t.Fatal()
	}
}

func pseudorandomForTesting(x uint64) common.Hash {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], x)
	return crypto.Keccak256Hash(buf[:])
}
