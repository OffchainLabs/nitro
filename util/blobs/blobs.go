// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package blobs

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

func fillBlobBytes(blob []byte, data []byte) []byte {
	for fieldElement := 0; fieldElement < params.BlobTxFieldElementsPerBlob; fieldElement++ {
		startIdx := fieldElement*32 + 1
		copy(blob[startIdx:startIdx+31], data)
		if len(data) <= 31 {
			return nil
		}
		data = data[31:]
	}
	return data
}

// The number of bits in a BLS scalar that aren't part of a whole byte.
const spareBlobBits = 6 // = math.floor(math.log2(BLS_MODULUS)) % 8

// The number of bytes encodable in a blob with the current encoding scheme.
const BlobEncodableData = 254 * params.BlobTxFieldElementsPerBlob / 8

func fillBlobBits(blob []byte, data []byte) ([]byte, error) {
	var acc uint16
	accBits := 0
	for fieldElement := 0; fieldElement < params.BlobTxFieldElementsPerBlob; fieldElement++ {
		if accBits < spareBlobBits && len(data) > 0 {
			acc |= uint16(data[0]) << accBits
			accBits += 8
			data = data[1:]
		}
		blob[fieldElement*32] = uint8(acc & ((1 << spareBlobBits) - 1))
		accBits -= spareBlobBits
		if accBits < 0 {
			// We're out of data
			break
		}
		acc >>= spareBlobBits
	}
	if accBits > 0 {
		return nil, fmt.Errorf("somehow ended up with %v spare accBits", accBits)
	}
	return data, nil
}

// EncodeBlobs takes in raw bytes data to convert into blobs used for KZG commitment EIP-4844
// transactions on Ethereum.
func EncodeBlobs(data []byte) ([]kzg4844.Blob, error) {
	data, err := rlp.EncodeToBytes(data)
	if err != nil {
		return nil, err
	}
	var blobs []kzg4844.Blob
	for len(data) > 0 {
		var b kzg4844.Blob
		data = fillBlobBytes(b[:], data)
		data, err = fillBlobBits(b[:], data)
		if err != nil {
			return nil, err
		}
		blobs = append(blobs, b)
	}
	return blobs, nil
}

// DecodeBlobs decodes blobs into the batch data encoded in them.
func DecodeBlobs(blobs []kzg4844.Blob) ([]byte, error) {
	var rlpData []byte
	for _, blob := range blobs {
		for fieldIndex := 0; fieldIndex < params.BlobTxFieldElementsPerBlob; fieldIndex++ {
			rlpData = append(rlpData, blob[fieldIndex*32+1:(fieldIndex+1)*32]...)
		}
		var acc uint16
		accBits := 0
		for fieldIndex := 0; fieldIndex < params.BlobTxFieldElementsPerBlob; fieldIndex++ {
			acc |= uint16(blob[fieldIndex*32]) << accBits
			accBits += spareBlobBits
			if accBits >= 8 {
				rlpData = append(rlpData, uint8(acc))
				acc >>= 8
				accBits -= 8
			}
		}
		if accBits != 0 {
			return nil, fmt.Errorf("somehow ended up with %v spare accBits", accBits)
		}
	}
	var outputData []byte
	err := rlp.Decode(bytes.NewReader(rlpData), &outputData)
	return outputData, err
}

func CommitmentToVersionedHash(commitment kzg4844.Commitment) common.Hash {
	// As per the EIP-4844 spec, the versioned hash is the SHA-256 hash of the commitment with the first byte set to 1.
	hash := sha256.Sum256(commitment[:])
	hash[0] = 1
	return hash
}

// Return KZG commitments, proofs, and versioned hashes that corresponds to these blobs
func ComputeCommitmentsAndHashes(blobs []kzg4844.Blob) ([]kzg4844.Commitment, []common.Hash, error) {
	commitments := make([]kzg4844.Commitment, len(blobs))
	versionedHashes := make([]common.Hash, len(blobs))

	for i := range blobs {
		var err error
		commitments[i], err = kzg4844.BlobToCommitment(blobs[i])
		if err != nil {
			return nil, nil, err
		}
		versionedHashes[i] = CommitmentToVersionedHash(commitments[i])
	}

	return commitments, versionedHashes, nil
}

func ComputeBlobProofs(blobs []kzg4844.Blob, commitments []kzg4844.Commitment) ([]kzg4844.Proof, error) {
	if len(blobs) != len(commitments) {
		return nil, fmt.Errorf("ComputeBlobProofs got %v blobs but %v commitments", len(blobs), len(commitments))
	}
	proofs := make([]kzg4844.Proof, len(blobs))
	for i := range blobs {
		var err error
		proofs[i], err = kzg4844.ComputeBlobProof(blobs[i], commitments[i])
		if err != nil {
			return nil, err
		}
	}

	return proofs, nil
}
