// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package daprovider

import (
	"fmt"
)

// registeredProvider associates a header byte with a reader and/or validator
type registeredProvider struct {
	headerByte byte
	reader     Reader
	validator  Validator
}

// DAProviderRegistry maintains a mapping of header bytes to their corresponding DA providers
type DAProviderRegistry struct {
	providers []registeredProvider
}

// NewDAProviderRegistry creates a new DA provider registry
func NewDAProviderRegistry() *DAProviderRegistry {
	return &DAProviderRegistry{
		providers: make([]registeredProvider, 0),
	}
}

// Register associates a header byte with a reader and/or validator
func (r *DAProviderRegistry) Register(headerByte byte, reader Reader, validator Validator) error {
	if reader == nil && validator == nil {
		return fmt.Errorf("cannot register with both reader and validator nil")
	}

	// Check for duplicate registrations
	for _, registered := range r.providers {
		if registered.headerByte == headerByte {
			return fmt.Errorf("header byte 0x%02x already registered", headerByte)
		}
	}

	r.providers = append(r.providers, registeredProvider{
		headerByte: headerByte,
		reader:     reader,
		validator:  validator,
	})
	return nil
}

// GetReader returns the reader associated with the given header byte
// Returns nil if no matching reader is found
func (r *DAProviderRegistry) GetReader(headerByte byte) Reader {
	for _, registered := range r.providers {
		if registered.headerByte == headerByte {
			return registered.reader
		}
	}
	return nil
}

// GetValidator returns the validator associated with the given header byte
// Returns nil if no matching validator is found
func (r *DAProviderRegistry) GetValidator(headerByte byte) Validator {
	for _, registered := range r.providers {
		if registered.headerByte == headerByte {
			return registered.validator
		}
	}
	return nil
}

// SupportedHeaderBytes returns all registered header bytes
func (r *DAProviderRegistry) SupportedHeaderBytes() []byte {
	result := make([]byte, 0, len(r.providers))
	for _, registered := range r.providers {
		result = append(result, registered.headerByte)
	}
	return result
}

// SetupDASReader registers a DAS reader and validator for the DAS header bytes (with and without Tree flag)
func (r *DAProviderRegistry) SetupDASReader(reader Reader, validator Validator) error {
	// Register for DAS without tree flag (0x80)
	if err := r.Register(DASMessageHeaderFlag, reader, validator); err != nil {
		return err
	}
	// Register for DAS with tree flag (0x88 = 0x80 | 0x08)
	return r.Register(DASMessageHeaderFlag|TreeDASMessageHeaderFlag, reader, validator)
}

// SetupBlobReader registers a blob reader for the blob header byte (no validator)
func (r *DAProviderRegistry) SetupBlobReader(reader Reader) error {
	return r.Register(BlobHashesHeaderFlag, reader, nil)
}

// SetupDACertificateReader registers a DA certificate reader and validator for the certificate header byte
func (r *DAProviderRegistry) SetupDACertificateReader(reader Reader, validator Validator) error {
	return r.Register(DACertificateMessageHeaderFlag, reader, validator)
}
