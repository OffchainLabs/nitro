package avail

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"

	gsrpc_types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
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

var byte32Type = abi.Type{T: abi.FixedBytesTy, Size: 32}
var uint32Type = abi.Type{Size: 32, T: abi.UintTy}
var stringType = abi.Type{T: abi.StringTy}
var byte32ArrayType = abi.Type{T: abi.SliceTy, Elem: &abi.Type{T: abi.FixedBytesTy, Size: 32}}
var uint64Type = abi.Type{Size: 64, T: abi.UintTy}
var merkleProofInputType = abi.Type{T: abi.TupleTy, TupleType: reflect.TypeOf(MerkleProofInput{}), TupleElems: []*abi.Type{&byte32ArrayType, &byte32ArrayType, &byte32Type, &uint64Type, &byte32Type, &byte32Type, &byte32Type, &uint64Type}, TupleRawNames: []string{"dataRootProof", "leafProof", "rangeHash", "dataRootIndex", "blobRoot", "bridgeRoot", "leaf", "leafIndex"}}

var arguments = abi.Arguments{
	{Type: byte32Type}, {Type: stringType}, {Type: uint32Type}, {Type: byte32Type}, {Type: merkleProofInputType},
}

// MarshalBinary encodes the BlobPointer to binary
// serialization format: AvailMessageHeaderFlag + BlockHash + Sender + Nonce + DasTreeRootHash + MerkleProofInput
//
//	minimum size = 330 bytes
//
// -------------------------------------------------------------------------------------------------------------------------------------------------------------
//
// | 			1 byte 	  		  |   	  32 byte         |		 48 byte       |      8 byte       |		   32 byte	         |   minimum bytes size = 210   |
//
// -------------------------------------------------------------------------------------------------------------------------------------------------------------
//
// |<-- AvailMessageHeaderFlag -->|<----- BlockHash ----->|<----- Sender ----->|<----- Nonce ----->|<----- DasTreeRootHash ----->|<----- MerkleProofInput ----->|
//
// -------------------------------------------------------------------------------------------------------------------------------------------------------------
func (b *BlobPointer) MarshalToBinary() ([]byte, error) {
	packedData, err := arguments.PackValues([]interface{}{b.BlockHash, b.Sender, b.Nonce, b.DasTreeRootHash, b.MerkleProofInput})
	if err != nil {
		return []byte{}, fmt.Errorf("unable to covert the blobPointer into array of bytes and getting error:%v", err)
	}

	// Encoding at first the avail message header flag
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, AvailMessageHeaderFlag); err != nil {
		fmt.Println("binary.Write failed:", err)
		return []byte{}, fmt.Errorf("unable to covert the avail block referece into array of bytes and getting error:%w", err)
	}
	serializedBlobPointerData := append(buf.Bytes(), packedData...)
	return serializedBlobPointerData, nil
}

func (b *BlobPointer) UnmarshalFromBinary(data []byte) error {
	unpackedData, err := arguments.UnpackValues(data)
	if err != nil {
		return fmt.Errorf("unable to covert the data bytes into blobPointer and getting error:%v", err)
	}
	b.BlockHash = unpackedData[0].([32]uint8)
	b.Sender = unpackedData[1].(string)
	b.Nonce = unpackedData[2].(uint32)
	b.DasTreeRootHash = unpackedData[3].([32]uint8)
	b.MerkleProofInput = unpackedData[4].(MerkleProofInput)
	return nil
}
