// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package daprovider

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/util/containers"
)

// mockReader is a simple mock implementation of the Reader interface for testing
type mockReader struct {
	name string
}

func (m *mockReader) RecoverPayload(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[PayloadResult] {
	panic("not implemented in mock")
}

func (m *mockReader) CollectPreimages(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[PreimagesResult] {
	panic("not implemented in mock")
}

// mockValidator is a simple mock implementation of the Validator interface for testing
type mockValidator struct {
	name string
}

func (m *mockValidator) GenerateReadPreimageProof(
	certHash common.Hash,
	offset uint64,
	certificate []byte,
) containers.PromiseInterface[PreimageProofResult] {
	panic("not implemented in mock")
}

func (m *mockValidator) GenerateCertificateValidityProof(
	certificate []byte,
) containers.PromiseInterface[ValidityProofResult] {
	panic("not implemented in mock")
}

func TestRegister_ShadowingPrevention(t *testing.T) {
	tests := []struct {
		name          string
		first         []byte
		second        []byte
		expectError   bool
		errorContains string
	}{
		{
			name:          "new would shadow existing (shorter prefix)",
			first:         []byte{0x01, 0xFF},
			second:        []byte{0x01},
			expectError:   true,
			errorContains: "would shadow existing registration",
		},
		{
			name:          "new would be shadowed by existing (longer prefix)",
			first:         []byte{0x01},
			second:        []byte{0x01, 0xFF},
			expectError:   true,
			errorContains: "would be shadowed by existing registration",
		},
		{
			name:          "new would be shadowed by longer existing",
			first:         []byte{0x01, 0xFF, 0x00},
			second:        []byte{0x01, 0xFF},
			expectError:   true,
			errorContains: "would shadow existing registration",
		},
		{
			name:          "new would shadow shorter existing",
			first:         []byte{0x01, 0xFF},
			second:        []byte{0x01, 0xFF, 0x00},
			expectError:   true,
			errorContains: "would be shadowed by existing registration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewDAProviderRegistry()
			reader1 := &mockReader{name: "reader1"}
			reader2 := &mockReader{name: "reader2"}

			// Register first header bytes
			err := registry.Register(tt.first, reader1, nil)
			if err != nil {
				t.Fatalf("unexpected error registering first: %v", err)
			}

			// Attempt to register second header bytes
			err = registry.Register(tt.second, reader2, nil)
			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing %q, got: %v", tt.errorContains, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRegister_NonOverlappingSucceeds(t *testing.T) {
	tests := []struct {
		name        string
		headerBytes [][]byte
	}{
		{
			name: "different second bytes",
			headerBytes: [][]byte{
				{0x01, 0xFF},
				{0x01, 0xFE},
			},
		},
		{
			name: "different first bytes - single byte each",
			headerBytes: [][]byte{
				{0x80},
				{0x88},
			},
		},
		{
			name: "different first bytes - multi-byte",
			headerBytes: [][]byte{
				{0x01, 0xFF},
				{0x02, 0xFF},
			},
		},
		{
			name: "mixed lengths, non-overlapping",
			headerBytes: [][]byte{
				{0x50},
				{0x80},
				{0x01, 0xFF},
				{0x01, 0xFE},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewDAProviderRegistry()
			reader := &mockReader{name: "reader"}

			for i, hb := range tt.headerBytes {
				err := registry.Register(hb, reader, nil)
				if err != nil {
					t.Fatalf("unexpected error registering header bytes %d (%x): %v", i, hb, err)
				}
			}

			// Verify all are registered
			supported := registry.SupportedHeaderBytes()
			if len(supported) != len(tt.headerBytes) {
				t.Errorf("expected %d registered, got %d", len(tt.headerBytes), len(supported))
			}
		})
	}
}

func TestRegister_DuplicateRegistration(t *testing.T) {
	tests := []struct {
		name   string
		reader Reader
	}{
		{
			name:   "same reader",
			reader: &mockReader{name: "reader1"},
		},
		{
			name:   "different reader",
			reader: &mockReader{name: "reader2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewDAProviderRegistry()
			reader1 := &mockReader{name: "reader1"}
			headerBytes := []byte{0x01, 0xFF}

			// Register once
			err := registry.Register(headerBytes, reader1, nil)
			if err != nil {
				t.Fatalf("unexpected error on first registration: %v", err)
			}

			// Attempt to register again - should fail regardless of reader
			err = registry.Register(headerBytes, tt.reader, nil)
			if err == nil {
				t.Fatal("expected error when registering duplicate header bytes")
			}
			if !strings.Contains(err.Error(), "already registered") {
				t.Errorf("expected error containing 'already registered', got: %v", err)
			}
		})
	}
}

func TestRegister_InvalidInputs(t *testing.T) {
	tests := []struct {
		name          string
		reader        Reader
		validator     Validator
		headerBytes   []byte
		errorContains string
	}{
		{
			name:          "nil reader and validator",
			reader:        nil,
			validator:     nil,
			headerBytes:   []byte{0x01},
			errorContains: "cannot register with both reader and validator nil",
		},
		{
			name:          "empty header bytes",
			reader:        &mockReader{name: "reader"},
			validator:     nil,
			headerBytes:   []byte{},
			errorContains: "cannot register empty header bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewDAProviderRegistry()
			err := registry.Register(tt.headerBytes, tt.reader, tt.validator)
			if err == nil {
				t.Fatal("expected error but got none")
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q, got: %v", tt.errorContains, err)
			}
		})
	}
}

func TestGetReader_PrefixMatching(t *testing.T) {
	registry := NewDAProviderRegistry()
	reader := &mockReader{name: "reader"}
	headerBytes := []byte{0x01, 0xFF}

	err := registry.Register(headerBytes, reader, nil)
	if err != nil {
		t.Fatalf("unexpected error registering: %v", err)
	}

	tests := []struct {
		name        string
		message     []byte
		shouldFind  bool
		description string
	}{
		{
			name:        "exact match",
			message:     []byte{0x01, 0xFF},
			shouldFind:  true,
			description: "exact match should find reader",
		},
		{
			name:        "message with suffix",
			message:     []byte{0x01, 0xFF, 0x11, 0x22, 0x33},
			shouldFind:  true,
			description: "prefix match should find reader",
		},
		{
			name:        "different second byte",
			message:     []byte{0x01, 0xFE, 0x11, 0x22},
			shouldFind:  false,
			description: "different prefix should not match",
		},
		{
			name:        "too short",
			message:     []byte{0x01},
			shouldFind:  false,
			description: "message shorter than registered bytes should not match",
		},
		{
			name:        "completely different",
			message:     []byte{0x80, 0x11, 0x22},
			shouldFind:  false,
			description: "completely different prefix should not match",
		},
		{
			name:        "empty message",
			message:     []byte{},
			shouldFind:  false,
			description: "empty message should not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foundReader := registry.GetReader(tt.message)
			found := foundReader != nil
			if found != tt.shouldFind {
				t.Errorf("%s: expected found=%v, got %v", tt.description, tt.shouldFind, found)
			}
			if tt.shouldFind && foundReader != reader {
				t.Errorf("found wrong reader")
			}
		})
	}
}

func TestGetReader_FirstMatch(t *testing.T) {
	registry := NewDAProviderRegistry()
	reader1 := &mockReader{name: "reader1"}
	reader2 := &mockReader{name: "reader2"}

	// Register two non-overlapping patterns
	err := registry.Register([]byte{0x80}, reader1, nil)
	if err != nil {
		t.Fatalf("unexpected error registering first: %v", err)
	}
	err = registry.Register([]byte{0x88}, reader2, nil)
	if err != nil {
		t.Fatalf("unexpected error registering second: %v", err)
	}

	tests := []struct {
		name           string
		message        []byte
		expectedReader *mockReader
	}{
		{
			name:           "matches first pattern",
			message:        []byte{0x80, 0x11, 0x22},
			expectedReader: reader1,
		},
		{
			name:           "matches second pattern",
			message:        []byte{0x88, 0x11, 0x22},
			expectedReader: reader2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foundReader := registry.GetReader(tt.message)
			if foundReader == nil {
				t.Fatal("expected to find reader")
			}
			if foundReader != tt.expectedReader {
				t.Errorf("found wrong reader: expected %v, got %v", tt.expectedReader, foundReader)
			}
		})
	}
}

func TestRegisterAll_Success(t *testing.T) {
	registry := NewDAProviderRegistry()
	reader := &mockReader{name: "reader"}
	headerBytesList := [][]byte{
		{0x80},
		{0x88},
	}

	err := registry.RegisterAll(headerBytesList, reader, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify both are registered and can be looked up
	for _, hb := range headerBytesList {
		foundReader := registry.GetReader(hb)
		if foundReader == nil {
			t.Errorf("header bytes %x not found", hb)
		}
		if foundReader != reader {
			t.Errorf("wrong reader found for %x", hb)
		}
	}

	supported := registry.SupportedHeaderBytes()
	if len(supported) != 2 {
		t.Errorf("expected 2 registrations, got %d", len(supported))
	}
}

func TestRegisterAll_FailsOnShadowing(t *testing.T) {
	registry := NewDAProviderRegistry()
	reader1 := &mockReader{name: "reader1"}
	reader2 := &mockReader{name: "reader2"}

	// Pre-register a pattern
	err := registry.Register([]byte{0x01, 0xFF}, reader1, nil)
	if err != nil {
		t.Fatalf("unexpected error in setup: %v", err)
	}

	// Attempt to register list where second item would shadow
	headerBytesList := [][]byte{
		{0x01, 0xFE}, // This should succeed (non-overlapping)
		{0x01},       // This should fail (would shadow existing 0x01, 0xFF)
	}

	err = registry.RegisterAll(headerBytesList, reader2, nil)
	if err == nil {
		t.Fatal("expected error due to shadowing")
	}
	if !strings.Contains(err.Error(), "shadow") {
		t.Errorf("expected shadowing error, got: %v", err)
	}

	// Verify first item was registered
	foundReader := registry.GetReader([]byte{0x01, 0xFE})
	if foundReader == nil {
		t.Error("expected first item to be registered before error")
	}
	if foundReader != reader2 {
		t.Error("wrong reader for first item")
	}

	// Verify second item was NOT registered
	foundReader = registry.GetReader([]byte{0x01, 0x00, 0x00})
	if foundReader != nil {
		t.Error("second item should not have been registered")
	}
}

func TestSupportedHeaderBytes(t *testing.T) {
	registry := NewDAProviderRegistry()
	reader := &mockReader{name: "reader"}

	expected := [][]byte{
		{0x80},
		{0x88},
		{0x01, 0xFF},
	}

	for _, hb := range expected {
		err := registry.Register(hb, reader, nil)
		if err != nil {
			t.Fatalf("unexpected error registering %x: %v", hb, err)
		}
	}

	supported := registry.SupportedHeaderBytes()
	if len(supported) != len(expected) {
		t.Fatalf("expected %d supported bytes, got %d", len(expected), len(supported))
	}

	// Create a map for easier comparison (order doesn't matter)
	supportedMap := make(map[string]bool)
	for _, hb := range supported {
		supportedMap[string(hb)] = true
	}

	for _, hb := range expected {
		if !supportedMap[string(hb)] {
			t.Errorf("expected header bytes %x not found in supported list", hb)
		}
	}
}

func TestGetValidator_PrefixMatching(t *testing.T) {
	registry := NewDAProviderRegistry()
	validator := &mockValidator{name: "validator"}
	headerBytes := []byte{0x01, 0xFF}

	err := registry.Register(headerBytes, nil, validator)
	if err != nil {
		t.Fatalf("unexpected error registering: %v", err)
	}

	tests := []struct {
		name        string
		certificate []byte
		shouldFind  bool
		description string
	}{
		{
			name:        "exact match",
			certificate: []byte{0x01, 0xFF},
			shouldFind:  true,
			description: "exact match should find validator",
		},
		{
			name:        "certificate with suffix",
			certificate: []byte{0x01, 0xFF, 0x11, 0x22, 0x33},
			shouldFind:  true,
			description: "prefix match should find validator",
		},
		{
			name:        "different second byte",
			certificate: []byte{0x01, 0xFE, 0x11, 0x22},
			shouldFind:  false,
			description: "different prefix should not match",
		},
		{
			name:        "too short",
			certificate: []byte{0x01},
			shouldFind:  false,
			description: "certificate shorter than registered bytes should not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foundValidator := registry.GetValidator(tt.certificate)
			found := foundValidator != nil
			if found != tt.shouldFind {
				t.Errorf("%s: expected found=%v, got %v", tt.description, tt.shouldFind, found)
			}
			if tt.shouldFind && foundValidator != validator {
				t.Errorf("found wrong validator")
			}
		})
	}
}

func TestRegister_SameObjectAsReaderAndValidator(t *testing.T) {
	registry := NewDAProviderRegistry()

	// Use mockReader which can act as both (though not implementing Validator interface,
	// we're just testing the registration mechanism)
	provider := &mockReader{name: "provider"}
	headerBytes := []byte{0x01, 0xFF}

	// Register same object as both reader and validator
	err := registry.Register(headerBytes, provider, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify reader can be retrieved
	foundReader := registry.GetReader(headerBytes)
	if foundReader == nil {
		t.Fatal("expected to find reader")
	}
	if foundReader != provider {
		t.Error("found wrong reader")
	}
}

func TestRegister_ReaderOnly(t *testing.T) {
	registry := NewDAProviderRegistry()
	reader := &mockReader{name: "reader"}
	headerBytes := []byte{0x50}

	err := registry.Register(headerBytes, reader, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify reader works
	if foundReader := registry.GetReader(headerBytes); foundReader != reader {
		t.Error("reader not found or wrong reader")
	}

	// Verify no validator registered
	if foundValidator := registry.GetValidator(headerBytes); foundValidator != nil {
		t.Error("expected no validator to be registered")
	}
}

func TestRegister_ValidatorOnly(t *testing.T) {
	registry := NewDAProviderRegistry()
	validator := &mockValidator{name: "validator"}
	headerBytes := []byte{0x01, 0xAA}

	err := registry.Register(headerBytes, nil, validator)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify validator works
	if foundValidator := registry.GetValidator(headerBytes); foundValidator != validator {
		t.Error("validator not found or wrong validator")
	}

	// Verify no reader registered
	if foundReader := registry.GetReader(headerBytes); foundReader != nil {
		t.Error("expected no reader to be registered")
	}
}
