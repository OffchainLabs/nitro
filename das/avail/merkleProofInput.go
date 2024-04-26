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

//	 MarshalBinary encodes the MerkleProofInput to binary
//	 serialization format: Len(DataRootProof)+  + MerkleProofInput
//		minimum size = 210 bytes
//		------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
//
//		| 1 byte uint8 : DataRootProof length | 32*(len) byte : DataRootProof | 1 byte uint8 : LeafProof length | 32*(len) byte : LeafProof | 32 byte : RangeHash | 8 byte uint64 : DataRootIndex | 32 byte : BlobRoot | 32 byte : BridgeRoot | 32 byte : Leaf | 8 byte uint64 : LeafIndex |
//
//		<-------- len(DataRootProof) -------->|<------- DataRootProof ------->|<------- len(LeafProof) -------->|<------- LeafProof ------->|<---- RangeHash ---->|<------- DataRootIndex ------->|<---- BlobRoot ---->|<---- BridgeRoot ---->|<---- Leaf ---->|<------- LeafIndex ------->|
//
//		------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
// func (i *MerkleProofInput) MarshalToBinary() ([]byte, error) {
// 	buf := new(bytes.Buffer)
// 	err := binary.Write(buf, binary.BigEndian, uint8(len(i.DataRootProof)))
// 	if err != nil {
// 		return []byte{}, fmt.Errorf("unable to covert the merkle proof input into array of bytes and getting error:%w", err)
// 	}

// 	err = binary.Write(buf, binary.BigEndian, i.DataRootProof)
// 	if err != nil {
// 		fmt.Println("binary.Write failed:", err)
// 		return []byte{}, fmt.Errorf("unable to covert the merkle proof input into array of bytes and getting error:%w", err)
// 	}

// 	err = binary.Write(buf, binary.BigEndian, uint8(len(i.LeafProof)))
// 	if err != nil {
// 		fmt.Println("binary.Write failed:", err)
// 		return []byte{}, fmt.Errorf("unable to covert the merkle proof input into array of bytes and getting error:%w", err)
// 	}

// 	err = binary.Write(buf, binary.BigEndian, i.LeafProof)
// 	if err != nil {
// 		fmt.Println("binary.Write failed:", err)
// 		return []byte{}, fmt.Errorf("unable to covert the merkle proof input into array of bytes and getting error:%w", err)
// 	}

// 	err = binary.Write(buf, binary.BigEndian, i.RangeHash)
// 	if err != nil {
// 		fmt.Println("binary.Write failed:", err)
// 		return []byte{}, fmt.Errorf("unable to covert the merkle proof input into array of bytes and getting error:%w", err)
// 	}

// 	err = binary.Write(buf, binary.BigEndian, i.DataRootIndex)
// 	if err != nil {
// 		fmt.Println("binary.Write failed:", err)
// 		return []byte{}, fmt.Errorf("unable to covert the merkle proof input into array of bytes and getting error:%w", err)
// 	}

// 	err = binary.Write(buf, binary.BigEndian, i.BlobRoot)
// 	if err != nil {
// 		fmt.Println("binary.Write failed:", err)
// 		return []byte{}, fmt.Errorf("unable to covert the merkle proof input into array of bytes and getting error:%w", err)
// 	}

// 	err = binary.Write(buf, binary.BigEndian, i.BridgeRoot)
// 	if err != nil {
// 		fmt.Println("binary.Write failed:", err)
// 		return []byte{}, fmt.Errorf("unable to covert the merkle proof input into array of bytes and getting error:%w", err)
// 	}

// 	err = binary.Write(buf, binary.BigEndian, i.Leaf)
// 	if err != nil {
// 		fmt.Println("binary.Write failed:", err)
// 		return []byte{}, fmt.Errorf("unable to covert the merkle proof input into array of bytes and getting error:%w", err)
// 	}

// 	err = binary.Write(buf, binary.BigEndian, i.LeafIndex)
// 	if err != nil {
// 		fmt.Println("binary.Write failed:", err)
// 		return []byte{}, fmt.Errorf("unable to covert the merkle proof input into array of bytes and getting error:%w", err)
// 	}

// 	return buf.Bytes(), nil
// }

// func (m *MerkleProofInput) UnmarshalFromBinary(buf *bytes.Reader) error {
// 	var len uint8
// 	if err := binary.Read(buf, binary.BigEndian, &len); err != nil {
// 		return err
// 	}

// 	m.DataRootProof = make([]gsrpc_types.Hash, len)
// 	for i := uint8(0); i < len; i++ {
// 		if err := binary.Read(buf, binary.BigEndian, &m.DataRootProof[i]); err != nil {
// 			return err
// 		}
// 	}

// 	if err := binary.Read(buf, binary.BigEndian, &len); err != nil {
// 		return err
// 	}
// 	m.LeafProof = make([]gsrpc_types.Hash, len)
// 	for i := uint8(0); i < len; i++ {
// 		if err := binary.Read(buf, binary.BigEndian, &m.LeafProof[i]); err != nil {
// 			return err
// 		}
// 	}

// 	if err := binary.Read(buf, binary.BigEndian, &m.RangeHash); err != nil {
// 		return err
// 	}

// 	if err := binary.Read(buf, binary.BigEndian, &m.DataRootIndex); err != nil {
// 		return err
// 	}

// 	if err := binary.Read(buf, binary.BigEndian, &m.BlobRoot); err != nil {
// 		return err
// 	}

// 	if err := binary.Read(buf, binary.BigEndian, &m.BridgeRoot); err != nil {
// 		return err
// 	}

// 	if err := binary.Read(buf, binary.BigEndian, &m.Leaf); err != nil {
// 		return err
// 	}

// 	if err := binary.Read(buf, binary.BigEndian, &m.LeafIndex); err != nil {
// 		return err
// 	}

// 	return nil
// }
