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
			registry := NewReaderRegistry()
			reader1 := &mockReader{name: "reader1"}
			reader2 := &mockReader{name: "reader2"}

			// Register first header bytes
			err := registry.Register(tt.first, reader1)
			if err != nil {
				t.Fatalf("unexpected error registering first: %v", err)
			}

			// Attempt to register second header bytes
			err = registry.Register(tt.second, reader2)
			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
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
			registry := NewReaderRegistry()
			reader := &mockReader{name: "reader"}

			for i, hb := range tt.headerBytes {
				err := registry.Register(hb, reader)
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
			registry := NewReaderRegistry()
			reader1 := &mockReader{name: "reader1"}
			headerBytes := []byte{0x01, 0xFF}

			// Register once
			err := registry.Register(headerBytes, reader1)
			if err != nil {
				t.Fatalf("unexpected error on first registration: %v", err)
			}

			// Attempt to register again - should fail regardless of reader
			err = registry.Register(headerBytes, tt.reader)
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
		headerBytes   []byte
		errorContains string
	}{
		{
			name:          "nil reader",
			reader:        nil,
			headerBytes:   []byte{0x01},
			errorContains: "cannot register nil reader",
		},
		{
			name:          "empty header bytes",
			reader:        &mockReader{name: "reader"},
			headerBytes:   []byte{},
			errorContains: "cannot register empty header bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewReaderRegistry()
			err := registry.Register(tt.headerBytes, tt.reader)
			if err == nil {
				t.Fatal("expected error but got none")
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q, got: %v", tt.errorContains, err)
			}
		})
	}
}

func TestGetByHeaderBytes_PrefixMatching(t *testing.T) {
	registry := NewReaderRegistry()
	reader := &mockReader{name: "reader"}
	headerBytes := []byte{0x01, 0xFF}

	err := registry.Register(headerBytes, reader)
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
			foundReader, found := registry.GetByHeaderBytes(tt.message)
			if found != tt.shouldFind {
				t.Errorf("%s: expected found=%v, got %v", tt.description, tt.shouldFind, found)
			}
			if tt.shouldFind && foundReader != reader {
				t.Errorf("found wrong reader")
			}
		})
	}
}

func TestGetByHeaderBytes_FirstMatch(t *testing.T) {
	registry := NewReaderRegistry()
	reader1 := &mockReader{name: "reader1"}
	reader2 := &mockReader{name: "reader2"}

	// Register two non-overlapping patterns
	err := registry.Register([]byte{0x80}, reader1)
	if err != nil {
		t.Fatalf("unexpected error registering first: %v", err)
	}
	err = registry.Register([]byte{0x88}, reader2)
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
			foundReader, found := registry.GetByHeaderBytes(tt.message)
			if !found {
				t.Fatal("expected to find reader")
			}
			if foundReader != tt.expectedReader {
				t.Errorf("found wrong reader: expected %v, got %v", tt.expectedReader, foundReader)
			}
		})
	}
}

func TestRegisterAll_Success(t *testing.T) {
	registry := NewReaderRegistry()
	reader := &mockReader{name: "reader"}
	headerBytesList := [][]byte{
		{0x80},
		{0x88},
	}

	err := registry.RegisterAll(headerBytesList, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify both are registered and can be looked up
	for _, hb := range headerBytesList {
		foundReader, found := registry.GetByHeaderBytes(hb)
		if !found {
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
	registry := NewReaderRegistry()
	reader1 := &mockReader{name: "reader1"}
	reader2 := &mockReader{name: "reader2"}

	// Pre-register a pattern
	err := registry.Register([]byte{0x01, 0xFF}, reader1)
	if err != nil {
		t.Fatalf("unexpected error in setup: %v", err)
	}

	// Attempt to register list where second item would shadow
	headerBytesList := [][]byte{
		{0x01, 0xFE}, // This should succeed (non-overlapping)
		{0x01},       // This should fail (would shadow existing 0x01, 0xFF)
	}

	err = registry.RegisterAll(headerBytesList, reader2)
	if err == nil {
		t.Fatal("expected error due to shadowing")
	}
	if !strings.Contains(err.Error(), "shadow") {
		t.Errorf("expected shadowing error, got: %v", err)
	}

	// Verify first item was registered
	foundReader, found := registry.GetByHeaderBytes([]byte{0x01, 0xFE})
	if !found {
		t.Error("expected first item to be registered before error")
	}
	if foundReader != reader2 {
		t.Error("wrong reader for first item")
	}

	// Verify second item was NOT registered
	_, found = registry.GetByHeaderBytes([]byte{0x01, 0x00, 0x00})
	if found {
		t.Error("second item should not have been registered")
	}
}

func TestSupportedHeaderBytes(t *testing.T) {
	registry := NewReaderRegistry()
	reader := &mockReader{name: "reader"}

	expected := [][]byte{
		{0x80},
		{0x88},
		{0x01, 0xFF},
	}

	for _, hb := range expected {
		err := registry.Register(hb, reader)
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
