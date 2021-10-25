//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package merkleAccumulator

import (
	"bytes"
	"encoding/binary"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/util/merkletree"
	"testing"
)

func TestEmptyAccumulator(t *testing.T) {
	acc := initializedMerkleAccumulatorForTesting()
	if acc.Root() != (common.Hash{}) {
		t.Fatal()
	}
	mt := merkletree.NewEmptyMerkleTree()
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
	mt := merkletree.NewEmptyMerkleTree()

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
	mt := merkletree.NewEmptyMerkleTree()

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
	mt := merkletree.NewEmptyMerkleTree()

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

func initializedMerkleAccumulatorForTesting() *MerkleAccumulator {
	sto := storage.NewMemoryBacked()
	InitializeMerkleAccumulator(sto)
	return OpenMerkleAccumulator(sto)
}

func pseudorandomForTesting(x uint64) common.Hash {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], x)
	return crypto.Keccak256Hash(buf[:])
}

func testAllSummarySizes(tree merkletree.MerkleTree, t *testing.T) {
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

func testSerDe(tree merkletree.MerkleTree, t *testing.T) {
	var wr bytes.Buffer
	if err := tree.Serialize(&wr); err != nil {
		t.Fatal(err)
	}
	rd := bytes.NewReader(wr.Bytes())
	result, err := merkletree.NewMerkleTreeFromReader(rd)
	if err != nil {
		t.Fatal(err)
	}
	if tree.Hash() != result.Hash() {
		t.Fatal()
	}
}

func TestReconstructFromEvents(t *testing.T) {
	leaves := make([]common.Hash, 13)
	for i := range leaves {
		leaves[i] = pseudorandomForTesting(uint64(i))
	}

	acc := initializedMerkleAccumulatorForTesting()
	events := []EventForTreeBuilding{}

	for i, leaf := range leaves {
		ev := acc.Append(leaf)
		if ev.Level >= uint64(len(events)) {
			events = append(events, *ev)
		} else {
			events[ev.Level] = *ev
		}
		if acc.Root() != acc.ToMerkleTree().Hash() {
			t.Fatal(i, acc.partials)
		}
	}

	if acc.Root() != acc.ToMerkleTree().Hash() {
		t.Fatal()
	}

	reconstructedAcc := NewNonPersistentMerkleAccumulatorFromEvents(events)
	if reconstructedAcc.Root() != acc.Root() {
		t.Fatal()
	}

	if reconstructedAcc.size != acc.size {
		t.Fatal()
	}
	if reconstructedAcc.numPartials != acc.numPartials {
		t.Fatal()
	}
	for i := uint64(0); i < reconstructedAcc.numPartials; i++ {
		if *reconstructedAcc.getPartial(i) != *acc.getPartial(i) {
			t.Fatal(i)
		}
	}

	reconstructedTree := reconstructedAcc.ToMerkleTree()
	if reconstructedAcc.Root() != reconstructedTree.Hash() {
		t.Fatal()
	}

	reconstructedTree = NewMerkleTreeFromEvents(events)
	if reconstructedTree.Hash() != acc.Root() {
		t.Fatal()
	}
}

func TestProofForNext(t *testing.T) {
	leaves := make([]common.Hash, 13)
	for i := range leaves {
		leaves[i] = pseudorandomForTesting(uint64(i))
	}

	acc := initializedMerkleAccumulatorForTesting()
	for i, leaf := range leaves {
		proof := acc.ProofForNext(leaf)
		if proof == nil {
			t.Fatal(i)
		}
		if proof.LeafHash != leaf {
			t.Fatal(i)
		}
		if !proof.IsCorrect() {
			t.Fatal(proof)
		}
		acc.Append(leaf)
		if proof.RootHash != acc.Root() {
			t.Fatal(i)
		}
	}
}
