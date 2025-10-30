// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package daprovider

import (
	"bytes"
	"fmt"
)

// registeredProvider associates a byte string prefix with a reader and/or validator
type registeredProvider struct {
	headerBytes []byte
	reader      Reader
	validator   Validator
}

// DAProviderRegistry maintains a mapping of header byte strings to their corresponding DA providers
type DAProviderRegistry struct {
	providers []registeredProvider
}

// NewDAProviderRegistry creates a new DA provider registry
func NewDAProviderRegistry() *DAProviderRegistry {
	return &DAProviderRegistry{
		providers: make([]registeredProvider, 0),
	}
}

// Register associates a header byte string with a reader and/or validator
// Prevents shadowing: rejects registration if the new headerBytes is a prefix of any existing registration,
// or if any existing registration is a prefix of the new headerBytes
func (r *DAProviderRegistry) Register(headerBytes []byte, reader Reader, validator Validator) error {
	if reader == nil && validator == nil {
		return fmt.Errorf("cannot register with both reader and validator nil")
	}
	if len(headerBytes) == 0 {
		return fmt.Errorf("cannot register empty header bytes")
	}

	// Check for exact matches and shadowing
	for _, registered := range r.providers {
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
	r.providers = append(r.providers, registeredProvider{
		headerBytes: headerBytes,
		reader:      reader,
		validator:   validator,
	})
	return nil
}

// RegisterAll associates multiple header byte strings with a reader and validator
func (r *DAProviderRegistry) RegisterAll(headerBytesList [][]byte, reader Reader, validator Validator) error {
	for _, headerBytes := range headerBytesList {
		if err := r.Register(headerBytes, reader, validator); err != nil {
			return err
		}
	}
	return nil
}

// GetReader returns the reader associated with a message by matching registered header byte prefixes
// Uses first-match strategy since shadowing prevention ensures at most one match
// Returns nil if no matching reader is found
func (r *DAProviderRegistry) GetReader(message []byte) Reader {
	for _, registered := range r.providers {
		if bytes.HasPrefix(message, registered.headerBytes) {
			return registered.reader
		}
	}
	return nil
}

// GetValidator returns the validator associated with a certificate by matching registered header byte prefixes
// Uses first-match strategy since shadowing prevention ensures at most one match
// Returns nil if no matching validator is found
func (r *DAProviderRegistry) GetValidator(certificate []byte) Validator {
	for _, registered := range r.providers {
		if bytes.HasPrefix(certificate, registered.headerBytes) {
			return registered.validator
		}
	}
	return nil
}

// SupportedHeaderBytes returns all registered header byte strings
func (r *DAProviderRegistry) SupportedHeaderBytes() [][]byte {
	result := make([][]byte, 0, len(r.providers))
	for _, registered := range r.providers {
		result = append(result, registered.headerBytes)
	}
	return result
}

// SetupDASReader registers a DAS reader and validator for the DAS header bytes (with and without Tree flag)
func (r *DAProviderRegistry) SetupDASReader(reader Reader, validator Validator) error {
	// Register for DAS without tree flag (0x80)
	if err := r.Register([]byte{DASMessageHeaderFlag}, reader, validator); err != nil {
		return err
	}
	// Register for DAS with tree flag (0x88 = 0x80 | 0x08)
	return r.Register([]byte{DASMessageHeaderFlag | TreeDASMessageHeaderFlag}, reader, validator)
}

// SetupBlobReader registers a blob reader for the blob header byte (no validator)
func (r *DAProviderRegistry) SetupBlobReader(reader Reader) error {
	return r.Register([]byte{BlobHashesHeaderFlag}, reader, nil)
}

// SetupDACertificateReader registers a DA certificate reader and validator for the certificate header byte
func (r *DAProviderRegistry) SetupDACertificateReader(reader Reader, validator Validator) error {
	return r.Register([]byte{DACertificateMessageHeaderFlag}, reader, validator)
}
