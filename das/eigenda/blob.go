package eigenda

import (
	"encoding/binary"
	"errors"
)

const batchHeaderHashLenBytes = 32

var ErrTooManyQuorumIDs = errors.New("too many quorum IDs, BlobPointer can contain at most 255 quorumIDs")
var ErrNoQuorumIDs = errors.New("no quorum IDs, BlobPointer must contain at least 1 quorumIDs")

// BlobRef contains the reference to the data blob on EigenDA
type BlobRef struct {
	BatchHeaderHash      []byte
	BlobIndex            uint32
	ReferenceBlockNumber uint32
	BlobLength           uint32
	QuorumIDs            []uint32
}

// MarshalBinary encodes the BlobPointer to binary
// serialization format: height + commitment
//
//	-------------------------------------------------------------
//
// | 32 byte commitment | 4 byte uint32  | 4 byte uint32 | 4 byte uint32 | 1 byte uint8 | 4 byte uint32 | ...
//
//	-------------------------------------------------------------
//
// | <-- batch header hash --> | <-- blob index --> | <-- reference block number --> | <-- blobLength --> | <-- num quorum IDs --> | <-- quorum ID --> | ...
//
//	-------------------------------------------------------------
func (b *BlobRef) MarshalBinary() ([]byte, error) {
	if len(b.QuorumIDs) > 255 {
		return nil, ErrTooManyQuorumIDs
	}
	if len(b.QuorumIDs) == 0 {
		return nil, ErrNoQuorumIDs
	}
	numQuorumIDs := uint8(len(b.QuorumIDs))

	blobPointerLen := batchHeaderHashLenBytes + 4 + 4 + 1 + 4*numQuorumIDs
	blob := make([]byte, blobPointerLen)

	copy(blob[:32], b.BatchHeaderHash)
	binary.LittleEndian.PutUint32(blob[32:], b.BlobIndex)
	binary.LittleEndian.PutUint32(blob[36:], b.ReferenceBlockNumber)
	binary.LittleEndian.PutUint32(blob[40:], b.BlobLength)
	blob[44] = numQuorumIDs

	for i, quorumID := range b.QuorumIDs {
		offset := 45 + i*4
		binary.LittleEndian.PutUint32(blob[offset:], quorumID)
	}

	return blob, nil
}

// UnmarshalBinary decodes the binary to BlobPointer
func (b *BlobRef) UnmarshalBinary(ref []byte) error {
	copy(b.BatchHeaderHash, ref[:32])
	b.BlobIndex = binary.LittleEndian.Uint32(ref[32:36])
	b.ReferenceBlockNumber = binary.LittleEndian.Uint32(ref[36:40])
	b.BlobLength = binary.LittleEndian.Uint32(ref[40:44])
	var numQuorumIDs uint8 = ref[44]
	quorumIDs := make([]uint32, numQuorumIDs)
	for i := 0; i < int(numQuorumIDs); i++ {
		offset := 45 + i*4
		quorumIDs[i] = binary.LittleEndian.Uint32(ref[offset : offset+4])
	}
	return nil
}
