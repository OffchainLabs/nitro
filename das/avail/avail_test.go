package avail

import (
	"encoding/hex"
	"log"
	"testing"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/common"
)

func TestMarshallAndUnmarshalBlobPointer(t *testing.T) {
	extrinsicIndex := 1
	blockHeight := 2024

	dataProof := DataProof{
		Roots: TxDataRoot{
			DataRoot:   hexToHash("835f5ef98bd9d8cbb41b0786f3f4f7726d54500cb15fc0f4d607d47b419a9a09"),
			BlobRoot:   hexToHash("532168c18a2b1e006ffd66c402a31fc61c82c62e7ec9bd983dc6b87e75f59479"),
			BridgeRoot: hexToHash("0000000000000000000000000000000000000000000000000000000000000000"),
		},
		Proof: []types.Hash{
			hexToHash("4d539f7068ca4f7848f6ff751382936de97c5f36f1119d02b8409b4d5ce908e4"),
			hexToHash("ef44f2bb8a5ac9b38e1e67d7f5defc21afeb2015085de38e82888a2d8a60cfdb"),
			hexToHash("fa1ebbc9251f163a9a4ef4418f4c80765c8fb9d7a709b77946bb846f597ff00a"),
			hexToHash("ae1ecba2081478fa8eddeb64dfbe3e1f16f587bc02a29259796bea9256cf05ab"),
		},
		NumberOfLeaves: 16,
		LeafIndex:      0,
		Leaf:           hexToHash("b71373eb01c940a02447728c5708ae02b443e525b0a98ba42b143189fad2ab11"),
	}

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
		common.HexToHash("97a3dacf2a1bfc09eb047e4194084b021fa949cb9b660e1f94d484c070e154f5"),
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
