package proofenhancement

import (
	"context"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/containers"
)

// Mock implementations for testing - only implementing the methods we actually use
type mockInboxTracker struct {
	batchForMessage uint64
	found           bool
	err             error
}

// Implement staker.InboxTrackerInterface - only the methods we use
func (m *mockInboxTracker) SetBlockValidator(v *staker.BlockValidator) {}
func (m *mockInboxTracker) GetDelayedMessageBytes(ctx context.Context, seqNum uint64) ([]byte, error) {
	return nil, nil
}
func (m *mockInboxTracker) GetBatchMessageCount(seqNum uint64) (arbutil.MessageIndex, error) {
	return 0, nil
}
func (m *mockInboxTracker) GetBatchAcc(seqNum uint64) (common.Hash, error) {
	return common.Hash{}, nil
}
func (m *mockInboxTracker) GetBatchCount() (uint64, error) {
	return 0, nil
}
func (m *mockInboxTracker) FindInboxBatchContainingMessage(msgNum arbutil.MessageIndex) (uint64, bool, error) {
	return m.batchForMessage, m.found, m.err
}

type mockInboxReader struct {
	sequencerMessage []byte
	err              error
}

// Implement staker.InboxReaderInterface - only the methods we use
func (m *mockInboxReader) GetSequencerMessageBytes(ctx context.Context, batchNum uint64) ([]byte, common.Hash, error) {
	return m.sequencerMessage, common.Hash{}, m.err
}
func (m *mockInboxReader) GetFinalizedMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	return 0, nil
}

type mockValidator struct {
	generateReadPreimageProofResult []byte
	generateCertValidityProofResult []byte
	err                             error
}

func (m *mockValidator) GenerateReadPreimageProof(certHash common.Hash, offset uint64, certificate []byte) containers.PromiseInterface[daprovider.PreimageProofResult] {
	if m.err != nil {
		return containers.NewReadyPromise(daprovider.PreimageProofResult{}, m.err)
	}
	return containers.NewReadyPromise(daprovider.PreimageProofResult{
		Proof: m.generateReadPreimageProofResult,
	}, nil)
}

func (m *mockValidator) GenerateCertificateValidityProof(certificate []byte) containers.PromiseInterface[daprovider.ValidityProofResult] {
	if m.err != nil {
		return containers.NewReadyPromise(daprovider.ValidityProofResult{}, m.err)
	}
	return containers.NewReadyPromise(daprovider.ValidityProofResult{
		Proof: m.generateCertValidityProofResult,
	}, nil)
}

func createTestCertificate(t *testing.T, data []byte) []byte {
	// Create a simple test certificate
	// Format: [header(1), dataHash(32), v(1), r(32), s(32)]
	cert := make([]byte, 1+32+1+32+32)
	cert[0] = daprovider.DACertificateMessageHeaderFlag

	// Use Keccak256 for data hash
	dataHash := crypto.Keccak256(data)
	copy(cert[1:33], dataHash)

	// Mock signature values (v, r, s)
	cert[33] = 27 // v
	// r and s are left as zeros for simplicity

	return cert
}

