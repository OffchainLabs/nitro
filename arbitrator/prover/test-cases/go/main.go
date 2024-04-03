// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
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

func testCompression(data []byte, doneChan chan struct{}) {
	compressed, err := arbcompress.CompressLevel(data, 0)
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
	doneChan <- struct{}{}
}

const FIELD_ELEMENTS_PER_BLOB = 4096
const BYTES_PER_FIELD_ELEMENT = 32

var BLS_MODULUS, _ = new(big.Int).SetString("52435875175126190479447740508185965837690552500527637822603658699938581184513", 10)

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

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		verified, err := MerkleSample(data, 0)
		if err != nil {
			panic(err)
		}
		if !verified {
			panic("failed to verify proof for Baz")
		}
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		verified, err := MerkleSample(data, 1)
		if err != nil {
			panic(err)
		}
		if !verified {
			panic("failed to verify proof for Baz")
		}
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		verified, err := MerkleSample(data, -1)
		if err != nil {
			if verified {
				panic("succeeded to verify proof invalid")
			}
		}
		wg.Done()
	}()
	wg.Wait()
	println("verified proofs with waitgroup!\n")

	doneChan1 := make(chan struct{})
	doneChan2 := make(chan struct{})
	go testCompression([]byte{}, doneChan1)
	go testCompression([]byte("This is a test string la la la la la la la la la la"), doneChan2)
	<-doneChan2
	<-doneChan1

	println("compression + chan test passed!\n")

	if wavmio.GetInboxPosition() != 0 {
		panic("unexpected inbox pos")
	}
	if wavmio.GetLastBlockHash() != (common.Hash{}) {
		panic("unexpected lastblock hash")
	}
	println("wavmio test passed!\n")

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

	kzgPreimage, err := wavmio.ResolveTypedPreimage(arbutil.EthVersionedHashPreimageType, common.HexToHash("01c277af4074155da57fd0f1065fc8b2e1d475e6639371b7300a2f1fb46296fa"))
	if err != nil {
		panic(fmt.Sprintf("failed to resolve eth versioned hash preimage: %v", err))
	}
	blobLength := FIELD_ELEMENTS_PER_BLOB * BYTES_PER_FIELD_ELEMENT
	if len(kzgPreimage) != blobLength {
		panic(fmt.Sprintf("expected blob length to be %v but got %v", blobLength, len(kzgPreimage)))
	}
	for i := 0; i < FIELD_ELEMENTS_PER_BLOB; i++ {
		hash := sha512.Sum512([]byte(fmt.Sprintf("%v", i)))
		scalar := new(big.Int).SetBytes(hash[:])
		scalar.Mod(scalar, BLS_MODULUS)
		expectedElement := math.U256Bytes(scalar)
		gotElement := kzgPreimage[i*BYTES_PER_FIELD_ELEMENT : (i+1)*BYTES_PER_FIELD_ELEMENT]
		if !bytes.Equal(gotElement, expectedElement) {
			panic(fmt.Sprintf("expected blob element %v to be %v but got %v", i, hex.EncodeToString(expectedElement), hex.EncodeToString(gotElement)))
		}
	}

	println("verified preimage resolution!\n")
}
