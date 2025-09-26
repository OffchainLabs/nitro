// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package referenceda

import (
	"crypto/sha256"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/solgen/go/ospgen"
	"github.com/offchainlabs/nitro/util/signature"
)

// Certificate represents a ReferenceDA certificate with signature
type Certificate struct {
	Header   byte
	DataHash [32]byte
	V        uint8
	R        [32]byte
	S        [32]byte
}

// NewCertificate creates a certificate from data and signs it
func NewCertificate(data []byte, signer signature.DataSignerFunc) (*Certificate, error) {
	dataHash := sha256.Sum256(data)

	sig, err := signer(dataHash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign data hash: %w", err)
	}

	cert := &Certificate{
		Header:   daprovider.DACertificateMessageHeaderFlag,
		DataHash: dataHash,
		V:        sig[64] + 27,
	}
	copy(cert.R[:], sig[0:32])
	copy(cert.S[:], sig[32:64])

	return cert, nil
}

// Serialize converts certificate to bytes (98 bytes total)
func (c *Certificate) Serialize() []byte {
	result := make([]byte, 98)
	result[0] = c.Header
	copy(result[1:33], c.DataHash[:])
	result[33] = c.V
	copy(result[34:66], c.R[:])
	copy(result[66:98], c.S[:])
	return result
}

// Deserialize creates a certificate from bytes
func Deserialize(data []byte) (*Certificate, error) {
	if len(data) != 98 {
		return nil, fmt.Errorf("invalid certificate length: expected 98, got %d", len(data))
	}

	cert := &Certificate{
		Header: data[0],
		V:      data[33],
	}
	copy(cert.DataHash[:], data[1:33])
	copy(cert.R[:], data[34:66])
	copy(cert.S[:], data[66:98])

	if cert.Header != daprovider.DACertificateMessageHeaderFlag {
		return nil, fmt.Errorf("invalid certificate header: %x", cert.Header)
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
func (c *Certificate) ValidateWithContract(validator *ospgen.ReferenceDAProofValidator, opts *bind.CallOpts) error {
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
