// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package daprovider

import (
	"fmt"
)

// ReaderRegistry maintains a mapping of header bytes to their corresponding readers
type ReaderRegistry struct {
	readers map[byte]Reader
}

// NewReaderRegistry creates a new reader registry
func NewReaderRegistry() *ReaderRegistry {
	return &ReaderRegistry{
		readers: make(map[byte]Reader),
	}
}

// Register associates a header byte with a reader
func (r *ReaderRegistry) Register(headerByte byte, reader Reader) error {
	if reader == nil {
		return fmt.Errorf("cannot register nil reader")
	}
	if existing, exists := r.readers[headerByte]; exists && existing != reader {
		return fmt.Errorf("header byte 0x%02x already registered", headerByte)
	}
	r.readers[headerByte] = reader
	return nil
}

// RegisterAll associates multiple header bytes with a reader
func (r *ReaderRegistry) RegisterAll(headerBytes []byte, reader Reader) error {
	for _, headerByte := range headerBytes {
		if err := r.Register(headerByte, reader); err != nil {
			return err
		}
	}
	return nil
}

// GetByHeaderByte returns the reader associated with the given header byte
func (r *ReaderRegistry) GetByHeaderByte(headerByte byte) (Reader, bool) {
	reader, exists := r.readers[headerByte]
	return reader, exists
}

// SupportedHeaderBytes returns all registered header bytes
func (r *ReaderRegistry) SupportedHeaderBytes() []byte {
	bytes := make([]byte, 0, len(r.readers))
	for b := range r.readers {
		bytes = append(bytes, b)
	}
	return bytes
}

// SetupDASReader registers a DAS reader for the DAS header bytes (with and without Tree flag)
func (r *ReaderRegistry) SetupDASReader(reader Reader) error {
	// Register for DAS without tree flag (0x80)
	if err := r.Register(DASMessageHeaderFlag, reader); err != nil {
		return err
	}
	// Register for DAS with tree flag (0x88 = 0x80 | 0x08)
	return r.Register(DASMessageHeaderFlag|TreeDASMessageHeaderFlag, reader)
}

// SetupBlobReader registers a blob reader for the blob header byte
func (r *ReaderRegistry) SetupBlobReader(reader Reader) error {
	return r.Register(BlobHashesHeaderFlag, reader)
}

// SetupDACertificateReader registers a DA certificate reader for the certificate header byte
func (r *ReaderRegistry) SetupDACertificateReader(reader Reader) error {
	return r.Register(DACertificateMessageHeaderFlag, reader)
}
