package main

import (
	"runtime"
	"time"

	merkletree "github.com/wealdtech/go-merkletree"
)

// Example using the Merkle tree to generate and verify proofs.
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

func main() {
	println("start")
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
}
