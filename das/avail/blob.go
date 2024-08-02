package avail

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// BlobPointer contains the reference to the data blob on Avail
type BlobPointer struct {
	BlockHeight        uint32      // Block height for avail chain in which data in being included
	ExtrinsicIndex     uint32      // extrinsic index in the block height
	DasTreeRootHash    common.Hash // Das tree root hash created when preimage is stored on das tree
	BlobDataKeccak265H common.Hash // Keccak256(blobData) to verify the originality of proof (it will work as preimage of the commitment)
	BlobProof          BlobProof   // Blob proof of blob inclusion into avail finalised block
}

var byte32Type = abi.Type{T: abi.FixedBytesTy, Size: 32}
var uint32Type = abi.Type{Size: 32, T: abi.UintTy}
var stringType = abi.Type{T: abi.StringTy}
var byte32ArrayType = abi.Type{T: abi.SliceTy, Elem: &abi.Type{T: abi.FixedBytesTy, Size: 32}}
var uint64Type = abi.Type{Size: 64, T: abi.UintTy}
var merkleProofInputType = abi.Type{T: abi.TupleTy, TupleType: reflect.TypeOf(MerkleProofInput{}), TupleElems: []*abi.Type{&byte32ArrayType, &byte32ArrayType, &byte32Type, &uint64Type, &byte32Type, &byte32Type, &byte32Type, &uint64Type}, TupleRawNames: []string{"dataRootProof", "leafProof", "rangeHash", "dataRootIndex", "blobRoot", "bridgeRoot", "leaf", "leafIndex"}}
var blobProofType = abi.Type{T: abi.TupleTy, TupleType: reflect.TypeOf(BlobProof{}), TupleElems: []*abi.Type{&byte32Type, &byte32Type, &byte32Type, &byte32ArrayType, &uint32Type, &uint32Type, &byte32Type}, TupleRawNames: []string{"dataRoot", "blobRoot", "bridgeRoot", "leafProof", "numberOfLeaves", "leafIndex", "leaf"}}
var arguments = abi.Arguments{
	{Type: uint32Type}, {Type: uint32Type}, {Type: byte32Type}, {Type: byte32Type}, {Type: blobProofType},
}

// MarshalBinary encodes the BlobPointer to binary
// serialization format: AvailMessageHeaderFlag + BlockHeight + ExtrinsicIndex + DasTreeRootHash + BlobProof
//
//	minimum size approx = 300 bytes
//
// ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
//
// | 			1 byte 	  		  |   	  32 byte         |		 32 byte       			 | 		      32 byte	       |			32 byte			   |   minimum bytes size = 176   |
//
// -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
//
// |<-- AvailMessageHeaderFlag -->|<----- BlockHeight ----->|<----- ExtrinsicIndex ----->|<----- DasTreeRootHash ----->|<-----BlobDataKeccak265H------>|<---------BlobProof --------->|
//
// -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
func (b *BlobPointer) MarshalToBinary() ([]byte, error) {
	packedData, err := arguments.PackValues([]interface{}{b.BlockHeight, b.ExtrinsicIndex, b.DasTreeRootHash, b.BlobDataKeccak265H, b.BlobProof})
	if err != nil {
		return []byte{}, fmt.Errorf("unable to covert the blobPointer into array of bytes and getting error:%w", err)
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
		return fmt.Errorf("unable to covert the data bytes into blobPointer and getting error:%w", err)
	}
	b.BlockHeight = unpackedData[0].(uint32)           //nolint:all
	b.ExtrinsicIndex = unpackedData[1].(uint32)        //nolint:all
	b.DasTreeRootHash = unpackedData[2].([32]uint8)    //nolint:all
	b.BlobDataKeccak265H = unpackedData[3].([32]uint8) //nolint:all
	b.BlobProof = unpackedData[4].(BlobProof)          //nolint:all
	return nil
}

// Method to convert BlobPointer to string
func (bp *BlobPointer) String() string {
	return fmt.Sprintf(
		"BlockHeight: %d,  ExtrinsicIndex: %d,  DasTreeRootHash: %s,  BlobDataKeccak265H: %s,  BlobProof: %s",
		bp.BlockHeight,
		bp.ExtrinsicIndex,
		bp.DasTreeRootHash.Hex(),
		bp.BlobDataKeccak265H.Hex(),
		bp.BlobProof.String(),
	)
}
