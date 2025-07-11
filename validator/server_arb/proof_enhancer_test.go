//go:build disabletodofix

package server_arb

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/daprovider/referenceda"
)

func TestCustomDAProofEnhancement(t *testing.T) {
	t.Skip("TODO: Update test to work with new CustomDAProofEnhancer that requires InboxTracker and InboxReader")
	return

	// Create a reference DA validator with test data
	validator := referenceda.NewValidator()
	storage := referenceda.GetInMemoryStorage()

	// Store a test preimage
	testPreimage := []byte("test custom DA preimage data")
	hashBytes := sha256.Sum256(testPreimage)
	hash := common.BytesToHash(hashBytes[:])
	err := storage.Store(ctx, testPreimage)
	if err != nil {
		t.Fatalf("Failed to store preimage: %v", err)
	}

	// Create proof enhancer
	enhancerManager := NewProofEnhancementManager()
	// TODO: Create proper mock InboxTracker and InboxReader for testing
	customDAEnhancer := NewCustomDAProofEnhancer(validator, nil, nil)
	enhancerManager.RegisterEnhancer(MarkerCustomDARead, customDAEnhancer)

	// Create a mock proof with enhancement flag and marker
	// Format: [machine_status | 0x80, ...proof data..., hash(32), offset(8), marker(1)]
	mockProof := make([]byte, 100+32+8+1)
	mockProof[0] = 0x00 | ProofEnhancementFlag // Running status with enhancement flag
	// Fill with some dummy proof data
	for i := 1; i < 100; i++ {
		mockProof[i] = byte(i)
	}
	// Add hash
	copy(mockProof[100:132], hash[:])
	// Add offset (let's say 64)
	offset := uint64(64)
	binary.BigEndian.PutUint64(mockProof[132:140], offset)
	// Add marker
	mockProof[140] = MarkerCustomDARead

	// Enhance the proof
	// TODO: Use proper message number from test context
	enhancedProof, err := enhancerManager.EnhanceProof(ctx, 0, mockProof)
	if err != nil {
		t.Fatalf("Failed to enhance proof: %v", err)
	}

	// Verify the enhanced proof:
	// 1. Machine status should have enhancement flag removed
	if enhancedProof[0]&ProofEnhancementFlag != 0 {
		t.Error("Enhancement flag not removed from machine status")
	}

	// 2. The proof should end with hash, offset, and the actual preimage data
	// Expected format: [...original proof..., hash(32), offset(8), version(1), size(8), preimage]
	expectedEndPos := 100 + 32 + 8 + 1 + 8 + len(testPreimage)
	if len(enhancedProof) != expectedEndPos {
		t.Errorf("Enhanced proof has wrong length: got %d, expected %d", len(enhancedProof), expectedEndPos)
	}

	// Verify hash is present
	if common.BytesToHash(enhancedProof[100:132]) != hash {
		t.Error("Hash not found at expected position in enhanced proof")
	}

	// Verify offset
	gotOffset := binary.BigEndian.Uint64(enhancedProof[132:140])
	if gotOffset != offset {
		t.Errorf("Wrong offset: got %d, expected %d", gotOffset, offset)
	}

	// Verify ReferenceDA proof format
	if enhancedProof[140] != 1 { // Version
		t.Errorf("Wrong version: got %d, expected 1", enhancedProof[140])
	}

	preimageSize := binary.BigEndian.Uint64(enhancedProof[141:149])
	if preimageSize != uint64(len(testPreimage)) {
		t.Errorf("Wrong preimage size: got %d, expected %d", preimageSize, len(testPreimage))
	}

	// Verify preimage data
	gotPreimage := enhancedProof[149:]
	if string(gotPreimage) != string(testPreimage) {
		t.Errorf("Wrong preimage data: got %s, expected %s", gotPreimage, testPreimage)
	}
}

func TestNoEnhancementNeeded(t *testing.T) {
	t.Skip("TODO: Update test to work with new EnhanceProof signature that requires messageNum")
	return

	enhancerManager := NewProofEnhancementManager()

	// Create a proof without enhancement flag
	mockProof := make([]byte, 100)
	mockProof[0] = 0x00 // Running status without enhancement flag

	// Should return the same proof
	result, err := enhancerManager.EnhanceProof(ctx, 0, mockProof)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != len(mockProof) {
		t.Error("Proof was modified when no enhancement was needed")
	}
	for i := range result {
		if result[i] != mockProof[i] {
			t.Error("Proof content was modified when no enhancement was needed")
			break
		}
	}
}
