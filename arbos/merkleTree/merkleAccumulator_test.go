//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package merkleTree

import (
	"encoding/binary"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"testing"
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
}

func TestAccumulator1(t *testing.T) {
	acc := initializedMerkleAccumulatorForTesting()
	if acc.Root() != (common.Hash{}) {
		t.Fatal()
	}
	mt := NewEmptyMerkleTree()

	itemHash := pseudorandomForTesting(0)
	acc.Append(itemHash)
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

	acc.Append(itemHash0)
	mt = mt.Append(itemHash0)
	acc.Append(itemHash1)
	mt = mt.Append(itemHash1)
	acc.Append(itemHash2)
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

	acc.Append(itemHash0)
	mt = mt.Append(itemHash0)
	acc.Append(itemHash1)
	mt = mt.Append(itemHash1)
	acc.Append(itemHash2)
	mt = mt.Append(itemHash2)
	acc.Append(itemHash3)
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
	}
}
