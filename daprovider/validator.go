// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package daprovider

import (
	"github.com/offchainlabs/nitro/util/containers"
)

// PreimageProofResult contains the generated preimage proof
type PreimageProofResult struct {
	Proof []byte
}

// ValidityProofResult contains the generated validity proof
type ValidityProofResult struct {
	Proof []byte
}

// Validator defines the interface for custom data availability systems.
// This interface is used to generate proofs for DACertificate certificates and preimages.
type Validator interface {
	// GenerateReadPreimageProof generates a proof for a specific preimage at a given offset.
	// The proof format depends on the implementation and must be compatible with the Solidity
	// IDACertificateValidator contract.
	GenerateReadPreimageProof(offset uint64, certificate []byte) containers.PromiseInterface[PreimageProofResult]

	// GenerateCertificateValidityProof returns a proof of whether the certificate
	// is valid according to the DA system's rules.
	GenerateCertificateValidityProof(certificate []byte) containers.PromiseInterface[ValidityProofResult]
}
