// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/common"
	merkletree "github.com/wealdtech/go-merkletree"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/wavmio"
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

	checkPreimage := func(ty arbutil.PreimageType, hash common.Hash) {
		preimage, err := wavmio.ResolveTypedPreimage(ty, hash)
		if err != nil {
			panic(fmt.Sprintf("failed to resolve preimage of type %v: %v", ty, err))
		}
		if !bytes.Equal(preimage, []byte("hello world")) {
			panic(fmt.Sprintf("got wrong preimage of type %v: %v", ty, hex.EncodeToString(preimage)))
		}
	}

	checkPreimage(arbutil.Keccak256PreimageType, common.HexToHash("47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"))
	checkPreimage(arbutil.Sha2_256PreimageType, common.HexToHash("b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"))

	println("verified preimage resolution!\n")
}