func TestCustomDAProofEnhancement(t *testing.T) {
	ctx := context.Background()

	// Test data
	testData := []byte("test custom DA preimage data")
	testCertificate := createTestCertificate(t, testData)
	certHash := crypto.Keccak256Hash(testCertificate)
	testOffset := uint64(10)

	// Create sequencer message with 40-byte header + certificate
	sequencerMessage := make([]byte, 40+len(testCertificate))
	copy(sequencerMessage[40:], testCertificate)

	// Mock components
	inboxTracker := &mockInboxTracker{
		batchForMessage: 123,
		found:           true,
	}

	inboxReader := &mockInboxReader{
		sequencerMessage: sequencerMessage,
	}

	// Mock validator that returns a simple proof
	mockProof := []byte{0x01, 0x02, 0x03, 0x04} // Simple test proof
	mockValidator := &mockValidator{
		generateReadPreimageProofResult: mockProof,
	}

	// Create proof enhancer
	enhancerManager := NewProofEnhancementManager()
	customDAEnhancer := NewReadPreimageProofEnhancer(mockValidator, inboxTracker, inboxReader)
	enhancerManager.RegisterEnhancer(MarkerCustomDAReadPreimage, customDAEnhancer)

	// Create a mock proof with enhancement flag and marker
	// Format: [machine_status | 0x80, ...proof data..., certHash(32), offset(8), marker(1)]
	originalProofSize := 100
	originalProof := make([]byte, originalProofSize+32+8+1)
	originalProof[0] = 0x00 | ProofEnhancementFlag // Running status with enhancement flag
	// Fill with some dummy proof data
	for i := 1; i < originalProofSize; i++ {
		originalProof[i] = byte(i)
	}
	// Add certificate hash
	copy(originalProof[originalProofSize:originalProofSize+32], certHash[:])
	// Add offset
	binary.BigEndian.PutUint64(originalProof[originalProofSize+32:originalProofSize+40], testOffset)
	// Add marker
	originalProof[originalProofSize+40] = MarkerCustomDAReadPreimage

	// Enhance the proof
	testMessageNum := arbutil.MessageIndex(42)
	enhancedProof, err := enhancerManager.EnhanceProof(ctx, testMessageNum, originalProof)
	if err != nil {
		t.Fatalf("Failed to enhance proof: %v", err)
	}

	// Verify the enhanced proof:
	// 1. Machine status should have enhancement flag removed
	if enhancedProof[0]&ProofEnhancementFlag != 0 {
		t.Error("Enhancement flag not removed from machine status")
	}

	// 2. The marker data (certHash, offset, marker) should be removed
	// Expected format: [...original proof..., certSize(8), certificate, customProof]
	expectedSize := originalProofSize + 8 + len(testCertificate) + len(mockProof)
	if len(enhancedProof) != expectedSize {
		t.Errorf("Enhanced proof has wrong length: got %d, expected %d", len(enhancedProof), expectedSize)
	}

	// 3. Verify original proof is preserved (minus enhancement flag)
	for i := 1; i < originalProofSize; i++ {
		if enhancedProof[i] != byte(i) {
			t.Errorf("Original proof data modified at position %d: got %d, expected %d", i, enhancedProof[i], i)
			break
		}
	}

	// 4. Verify certificate size
	offset := originalProofSize
	certSize := binary.BigEndian.Uint64(enhancedProof[offset : offset+8])
	if certSize != uint64(len(testCertificate)) {
		t.Errorf("Wrong certificate size: got %d, expected %d", certSize, len(testCertificate))
	}
	offset += 8

	// 5. Verify certificate
	// #nosec G115
	gotCertificate := enhancedProof[offset : offset+int(certSize)]
	if !equal(gotCertificate, testCertificate) {
		t.Errorf("Wrong certificate in enhanced proof")
	}
	// #nosec G115
	offset += int(certSize)

	// 6. Verify custom proof from validator
	gotCustomProof := enhancedProof[offset:]
	if !equal(gotCustomProof, mockProof) {
		t.Errorf("Wrong custom proof: got %v, expected %v", gotCustomProof, mockProof)
	}
}

