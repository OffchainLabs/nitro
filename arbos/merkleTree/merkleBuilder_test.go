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

func TestEmptyBuilder(t *testing.T) {
	b := initializedMerkleBuilderForTesting()
	if b.Root() != (common.Hash{}) {
		t.Fatal()
	}
	mt := NewEmptyMerkleTree()
	if b.Root() != mt.Hash() {
		t.Fatal()
	}
	testAllSummarySizes(mt, t)
}

func TestBuilder1(t *testing.T) {
	b := initializedMerkleBuilderForTesting()
	if b.Root() != (common.Hash{}) {
		t.Fatal()
	}
	mt := NewEmptyMerkleTree()

	itemHash := pseudorandomForTesting(0)
	b.Append(itemHash)
	if b.Size() != 1 {
		t.Fatal()
	}
	mt = mt.Append(itemHash)
	if mt.Size() != 1 {
		t.Fatal(mt.Size())
	}
	if b.Root() != itemHash {
		t.Fatal()
	}
	if b.Root() != mt.Hash() {
		t.Fatal()
	}
	testAllSummarySizes(mt, t)
}

func TestBuilder3(t *testing.T) {
	b := initializedMerkleBuilderForTesting()
	if b.Root() != (common.Hash{}) {
		t.Fatal()
	}
	mt := NewEmptyMerkleTree()

	itemHash0 := pseudorandomForTesting(0)
	itemHash1 := pseudorandomForTesting(1)
	itemHash2 := pseudorandomForTesting(2)

	b.Append(itemHash0)
	mt = mt.Append(itemHash0)
	b.Append(itemHash1)
	mt = mt.Append(itemHash1)
	b.Append(itemHash2)
	mt = mt.Append(itemHash2)

	if b.Size() != 3 {
		t.Fatal()
	}
	if mt.Size() != 3 {
		t.Fatal()
	}

	expectedHash := crypto.Keccak256(
		crypto.Keccak256(itemHash0.Bytes(), itemHash1.Bytes()),
		crypto.Keccak256(itemHash2.Bytes(), make([]byte, 32)),
	)
	if b.Root() != common.BytesToHash(expectedHash) {
		t.Fatal()
	}
	if b.Root() != mt.Hash() {
		t.Fatal()
	}
	testAllSummarySizes(mt, t)
}

func TestBuilder4(t *testing.T) {
	b := initializedMerkleBuilderForTesting()
	if b.Root() != (common.Hash{}) {
		t.Fatal()
	}
	mt := NewEmptyMerkleTree()

	itemHash0 := pseudorandomForTesting(0)
	itemHash1 := pseudorandomForTesting(1)
	itemHash2 := pseudorandomForTesting(2)
	itemHash3 := pseudorandomForTesting(3)

	b.Append(itemHash0)
	mt = mt.Append(itemHash0)
	b.Append(itemHash1)
	mt = mt.Append(itemHash1)
	b.Append(itemHash2)
	mt = mt.Append(itemHash2)
	b.Append(itemHash3)
	mt = mt.Append(itemHash3)

	if b.Size() != 4 {
		t.Fatal()
	}
	if mt.Size() != 4 {
		t.Fatal()
	}

	expectedHash := crypto.Keccak256(
		crypto.Keccak256(itemHash0.Bytes(), itemHash1.Bytes()),
		crypto.Keccak256(itemHash2.Bytes(), itemHash3.Bytes()),
	)
	if b.Root() != common.BytesToHash(expectedHash) {
		t.Fatal()
	}
	if b.Root() != mt.Hash() {
		t.Fatal()
	}
	testAllSummarySizes(mt, t)
}

func initializedMerkleBuilderForTesting() *MerkleBuilder {
	sto := storage.NewMemoryBacked()
	InitializeMerkleBuilder(sto)
	return OpenMerkleBuilder(sto)
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
