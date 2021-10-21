//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"encoding/binary"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/merkleBuilder"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"testing"
)

func TestEmptyBuilder(t *testing.T) {
	b := initializedMerkleBuilderForTesting()
	if b.Root() != (common.Hash{}) {
		t.Fatal()
	}
}

func TestBuilder1(t *testing.T) {
	b := initializedMerkleBuilderForTesting()
	if b.Root() != (common.Hash{}) {
		t.Fatal()
	}

	itemHash := pseudorandomForTesting(0)
	b.Append(itemHash)
	if b.Size() != 1 {
		t.Fatal()
	}
	if b.Root() != itemHash {
		t.Fatal()
	}
}

func TestBuilder3(t *testing.T) {
	b := initializedMerkleBuilderForTesting()
	if b.Root() != (common.Hash{}) {
		t.Fatal()
	}

	itemHash0 := pseudorandomForTesting(0)
	itemHash1 := pseudorandomForTesting(1)
	itemHash2 := pseudorandomForTesting(2)

	b.Append(itemHash0)
	b.Append(itemHash1)
	b.Append(itemHash2)

	if b.Size() != 3 {
		t.Fatal()
	}

	expectedHash := crypto.Keccak256(
		crypto.Keccak256(itemHash0.Bytes(), itemHash1.Bytes()),
		crypto.Keccak256(itemHash2.Bytes(), make([]byte, 32)),
	)
	if b.Root() != common.BytesToHash(expectedHash) {
		t.Fatal()
	}
}

func TestBuilder4(t *testing.T) {
	b := initializedMerkleBuilderForTesting()
	if b.Root() != (common.Hash{}) {
		t.Fatal()
	}

	itemHash0 := pseudorandomForTesting(0)
	itemHash1 := pseudorandomForTesting(1)
	itemHash2 := pseudorandomForTesting(2)
	itemHash3 := pseudorandomForTesting(3)

	b.Append(itemHash0)
	b.Append(itemHash1)
	b.Append(itemHash2)
	b.Append(itemHash3)

	if b.Size() != 4 {
		t.Fatal()
	}

	expectedHash := crypto.Keccak256(
		crypto.Keccak256(itemHash0.Bytes(), itemHash1.Bytes()),
		crypto.Keccak256(itemHash2.Bytes(), itemHash3.Bytes()),
	)
	if b.Root() != common.BytesToHash(expectedHash) {
		t.Fatal()
	}
}

func initializedMerkleBuilderForTesting() *merkleBuilder.MerkleBuilder {
	sto := storage.NewMemoryBacked()
	merkleBuilder.InitializeMerkleBuilder(sto)
	return merkleBuilder.OpenMerkleBuilder(sto)
}

func pseudorandomForTesting(x uint64) common.Hash {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], x)
	return crypto.Keccak256Hash(buf[:])
}
