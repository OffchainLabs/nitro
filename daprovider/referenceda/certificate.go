// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	"crypto/sha256"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/util/signature"
)

// ReferenceDAProviderType identifies this as a ReferenceDA certificate.
// It follows the DACertificateMessageHeaderFlag in the certificate format.
// This allows for different DA providers using the CustomDA system to
// differentiate themselves.
// It is not required for external DA providers to use a byte to distinguish
// providers if there will only ever be one provider on a given chain.
// Providers can also use more than one byte, using a single byte is just
// an example. Unrelated providers don't need to coordinate their header
// bytes unless they intend to coexist on the same chain.
const ReferenceDAProviderType byte = 0xFF

// Certificate represents a ReferenceDA certificate with signature
type Certificate struct {
	Header       byte
	ProviderType byte
	DataHash     [32]byte
	V            uint8
	R            [32]byte
	S            [32]byte
}

// NewCertificate creates a certificate from data and signs it
func NewCertificate(data []byte, signer signature.DataSignerFunc) (*Certificate, error) {
	dataHash := sha256.Sum256(data)

	sig, err := signer(dataHash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign data hash: %w", err)
	}

	cert := &Certificate{
		Header:       daprovider.DACertificateMessageHeaderFlag,
		ProviderType: ReferenceDAProviderType,
		DataHash:     dataHash,
		V:            sig[64] + 27,
	}
	copy(cert.R[:], sig[0:32])
	copy(cert.S[:], sig[32:64])

	return cert, nil
}

// Serialize converts certificate to bytes (99 bytes total)
func (c *Certificate) Serialize() []byte {
	result := make([]byte, 99)
	result[0] = c.Header
	result[1] = c.ProviderType
	copy(result[2:34], c.DataHash[:])
	result[34] = c.V
	copy(result[35:67], c.R[:])
	copy(result[67:99], c.S[:])
	return result
}

// Deserialize creates a certificate from bytes
func Deserialize(data []byte) (*Certificate, error) {
	if len(data) != 99 {
		return nil, fmt.Errorf("invalid certificate length: expected 99, got %d", len(data))
	}

	cert := &Certificate{
		Header:       data[0],
		ProviderType: data[1],
		V:            data[34],
	}
	copy(cert.DataHash[:], data[2:34])
	copy(cert.R[:], data[35:67])
	copy(cert.S[:], data[67:99])

	if cert.Header != daprovider.DACertificateMessageHeaderFlag {
		return nil, fmt.Errorf("invalid certificate header: %x", cert.Header)
	}

	if cert.ProviderType != ReferenceDAProviderType {
		return nil, fmt.Errorf("invalid provider type: expected %x, got %x", ReferenceDAProviderType, cert.ProviderType)
	}

	return cert, nil
}

// RecoverSigner recovers the signer address from the certificate
func (c *Certificate) RecoverSigner() (common.Address, error) {
	if c.V < 27 {
		return common.Address{}, fmt.Errorf("invalid signature V value: %d (must be >= 27)", c.V)
	}

	sig := make([]byte, 65)
	copy(sig[0:32], c.R[:])
	copy(sig[32:64], c.S[:])
	sig[64] = c.V - 27

	pubKey, err := crypto.SigToPub(c.DataHash[:], sig)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to recover signer: %w", err)
	}

	return crypto.PubkeyToAddress(*pubKey), nil
}

// ValidateWithContract checks if the certificate is signed by a trusted signer using the contract
func (c *Certificate) ValidateWithContract(validator *localgen.ReferenceDAProofValidator, opts *bind.CallOpts) error {
	signer, err := c.RecoverSigner()
	if err != nil {
		return err
	}

	isTrusted, err := validator.TrustedSigners(opts, signer)
	if err != nil {
		return fmt.Errorf("failed to check trusted signer: %w", err)
	}

	if !isTrusted {
		return fmt.Errorf("certificate signed by untrusted signer: %s", signer.Hex())
	}

	return nil
}
