package avail

type MerkleProofInput struct {

	// proof of inclusion for the data root
	DataRootProof [][32]byte `json:"dataRootProof"`
	// proof of inclusion of leaf within blob/bridge root
	LeafProof [][32]byte `json:"leafProof"`
	// abi.encodePacked(startBlock, endBlock) of header range commitment on vectorx
	RangeHash [32]byte `json:"rangeHash"`
	// index of the data root in the commitment tree
	DataRootIndex uint64 `json:"dataRootIndex"`
	// blob root to check proof against, or reconstruct the data root
	BlobRoot [32]byte `json:"blobRoot"`
	// bridge root to check proof against, or reconstruct the data root
	BridgeRoot [32]byte `json:"bridgeRoot"`
	// leaf being proven
	Leaf [32]byte `json:"leaf"`
	// index of the leaf in the blob/bridge root tree
	LeafIndex uint64 `json:"leafIndex"`
}

//	MarshalBinary encodes the MerkleProofInput to binary
//	serialization format: Len(DataRootProof)+  + MerkleProofInput
//	minimum size = 210 bytes
//	------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
//
//	| 1 byte uint8 : DataRootProof length | 32*(len) byte : DataRootProof | 1 byte uint8 : LeafProof length | 32*(len) byte : LeafProof | 32 byte : RangeHash | 8 byte uint64 : DataRootIndex | 32 byte : BlobRoot | 32 byte : BridgeRoot | 32 byte : Leaf | 8 byte uint64 : LeafIndex |
//
//	<-------- len(DataRootProof) -------->|<------- DataRootProof ------->|<------- len(LeafProof) -------->|<------- LeafProof ------->|<---- RangeHash ---->|<------- DataRootIndex ------->|<---- BlobRoot ---->|<---- BridgeRoot ---->|<---- Leaf ---->|<------- LeafIndex ------->|
//
//	------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
