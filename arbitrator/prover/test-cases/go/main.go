// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package main

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"time"

	merkletree "github.com/wealdtech/go-merkletree"

	"github.com/offchainlabs/nitro/arbcompress"
)

// MerkleSample is an example using the Merkle tree to generate and verify proofs.
func MerkleSample(data [][]byte, toproove int) (bool, error) {
	// Create the tree
	tree, err := merkletree.New(data)
	if err != nil {
		return false, err
	}

	// Fetch the root hash of the tree
	root := tree.Root()

	baz := []byte("yoo")

	if toproove >= 0 {
		baz = data[toproove]
	}

	// Generate a proof for 'Baz'
	proof, err := tree.GenerateProof(baz)
	if err != nil {
		return false, err
	}
	return merkletree.VerifyProof(baz, proof, root)
	// Verify the proof for 'Baz'
}

func testCompression(data []byte) {
	compressed, err := arbcompress.CompressFast(data)
	if err != nil {
		panic(err)
	}
	decompressed, err := arbcompress.Decompress(compressed, len(data)*2+0x100)
	if err != nil {
		panic(err)
	}
	if !bytes.Equal(decompressed, data) {
		panic("data differs after compression / decompression")
	}
}

func main() {
	fmt.Printf("starting executable with %v arg(s): %v\n", len(os.Args), os.Args)
	runtime.GC()
	time.Sleep(time.Second)

	// Data for the tree
	data := [][]byte{
		[]byte("Foo"),
		[]byte("Bar"),
		[]byte("Baz"),
	}

	verified, err := MerkleSample(data, 0)
	if err != nil {
		panic(err)
	}
	if !verified {
		panic("failed to verify proof for Baz")
	}
	verified, err = MerkleSample(data, 1)
	if err != nil {
		panic(err)
	}
	if !verified {
		panic("failed to verify proof for Baz")
	}

	verified, err = MerkleSample(data, -1)
	if err != nil {
		if verified {
			panic("succeded to verify proof invalid")
		}
	}

	println("verified both proofs!\n")

	testCompression([]byte{})
	testCompression([]byte("This is a test string la la la la la la la la la la"))

	println("test compression passed!\n")
}
