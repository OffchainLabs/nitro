// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package blobs

import (
	"bytes"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
)

const bytesEncodedPerBlob = 254 * 4096 / 8

var blsModulus, _ = new(big.Int).SetString("52435875175126190479447740508185965837690552500527637822603658699938581184513", 10)

func TestBlobEncoding(t *testing.T) {
	r := rand.New(rand.NewSource(1))
outer:
	for i := 0; i < 40; i++ {
		data := make([]byte, r.Int()%bytesEncodedPerBlob*3)
		_, err := r.Read(data)
		if err != nil {
			t.Fatalf("failed to generate random bytes: %v", err)
		}
		enc, err := EncodeBlobs(data)
		if err != nil {
			t.Errorf("failed to encode blobs for length %v: %v", len(data), err)
			continue
		}
		for _, b := range enc {
			for fieldElement := 0; fieldElement < params.BlobTxFieldElementsPerBlob; fieldElement++ {
				bigInt := new(big.Int).SetBytes(b[fieldElement*32 : (fieldElement+1)*32])
				if bigInt.Cmp(blsModulus) >= 0 {
					t.Errorf("for length %v blob %v has field element %v value %v >= modulus %v", len(data), b, fieldElement, bigInt, blsModulus)
					continue outer
				}
			}
		}
		dec, err := DecodeBlobs(enc)
		if err != nil {
			t.Errorf("failed to decode blobs for length %v: %v", len(data), err)
			continue
		}
		if !bytes.Equal(data, dec) {
			t.Errorf("got different decoding for length %v", len(data))
			continue
		}
	}
}

func TestComputeBlobProofsVersion0(t *testing.T) {
	testData := []byte("test data for blob proof version 0")
	blobs, err := EncodeBlobs(testData)
	if err != nil {
		t.Fatalf("failed to encode blobs: %v", err)
	}
	if len(blobs) == 0 {
		t.Fatal("expected at least one blob")
	}
	commitments, _, err := ComputeCommitmentsAndHashes(blobs)
	if err != nil {
		t.Fatalf("failed to compute commitments: %v", err)
	}

	proofs, version, err := ComputeBlobProofs(blobs, commitments, false)
	if err != nil {
		t.Fatalf("failed to compute version 0 proofs: %v", err)
	}

	// Check version
	if version != 0 {
		t.Errorf("expected version 0, got %d", version)
	}

	// Check proof count: should be 1 proof per blob
	expectedProofCount := len(blobs)
	if len(proofs) != expectedProofCount {
		t.Errorf("expected %d proofs, got %d", expectedProofCount, len(proofs))
	}

	// Verify the proofs are valid
	for i := range blobs {
		err = kzg4844.VerifyBlobProof(&blobs[i], commitments[i], proofs[i])
		if err != nil {
			t.Errorf("blob proof verification failed for blob %d: %v", i, err)
		}
	}
}

func TestComputeBlobProofsVersion1(t *testing.T) {
	testData := []byte("test data for blob proof version 1 with cell proofs")
	blobs, err := EncodeBlobs(testData)
	if err != nil {
		t.Fatalf("failed to encode blobs: %v", err)
	}
	if len(blobs) == 0 {
		t.Fatal("expected at least one blob")
	}
	commitments, _, err := ComputeCommitmentsAndHashes(blobs)
	if err != nil {
		t.Fatalf("failed to compute commitments: %v", err)
	}

	proofs, version, err := ComputeBlobProofs(blobs, commitments, true)
	if err != nil {
		t.Fatalf("failed to compute version 1 proofs: %v", err)
	}

	// Check version
	if version != 1 {
		t.Errorf("expected version 1, got %d", version)
	}

	// Check proof count: should be CellProofsPerBlob (128) proofs per blob
	expectedProofCount := len(blobs) * kzg4844.CellProofsPerBlob
	if len(proofs) != expectedProofCount {
		t.Errorf("expected %d proofs, got %d", expectedProofCount, len(proofs))
	}

	// Verify the cell proofs are valid
	err = kzg4844.VerifyCellProofs(blobs, commitments, proofs)
	if err != nil {
		t.Errorf("cell proof verification failed: %v", err)
	}
}

func TestComputeBlobProofsMismatchedInputs(t *testing.T) {
	testData := []byte("test data")
	blobs, err := EncodeBlobs(testData)
	if err != nil {
		t.Fatalf("failed to encode blobs: %v", err)
	}

	_, _, err = ComputeBlobProofs(blobs, []kzg4844.Commitment{}, false)
	if err == nil {
		t.Error("expected error for mismatched blobs and commitments, got nil")
	}
}

func TestComputeBlobProofsMultipleBlobsVersion0(t *testing.T) {
	// Create test data large enough to span multiple blobs
	testData := make([]byte, bytesEncodedPerBlob*2)
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	multiBlobs, err := EncodeBlobs(testData)
	if err != nil {
		t.Fatalf("failed to encode blobs: %v", err)
	}
	if len(multiBlobs) < 2 {
		t.Fatalf("expected at least 2 blobs, got %d", len(multiBlobs))
	}

	multiCommitments, _, err := ComputeCommitmentsAndHashes(multiBlobs)
	if err != nil {
		t.Fatalf("failed to compute commitments: %v", err)
	}

	proofs, version, err := ComputeBlobProofs(multiBlobs, multiCommitments, false)
	if err != nil {
		t.Fatalf("failed to compute proofs: %v", err)
	}

	if version != 0 {
		t.Errorf("expected version 0, got %d", version)
	}

	// Should be 1 proof per blob
	if len(proofs) != len(multiBlobs) {
		t.Errorf("expected %d proofs, got %d", len(multiBlobs), len(proofs))
	}

	// Verify all proofs
	for i := range multiBlobs {
		err = kzg4844.VerifyBlobProof(&multiBlobs[i], multiCommitments[i], proofs[i])
		if err != nil {
			t.Errorf("blob proof verification failed for blob %d: %v", i, err)
		}
	}
}

func TestComputeBlobProofsMultipleBlobsVersion1(t *testing.T) {
	// Create test data large enough to span multiple blobs
	testData := make([]byte, bytesEncodedPerBlob*2)
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	multiBlobs, err := EncodeBlobs(testData)
	if err != nil {
		t.Fatalf("failed to encode blobs: %v", err)
	}
	if len(multiBlobs) < 2 {
		t.Fatalf("expected at least 2 blobs, got %d", len(multiBlobs))
	}

	multiCommitments, _, err := ComputeCommitmentsAndHashes(multiBlobs)
	if err != nil {
		t.Fatalf("failed to compute commitments: %v", err)
	}

	proofs, version, err := ComputeBlobProofs(multiBlobs, multiCommitments, true)
	if err != nil {
		t.Fatalf("failed to compute proofs: %v", err)
	}

	if version != 1 {
		t.Errorf("expected version 1, got %d", version)
	}

	// Should be CellProofsPerBlob proofs per blob
	expectedCount := len(multiBlobs) * kzg4844.CellProofsPerBlob
	if len(proofs) != expectedCount {
		t.Errorf("expected %d proofs, got %d", expectedCount, len(proofs))
	}

	// Verify all cell proofs
	err = kzg4844.VerifyCellProofs(multiBlobs, multiCommitments, proofs)
	if err != nil {
		t.Errorf("cell proof verification failed: %v", err)
	}
}
