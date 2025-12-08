// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package server_api

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// SupportedHeaderBytesResult is the result struct that data availability providers should use to respond with their supported header bytes
type SupportedHeaderBytesResult struct {
	HeaderBytes hexutil.Bytes `json:"headerBytes,omitempty"`
}

// MaxMessageSizeResult is the result struct for daprovider_getMaxMessageSize
type MaxMessageSizeResult struct {
	MaxSize int `json:"maxSize"`
}

// StoreResult is the result struct that data availability providers should use to respond with a commitment to a Store request for posting batch data to their DA service
type StoreResult struct {
	SerializedDACert hexutil.Bytes `json:"serialized-da-cert,omitempty"`
}

// GenerateReadPreimageProofResult is the result struct that data availability providers
// should use to respond with a proof for a specific preimage
type GenerateReadPreimageProofResult struct {
	Proof hexutil.Bytes `json:"proof,omitempty"`
}

// GenerateCertificateValidityProofResult is the result struct that data availability providers should use to respond with validity proof
type GenerateCertificateValidityProofResult struct {
	Proof hexutil.Bytes `json:"proof,omitempty"`
}
