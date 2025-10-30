// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package daprovider

import (
	"bytes"
	"fmt"
)

// registeredReader associates a byte string prefix with a reader
type registeredReader struct {
	headerBytes []byte
	reader      Reader
}

// ReaderRegistry maintains a mapping of header byte strings to their corresponding readers
type ReaderRegistry struct {
	readers []registeredReader
}

// NewReaderRegistry creates a new reader registry
func NewReaderRegistry() *ReaderRegistry {
	return &ReaderRegistry{
		readers: make([]registeredReader, 0),
	}
}

// Register associates a header byte string with a reader
// Prevents shadowing: rejects registration if the new headerBytes is a prefix of any existing registration,
// or if any existing registration is a prefix of the new headerBytes
func (r *ReaderRegistry) Register(headerBytes []byte, reader Reader) error {
	if reader == nil {
		return fmt.Errorf("cannot register nil reader")
	}
	if len(headerBytes) == 0 {
		return fmt.Errorf("cannot register empty header bytes")
	}

	// Check for exact matches and shadowing
	for _, registered := range r.readers {
		// If exact match, always error
		if bytes.Equal(registered.headerBytes, headerBytes) {
			return fmt.Errorf("header bytes %x already registered", headerBytes)
		}

		// Check for shadowing (prefix relationships, excluding exact matches already handled above)
		// Check if new is prefix of existing
		if bytes.HasPrefix(registered.headerBytes, headerBytes) {
			return fmt.Errorf("header bytes %x would shadow existing registration %x", headerBytes, registered.headerBytes)
		}
		// Check if existing is prefix of new
		if bytes.HasPrefix(headerBytes, registered.headerBytes) {
			return fmt.Errorf("header bytes %x would be shadowed by existing registration %x", headerBytes, registered.headerBytes)
		}
	}

	// No conflicts found, add the registration
	r.readers = append(r.readers, registeredReader{
		headerBytes: headerBytes,
		reader:      reader,
	})
	return nil
}

// RegisterAll associates multiple header byte strings with a reader
func (r *ReaderRegistry) RegisterAll(headerBytesList [][]byte, reader Reader) error {
	for _, headerBytes := range headerBytesList {
		if err := r.Register(headerBytes, reader); err != nil {
			return err
		}
	}
	return nil
}

// GetByHeaderBytes returns the reader associated with a message by matching registered header byte prefixes
// Uses first-match strategy since shadowing prevention ensures at most one match
func (r *ReaderRegistry) GetByHeaderBytes(message []byte) (Reader, bool) {
	for _, registered := range r.readers {
		if bytes.HasPrefix(message, registered.headerBytes) {
			return registered.reader, true
		}
	}
	return nil, false
}

// SupportedHeaderBytes returns all registered header byte strings
func (r *ReaderRegistry) SupportedHeaderBytes() [][]byte {
	result := make([][]byte, 0, len(r.readers))
	for _, registered := range r.readers {
		result = append(result, registered.headerBytes)
	}
	return result
}

// SetupDASReader registers a DAS reader for the DAS header bytes (with and without Tree flag)
func (r *ReaderRegistry) SetupDASReader(reader Reader) error {
	// Register for DAS without tree flag (0x80)
	if err := r.Register([]byte{DASMessageHeaderFlag}, reader); err != nil {
		return err
	}
	// Register for DAS with tree flag (0x88 = 0x80 | 0x08)
	return r.Register([]byte{DASMessageHeaderFlag | TreeDASMessageHeaderFlag}, reader)
}

// SetupBlobReader registers a blob reader for the blob header byte
func (r *ReaderRegistry) SetupBlobReader(reader Reader) error {
	return r.Register([]byte{BlobHashesHeaderFlag}, reader)
}

// SetupDACertificateReader registers a DA certificate reader for the certificate header byte
func (r *ReaderRegistry) SetupDACertificateReader(reader Reader) error {
	return r.Register([]byte{DACertificateMessageHeaderFlag}, reader)
}