func TestNoEnhancementNeeded(t *testing.T) {
	ctx := context.Background()
	enhancerManager := NewProofEnhancementManager()

	// Create a proof without enhancement flag
	mockProof := make([]byte, 100)
	mockProof[0] = 0x00 // Running status without enhancement flag
	for i := 1; i < 100; i++ {
		mockProof[i] = byte(i)
	}

	// Should return the same proof
	result, err := enhancerManager.EnhanceProof(ctx, arbutil.MessageIndex(1), mockProof)
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

func TestValidateCertificateProofEnhancement(t *testing.T) {
	ctx := context.Background()

	// Test data
	testData := []byte("test data for certificate validation")
	testCertificate := createTestCertificate(t, testData)
	certHash := crypto.Keccak256Hash(testCertificate)

	// Create sequencer message with 40-byte header + certificate
	sequencerMessage := make([]byte, 40+len(testCertificate))
	copy(sequencerMessage[40:], testCertificate)

	// Mock components
	inboxTracker := &mockInboxTracker{
		batchForMessage: 456,
		found:           true,
	}

	inboxReader := &mockInboxReader{
		sequencerMessage: sequencerMessage,
	}

	// Mock validator that returns a validity proof
	mockValidityProof := []byte{0x01, 0x01} // Valid certificate, version 1
	mockValidator := &mockValidator{
		generateCertValidityProofResult: mockValidityProof,
	}

	// Create proof enhancer
	enhancerManager := NewProofEnhancementManager()
	certEnhancer := NewValidateCertificateProofEnhancer(mockValidator, inboxTracker, inboxReader)
	enhancerManager.RegisterEnhancer(MarkerCustomDAValidateCertificate, certEnhancer)

	// Create a mock proof with enhancement flag and marker
	// Format: [machine_status | 0x80, ...proof data..., certHash(32), marker(1)]
	originalProofSize := 100
	mockProof := make([]byte, originalProofSize+32+1)
	mockProof[0] = 0x00 | ProofEnhancementFlag // Running status with enhancement flag
	for i := 1; i < originalProofSize; i++ {
		mockProof[i] = byte(i)
	}
	copy(mockProof[originalProofSize:originalProofSize+32], certHash[:])
	mockProof[originalProofSize+32] = MarkerCustomDAValidateCertificate

	// Enhance the proof
	testMessageNum := arbutil.MessageIndex(789)
	enhancedProof, err := enhancerManager.EnhanceProof(ctx, testMessageNum, mockProof)
	if err != nil {
		t.Fatalf("Failed to enhance proof: %v", err)
	}

	// Verify the enhanced proof
	if enhancedProof[0]&ProofEnhancementFlag != 0 {
		t.Error("Enhancement flag not removed from machine status")
	}

	// Expected format: [...original proof..., certSize(8), certificate, validityProof]
	expectedSize := originalProofSize + 8 + len(testCertificate) + len(mockValidityProof)
	if len(enhancedProof) != expectedSize {
		t.Errorf("Enhanced proof has wrong length: got %d, expected %d", len(enhancedProof), expectedSize)
	}

	// Verify certificate size and data
	offset := originalProofSize
	certSize := binary.BigEndian.Uint64(enhancedProof[offset : offset+8])
	if certSize != uint64(len(testCertificate)) {
		t.Errorf("Wrong certificate size: got %d, expected %d", certSize, len(testCertificate))
	}
	offset += 8

	// #nosec G115
	gotCertificate := enhancedProof[offset : offset+int(certSize)]
	if !equal(gotCertificate, testCertificate) {
		t.Errorf("Wrong certificate in enhanced proof")
	}
	// #nosec G115
	offset += int(certSize)

	// Verify validity proof
	gotValidityProof := enhancedProof[offset:]
	if !equal(gotValidityProof, mockValidityProof) {
		t.Errorf("Wrong validity proof: got %v, expected %v", gotValidityProof, mockValidityProof)
	}
}

// Helper function to compare byte slices
func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestProofEnhancerErrorCases(t *testing.T) {
	ctx := context.Background()

	t.Run("UnknownMarker", func(t *testing.T) {
		enhancerManager := NewProofEnhancementManager()
		// Don't register any enhancers

		mockProof := make([]byte, 10)
		mockProof[0] = ProofEnhancementFlag // Set enhancement flag
		mockProof[9] = 0xFF                 // Unknown marker

		_, err := enhancerManager.EnhanceProof(ctx, 0, mockProof)
		if err == nil {
			t.Error("Expected error for unknown marker")
		}
		if err.Error() != "unknown enhancement marker: 0xff" {
			t.Errorf("Wrong error message: %v", err)
		}
	})

	t.Run("ProofTooShort", func(t *testing.T) {
		enhancerManager := NewProofEnhancementManager()

		// Empty proof with enhancement flag
		mockProof := []byte{}

		result, err := enhancerManager.EnhanceProof(ctx, 0, mockProof)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Error("Empty proof should be returned unchanged")
		}
	})

	t.Run("CertificateHashMismatch", func(t *testing.T) {
		testCertificate := createTestCertificate(t, []byte("test data"))
		wrongHash := crypto.Keccak256Hash([]byte("wrong data"))

		sequencerMessage := make([]byte, 40+len(testCertificate))
		copy(sequencerMessage[40:], testCertificate)

		inboxTracker := &mockInboxTracker{
			batchForMessage: 1,
			found:           true,
		}

		inboxReader := &mockInboxReader{
			sequencerMessage: sequencerMessage,
		}

		validator := &mockValidator{}

		enhancerManager := NewProofEnhancementManager()
		enhancer := NewReadPreimageProofEnhancer(validator, inboxTracker, inboxReader)
		enhancerManager.RegisterEnhancer(MarkerCustomDAReadPreimage, enhancer)

		// Create proof with wrong hash
		mockProof := make([]byte, 100+32+8+1)
		mockProof[0] = ProofEnhancementFlag
		copy(mockProof[100:132], wrongHash[:])
		binary.BigEndian.PutUint64(mockProof[132:140], 0)
		mockProof[140] = MarkerCustomDAReadPreimage

		_, err := enhancerManager.EnhanceProof(ctx, 0, mockProof)
		if err == nil {
			t.Error("Expected error for certificate hash mismatch")
		}
		if !strings.Contains(err.Error(), "certificate hash mismatch") {
			t.Errorf("Wrong error message: %v", err)
		}
	})

	t.Run("BatchNotFound", func(t *testing.T) {
		inboxTracker := &mockInboxTracker{
			found: false,
		}

		inboxReader := &mockInboxReader{}
		validator := &mockValidator{}

		enhancerManager := NewProofEnhancementManager()
		enhancer := NewReadPreimageProofEnhancer(validator, inboxTracker, inboxReader)
		enhancerManager.RegisterEnhancer(MarkerCustomDAReadPreimage, enhancer)

		certHash := crypto.Keccak256Hash([]byte("test"))
		mockProof := make([]byte, 100+32+8+1)
		mockProof[0] = ProofEnhancementFlag
		copy(mockProof[100:132], certHash[:])
		mockProof[140] = MarkerCustomDAReadPreimage

		_, err := enhancerManager.EnhanceProof(ctx, 42, mockProof)
		if err == nil {
			t.Error("Expected error when batch not found")
		}
		if !strings.Contains(err.Error(), "Couldn't find batch") {
			t.Errorf("Wrong error message: %v", err)
		}
	})
}
