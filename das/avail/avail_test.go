package avail

import (
	"encoding/hex"
	"log"
	"testing"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/common"
)

var dataProof DataProof = DataProof{
	Roots: TxDataRoot{
		DataRoot:   hexToHash("17a9a988843e9a384779474e486d2e911e62a8ddb3f605ec4a1c245c42dfecf8"),
		BlobRoot:   hexToHash("7ba9b91957bb73f4284f93b65a3d7f0a53530cfb3a0fe53b1253ba0d1995248f"),
		BridgeRoot: hexToHash("0000000000000000000000000000000000000000000000000000000000000000"),
	},
	Proof: []types.Hash{
		hexToHash("5b7b99083c32347e2a2b6b5f54087ca93bd6a64471f774e1adb04aca57ef4d58"),
		hexToHash("a0a3591c547a0443f34bfc1513fee5dc717d42ab6e8bdaf3597533fc95851ae9"),
		hexToHash("8115d2411c2a53640c8801cc89e67d6fe49b30c14288094f85b408c1ff589b18"),
	},
	NumberOfLeaves: 8,
	LeafIndex:      0,
	Leaf:           hexToHash("b71373eb01c940a02447728c5708ae02b443e525b0a98ba42b143189fad2ab11"),
}

func TestMarshallAndUnmarshalBlobPointer(t *testing.T) {
	extrinsicIndex := 1
	blockHeight := 2024

	// Create ProofResponse
	res := ProofResponse{
		DataProof: dataProof,
		Message:   nil, // No message provided
	}
	var leafProof [][32]byte
	for _, hash := range res.DataProof.Proof {
		var byte32Array [32]byte
		copy(byte32Array[:], hash[:])
		leafProof = append(leafProof, byte32Array)
	}
	blobProof := BlobProof{DataRoot: res.DataProof.Roots.DataRoot, BlobRoot: res.DataProof.Roots.BlobRoot, BridgeRoot: res.DataProof.Roots.BridgeRoot, LeafProof: leafProof, NumberOfLeaves: res.DataProof.NumberOfLeaves, LeafIndex: res.DataProof.LeafIndex, Leaf: res.DataProof.Leaf}

	var blobPointer BlobPointer = BlobPointer{
		uint32(blockHeight),
		uint32(extrinsicIndex),
		common.HexToHash("97a3dacf2a1bfc09eb047e4194084b021fa949cb9b660e1f94d484c070e154f5"),
		common.HexToHash("b71373eb01c940a02447728c5708ae02b443e525b0a98ba42b143189fad2ab11"),
		blobProof,
	}

	data, err := blobPointer.MarshalToBinary()
	if err != nil {
		t.Fatalf("unable to marshal blobPointer to binary, err=%v", err)
	}
	t.Logf("%x", data)

	var newBlobPointer = BlobPointer{}
	if err := newBlobPointer.UnmarshalFromBinary(data[1:]); err != nil {
		t.Fatalf("unable to unmarhal blobPoiter from binary, err=%v", err)
	}

	t.Logf("%+v", newBlobPointer)
}

// Helper function to convert a hex string to a types.Hash
func hexToHash(hexStr string) types.Hash {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		log.Fatalf("Failed to decode hex string: %v", err)
	}
	return types.NewHash(bytes)
}
