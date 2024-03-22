package avail

import (
	"bytes"
	"encoding/binary"
	"fmt"

	gsrpc_types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/common"
)

// BlobPointer contains the reference to the data blob on Avail
type BlobPointer struct {
	BlockHash        gsrpc_types.Hash // Hash for block on avail chain
	Sender           string           // sender address to filter extrinsic out sepecifically for this address
	Nonce            uint32           // nonce to filter specific extrinsic
	DasTreeRootHash  common.Hash      // Das tree root hash created when preimage is stored on das tree
	MerkleProofInput MerkleProofInput // Merkle proof of the blob submission
}

// MarshalBinary encodes the BlobPointer to binary
// serialization format: AvailMessageHeaderFlag + MerkleProofInput + BlockHash + Sender + Nonce + DasTreeRootHash
//
//	minimum size = 330 bytes
//	-------------------------------------------------------------------------------------------------------------------------------------------------------------
//
// | 			1 byte 	  		  |   minimum bytes size = 210   |   	  32 byte        |		 48 byte      |      8 byte       |			  32 byte	        |
//
//	-------------------------------------------------------------------------------------------------------------------------------------------------------------
//
// |<-- AvailMessageHeaderFlag -->|<----- MerkleProofInput ----->|<----- BlockHash ----->|<----- Sender ----->|<----- Nonce ----->|<----- DasTreeRootHash ----->|
//
//	-------------------------------------------------------------------------------------------------------------------------------------------------------------
func (b *BlobPointer) MarshalToBinary() ([]byte, error) {

	buf := new(bytes.Buffer)

	// Encoding at first the avail message header flag
	if err := binary.Write(buf, binary.BigEndian, AvailMessageHeaderFlag); err != nil {
		fmt.Println("binary.Write failed:", err)
		return []byte{}, fmt.Errorf("unable to covert the avail block referece into array of bytes and getting error:%w", err)
	}

	// Marshaling in between: The Merkle proof input, which will be required for DA verification
	merkleProofInput, err := b.MerkleProofInput.MarshalToBinary()
	if err != nil {
		return []byte{}, fmt.Errorf("unable to covert the avail block referece into array of bytes and getting error:%w", err)
	}
	buf.Write(merkleProofInput)

	// Encoding at last: blockHash, sender address, nonce and DASTreeRootHash which will not be required for DA verification
	if err := binary.Write(buf, binary.BigEndian, b.BlockHash); err != nil {
		fmt.Println("binary.Write failed:", err)
		return []byte{}, fmt.Errorf("unable to covert the avail block referece into array of bytes and getting error:%w", err)
	}
	var senderBytes = []byte(b.Sender)
	if err = binary.Write(buf, binary.BigEndian, uint8(len(senderBytes))); err != nil {
		fmt.Println("binary.Write failed:", err)
		return []byte{}, fmt.Errorf("unable to covert the avail block referece into array of bytes and getting error:%w", err)
	}
	if err = binary.Write(buf, binary.BigEndian, senderBytes); err != nil {
		fmt.Println("binary.Write failed:", err)
		return []byte{}, fmt.Errorf("unable to covert the avail block referece into array of bytes and getting error:%w", err)
	}
	if err = binary.Write(buf, binary.BigEndian, b.Nonce); err != nil {
		fmt.Println("binary.Write failed:", err)
		return []byte{}, fmt.Errorf("unable to covert the avail block referece into array of bytes and getting error:%w", err)
	}
	if err = binary.Write(buf, binary.BigEndian, b.DasTreeRootHash); err != nil {
		fmt.Println("binary.Write failed:", err)
	}

	return buf.Bytes(), nil
}

// UnmarshalBinary decodes the binary to BlobPointer
func (b *BlobPointer) UnmarshalFromBinary(blobPointerData []byte) error {
	buf := bytes.NewReader(blobPointerData)

	if err := b.MerkleProofInput.UnmarshalFromBinary(buf); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.BigEndian, &b.BlockHash); err != nil {
		return err
	}

	var len uint8
	if err := binary.Read(buf, binary.BigEndian, &len); err != nil {
		return err
	}
	var senderBytes = make([]byte, len)
	if err := binary.Read(buf, binary.BigEndian, &senderBytes); err != nil {
		return err
	}
	b.Sender = string(senderBytes)
	if err := binary.Read(buf, binary.BigEndian, &b.Nonce); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.BigEndian, &b.DasTreeRootHash); err != nil {
		return err
	}

	return nil
}
