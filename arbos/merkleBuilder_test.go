//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"bytes"
	"encoding/binary"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"testing"
)

func TestEmptyBuilder(t *testing.T) {
	b := NewBuilder()
	if b.Root() != (common.Hash{}) {
		t.Fatal()
	}
	serdeBuilder(b, t)
}

func TestBuilder1(t *testing.T) {
	b := NewBuilder()
	if b.Root() != (common.Hash{}) {
		t.Fatal()
	}
	serdeBuilder(b, t)

	itemHash := pseudorandom(0)
	b.Append(itemHash)
	if b.Size() != 1 {
		t.Fatal()
	}
	if b.Root() != itemHash {
		t.Fatal()
	}
	serdeBuilder(b, t)
}

func TestBuilder3(t *testing.T) {
	b := NewBuilder()
	if b.Root() != (common.Hash{}) {
		t.Fatal()
	}
	serdeBuilder(b, t)

	itemHash0 := pseudorandom(0)
	itemHash1 := pseudorandom(1)
	itemHash2 := pseudorandom(2)

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

	serdeBuilder(b, t)
}

func TestBuilder4(t *testing.T) {
	b := NewBuilder()
	if b.Root() != (common.Hash{}) {
		t.Fatal()
	}
	serdeBuilder(b, t)

	itemHash0 := pseudorandom(0)
	itemHash1 := pseudorandom(1)
	itemHash2 := pseudorandom(2)
	itemHash3 := pseudorandom(3)

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

	serdeBuilder(b, t)
}

func serdeBuilder(b *MerkleBuilder, t *testing.T) {
	var buf bytes.Buffer
	err := b.Serialize(&buf)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := NewBuilderFromReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if b.Size() != b2.Size() {
		t.Fatal()
	}
	if b.Root() != b2.Root() {
		t.Fatal()
	}
}

func pseudorandom(x uint64) common.Hash {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], x)
	return crypto.Keccak256Hash(buf[:])
}
