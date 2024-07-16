package avail

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
