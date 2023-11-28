package avail

import (
	"encoding/json"
	"fmt"
)

// BlobPointer contains the reference to the data blob on Avail
type BlobPointer struct {
	BlockHash string // Hash for block on avail chain
	Sender    string // sender address to filter extrinsic out sepecifically for this address
	Nonce     int64  // nonce to filter specific extrinsic
}

// MarshalBinary encodes the BlobPointer to binary
func (b *BlobPointer) MarshalToBinary() ([]byte, error) {
	blobPointerData, err := json.Marshal(b)
	if err != nil {
		return []byte{}, fmt.Errorf("unable to covert the avail block referece into array of bytes and getting error:%v", err)
	}
	return blobPointerData, nil
}

// UnmarshalBinary decodes the binary to BlobPointer
func (b *BlobPointer) UnmarshalFromBinary(blobPointerData []byte) error {
	err := json.Unmarshal(blobPointerData, b)
	if err != nil {
		return fmt.Errorf("unable to convert avail_Blk_Ref bytes to AvailBlockRef Struct and getting error:%v", err)
	}
	return nil
}
