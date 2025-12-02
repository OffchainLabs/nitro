// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/util/containers"
)

type Validator struct {
	storage       *InMemoryStorage
	l1Client      *ethclient.Client
	validatorAddr common.Address
}

func NewValidator(l1Client *ethclient.Client, validatorAddr common.Address) *Validator {
	return &Validator{
		storage:       GetInMemoryStorage(),
		l1Client:      l1Client,
		validatorAddr: validatorAddr,
	}
}

// GenerateReadPreimageProof creates a ReadPreimage proof for ReferenceDA
// The proof enhancer will prepend the standardized header [certKeccak256, offset, certSize, certificate]
// So we only need to return the custom data: [Version(1), PreimageSize(8), PreimageData]
func (v *Validator) generateReadPreimageProofInternal(ctx context.Context, offset uint64, certificate []byte) ([]byte, error) {
	// Deserialize certificate to extract data hash
	cert, err := Deserialize(certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize certificate: %w", err)
	}

	// Extract data hash (SHA256) from certificate
	dataHash := cert.DataHash

	// Get preimage from storage using SHA256 hash
	preimage, err := v.storage.GetByHash(dataHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get preimage: %w", err)
	}
	if preimage == nil {
		return nil, fmt.Errorf("preimage not found for hash %x", dataHash)
	}

	// Build custom proof data: [Version(1), PreimageSize(8), PreimageData]
	// The certificate is NOT included here as it's already in the standardized header
	proof := make([]byte, 1+8+len(preimage))
	proof[0] = 1 // Version
	binary.BigEndian.PutUint64(proof[1:9], uint64(len(preimage)))
	copy(proof[9:], preimage)

	return proof, nil
}

// GenerateReadPreimageProof creates a ReadPreimage proof for ReferenceDA
// The proof enhancer will prepend the standardized header [certKeccak256, offset, certSize, certificate]
// So we only need to return the custom data: [Version(1), PreimageSize(8), PreimageData]
func (v *Validator) GenerateReadPreimageProof(offset uint64, certificate []byte) containers.PromiseInterface[daprovider.PreimageProofResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PreimageProofResult, error) {
		proof, err := v.generateReadPreimageProofInternal(ctx, offset, certificate)
		return daprovider.PreimageProofResult{Proof: proof}, err
	})
}

// GenerateCertificateValidityProof creates a certificate validity proof for ReferenceDA
// The ReferenceDA implementation returns a two-byte proof with:
// - claimedValid (1 byte): 1 if valid, 0 if invalid
// - version (1 byte): 0x01 for version 1
//
// This validates the certificate signature against trusted signers from the contract.
// Invalid certificates (wrong format, untrusted signer) return claimedValid=0.
// Only transient errors (like RPC failures) return an error.
func (v *Validator) generateCertificateValidityProofInternal(ctx context.Context, certificate []byte) ([]byte, error) {
	// Try to deserialize certificate
	cert, err := Deserialize(certificate)
	if err != nil {
		// Certificate is malformed (wrong length, etc.)
		// We return invalid proof rather than error for validation failures
		return []byte{0, 0x01}, nil //nolint:nilerr // Invalid certificate, version 1
	}

	// Create contract binding
	validator, err := localgen.NewReferenceDAProofValidator(v.validatorAddr, v.l1Client)
	if err != nil {
		// This is a transient error - can't connect to contract
		return nil, fmt.Errorf("failed to create validator binding: %w", err)
	}

	// Check if signer is trusted using contract
	signer, err := cert.RecoverSigner()
	if err != nil {
		// Invalid signature - can't recover signer
		// We return invalid proof rather than error for validation failures
		return []byte{0, 0x01}, nil //nolint:nilerr // Invalid certificate, version 1
	}

	// Query contract to check if signer is trusted
	isTrusted, err := validator.TrustedSigners(&bind.CallOpts{Context: ctx}, signer)
	if err != nil {
		// This is a transient error - RPC call failed
		return nil, fmt.Errorf("failed to check trusted signer: %w", err)
	}

	if !isTrusted {
		// Signer is not trusted
		return []byte{0, 0x01}, nil // Invalid certificate, version 1
	}

	// Certificate is valid (signed by trusted signer)
	return []byte{1, 0x01}, nil // Valid certificate, version 1
}

// GenerateCertificateValidityProof creates a certificate validity proof for ReferenceDA
// The ReferenceDA implementation returns a two-byte proof with:
// - claimedValid (1 byte): 1 if valid, 0 if invalid
// - version (1 byte): 0x01 for version 1
//
// This validates the certificate signature against trusted signers from the contract.
// Invalid certificates (wrong format, untrusted signer) return claimedValid=0.
// Only transient errors (like RPC failures) return an error.
func (v *Validator) GenerateCertificateValidityProof(certificate []byte) containers.PromiseInterface[daprovider.ValidityProofResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.ValidityProofResult, error) {
		proof, err := v.generateCertificateValidityProofInternal(ctx, certificate)
		return daprovider.ValidityProofResult{Proof: proof}, err
	})
}
