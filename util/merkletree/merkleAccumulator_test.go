// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package merkletree

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
)

func TestEmptyAccumulator(t *testing.T) {
	acc := initializedMerkleAccumulatorForTesting()
	if root(t, acc) != (common.Hash{}) {
		Fail(t)
	}
	mt := NewEmptyMerkleTree()
	if root(t, acc) != mt.Hash() {
		Fail(t)
	}
	testAllSummarySizes(mt, t)
	testSerDe(mt, t)
}

func TestAccumulator1(t *testing.T) {
	acc := initializedMerkleAccumulatorForTesting()
	if root(t, acc) != (common.Hash{}) {
		Fail(t)
	}
	mt := NewEmptyMerkleTree()

	itemHash := pseudorandomForTesting(0)
	accAppend(t, acc, itemHash)
	if size(t, acc) != 1 {
		Fail(t)
	}
	mt = mt.Append(itemHash)
	if mt.Size() != 1 {
		t.Fatal(mt.Size())
	}
	if root(t, acc) != crypto.Keccak256Hash(itemHash.Bytes()) {
		Fail(t)
	}
	if root(t, acc) != mt.Hash() {
		Fail(t)
	}
	testAllSummarySizes(mt, t)
	testSerDe(mt, t)
}

func TestAccumulator3(t *testing.T) {
	acc := initializedMerkleAccumulatorForTesting()
	if root(t, acc) != (common.Hash{}) {
		Fail(t)
	}
	mt := NewEmptyMerkleTree()

	itemHash0 := pseudorandomForTesting(0)
	itemHash1 := pseudorandomForTesting(1)
	itemHash2 := pseudorandomForTesting(2)

	accAppend(t, acc, itemHash0)
	mt = mt.Append(itemHash0)
	accAppend(t, acc, itemHash1)
	mt = mt.Append(itemHash1)
	accAppend(t, acc, itemHash2)
	mt = mt.Append(itemHash2)

	if size(t, acc) != 3 {
		Fail(t)
	}
	if mt.Size() != 3 {
		Fail(t)
	}

	expectedHash := crypto.Keccak256(
		crypto.Keccak256(crypto.Keccak256(itemHash0.Bytes()), crypto.Keccak256(itemHash1.Bytes())),
		crypto.Keccak256(crypto.Keccak256(itemHash2.Bytes()), make([]byte, 32)),
	)
	if root(t, acc) != common.BytesToHash(expectedHash) {
		Fail(t)
	}
	if root(t, acc) != mt.Hash() {
		Fail(t)
	}
	testAllSummarySizes(mt, t)
	testSerDe(mt, t)
}

func TestAccumulator4(t *testing.T) {
	acc := initializedMerkleAccumulatorForTesting()
	if root(t, acc) != (common.Hash{}) {
		Fail(t)
	}
	mt := NewEmptyMerkleTree()

	itemHash0 := pseudorandomForTesting(0)
	itemHash1 := pseudorandomForTesting(1)
	itemHash2 := pseudorandomForTesting(2)
	itemHash3 := pseudorandomForTesting(3)

	accAppend(t, acc, itemHash0)
	mt = mt.Append(itemHash0)
	accAppend(t, acc, itemHash1)
	mt = mt.Append(itemHash1)
	accAppend(t, acc, itemHash2)
	mt = mt.Append(itemHash2)
	accAppend(t, acc, itemHash3)
	mt = mt.Append(itemHash3)

	if size(t, acc) != 4 {
		Fail(t)
	}
	if mt.Size() != 4 {
		Fail(t)
	}

	expectedHash := crypto.Keccak256(
		crypto.Keccak256(crypto.Keccak256(itemHash0.Bytes()), crypto.Keccak256(itemHash1.Bytes())),
		crypto.Keccak256(crypto.Keccak256(itemHash2.Bytes()), crypto.Keccak256(itemHash3.Bytes())),
	)
	if root(t, acc) != common.BytesToHash(expectedHash) {
		Fail(t)
	}
	if root(t, acc) != mt.Hash() {
		Fail(t)
	}
	testAllSummarySizes(mt, t)
	testSerDe(mt, t)
}

func testAllSummarySizes(tree MerkleTree, t *testing.T) {
	for i := uint64(1); i <= tree.Size(); i++ {
		sum := tree.SummarizeUpTo(i)
		if tree.Hash() != sum.Hash() {
			Fail(t)
		}
		if tree.Size() != sum.Size() {
			Fail(t)
		}
		if tree.Capacity() != sum.Capacity() {
			Fail(t)
		}
		testSerDe(sum, t)
	}
}

func testSerDe(tree MerkleTree, t *testing.T) {
	var wr bytes.Buffer
	if err := tree.Serialize(&wr); err != nil {
		Fail(t, err)
	}
	rd := bytes.NewReader(wr.Bytes())
	result, err := NewMerkleTreeFromReader(rd)
	if err != nil {
		Fail(t, err)
	}
	if tree.Hash() != result.Hash() {
		Fail(t)
	}
}

func pseudorandomForTesting(x uint64) common.Hash {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], x)
	return crypto.Keccak256Hash(buf[:])
}

func accAppend(t *testing.T, acc *merkleAccumulator.MerkleAccumulator, itemHash common.Hash) {
	t.Helper()
	_, err := acc.Append(itemHash)
	Require(t, err)
}

func root(t *testing.T, acc *merkleAccumulator.MerkleAccumulator) common.Hash {
	t.Helper()
	root, err := acc.Root()
	Require(t, err)
	return root
}

func size(t *testing.T, acc *merkleAccumulator.MerkleAccumulator) uint64 {
	t.Helper()
	size, err := acc.Size()
	Require(t, err)
	return size
}
