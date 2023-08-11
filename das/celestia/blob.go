package celestia

import (
	"bytes"
	"encoding/binary"
)

// BlobPointer contains the reference to the data blob on Celestia
type BlobPointer struct {
	BlockHeight  uint64
	Start        uint64
	SharesLength uint64
	Key          uint64
	NumLeaves    uint64
	ProofNonce   uint64
	TxCommitment [32]byte
	DataRoot     [32]byte
	SideNodes    [][32]byte
}

// MarshalBinary encodes the BlobPointer to binary
// serialization format: height + start + end + commitment + data root
func (b *BlobPointer) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Writing fixed-size values
	if err := binary.Write(buf, binary.BigEndian, b.BlockHeight); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, b.Start); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, b.SharesLength); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, b.Key); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, b.NumLeaves); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, b.ProofNonce); err != nil {
		return nil, err
	}

	// Writing fixed-size byte arrays directly
	if _, err := buf.Write(b.TxCommitment[:]); err != nil {
		return nil, err
	}
	if _, err := buf.Write(b.DataRoot[:]); err != nil {
		return nil, err
	}

	// Writing slice of fixed-size byte arrays
	if err := binary.Write(buf, binary.BigEndian, uint64(len(b.SideNodes))); err != nil {
		return nil, err
	}
	for _, sideNode := range b.SideNodes {
		if _, err := buf.Write(sideNode[:]); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// UnmarshalBinary decodes the binary to BlobPointer
// serialization format: height + start + end + commitment + data root
func (b *BlobPointer) UnmarshalBinary(data []byte) error {
	buf := bytes.NewReader(data)

	// Reading fixed-size values
	if err := binary.Read(buf, binary.BigEndian, &b.BlockHeight); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.BigEndian, &b.Start); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.BigEndian, &b.SharesLength); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.BigEndian, &b.Key); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.BigEndian, &b.NumLeaves); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.BigEndian, &b.ProofNonce); err != nil {
		return err
	}

	// Reading fixed-size byte arrays directly
	if err := readFixedBytes(buf, b.TxCommitment[:]); err != nil {
		return err
	}
	if err := readFixedBytes(buf, b.DataRoot[:]); err != nil {
		return err
	}

	// Reading slice of fixed-size byte arrays
	var sideNodesLen uint64
	if err := binary.Read(buf, binary.BigEndian, &sideNodesLen); err != nil {
		return err
	}
	b.SideNodes = make([][32]byte, sideNodesLen)
	for i := uint64(0); i < sideNodesLen; i++ {
		if err := readFixedBytes(buf, b.SideNodes[i][:]); err != nil {
			return err
		}
	}

	return nil
}

// readFixedBytes reads a fixed number of bytes into a byte slice
func readFixedBytes(buf *bytes.Reader, data []byte) error {
	if _, err := buf.Read(data); err != nil {
		return err
	}
	return nil
}
