package avail

import (
	"encoding/hex"
	"fmt"
)

type BlobProof struct {
	DataRoot [32]byte `json:"dataRoot"`
	// blob root to check proof against, or reconstruct the data root
	BlobRoot [32]byte `json:"blobRoot"`
	// bridge root to check proof against, or reconstruct the data root
	BridgeRoot [32]byte `json:"bridgeRoot"`
	// proof of inclusion of leaf within blob/bridge root
	LeafProof      [][32]byte `json:"leafProof"`
	NumberOfLeaves uint32     // Change to uint32 to match Rust u32
	// index of the leaf in the blob/bridge root tree
	LeafIndex uint32 `json:"leafIndex"`
	// leaf being proven
	Leaf [32]byte `json:"leaf"`
}

// Method to convert BlobProof to string
func (bp *BlobProof) String() string {
	return fmt.Sprintf(
		"DataRoot: %s,  BlobRoot: %s,  BridgeRoot: %s,  LeafProof: %s,  NumberOfLeaves: %d,  LeafIndex: %d,  Leaf: %s",
		hex.EncodeToString(bp.DataRoot[:]),
		hex.EncodeToString(bp.BlobRoot[:]),
		hex.EncodeToString(bp.BridgeRoot[:]),
		formatLeafProof(bp.LeafProof),
		bp.NumberOfLeaves,
		bp.LeafIndex,
		hex.EncodeToString(bp.Leaf[:]),
	)
}

// Helper function to format LeafProof
func formatLeafProof(leafProof [][32]byte) string {
	proofs := ""
	for i, proof := range leafProof {
		proofs += fmt.Sprintf("\n  [%d]: %s", i, hex.EncodeToString(proof[:]))
	}
	return proofs
}
