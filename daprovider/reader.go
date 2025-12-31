// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package daprovider

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/blobs"
	"github.com/offchainlabs/nitro/util/containers"
)

// CertificateValidationError represents an error in certificate validation
type CertificateValidationError struct {
	Reason string
}

func (e *CertificateValidationError) Error() string {
	return e.Reason
}

// IsCertificateValidationError checks if an error is a certificate validation error
func IsCertificateValidationError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "certificate validation failed")
}

// PayloadResult contains the recovered payload data
type PayloadResult struct {
	Payload []byte
}

// PreimagesResult contains the collected preimages
type PreimagesResult struct {
	Preimages PreimagesMap
}

type Reader interface {
	// RecoverPayload fetches the underlying payload from the DA provider given the batch header information
	RecoverPayload(
		batchNum uint64,
		batchBlockHash common.Hash,
		sequencerMsg []byte,
	) containers.PromiseInterface[PayloadResult]

	// CollectPreimages collects preimages from the DA provider given the batch header information
	CollectPreimages(
		batchNum uint64,
		batchBlockHash common.Hash,
		sequencerMsg []byte,
	) containers.PromiseInterface[PreimagesResult]
}

// NewReaderForBlobReader is generally meant to be only used by nitro.
// DA Providers should implement methods in the Reader interface independently
func NewReaderForBlobReader(blobReader BlobReader) *readerForBlobReader {
	return &readerForBlobReader{blobReader: blobReader}
}

type readerForBlobReader struct {
	blobReader BlobReader
}

// recoverInternal is the shared implementation for both RecoverPayload and CollectPreimages
func (b *readerForBlobReader) recoverInternal(
	ctx context.Context,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
	needPayload bool,
	needPreimages bool,
) ([]byte, PreimagesMap, error) {
	blobHashes := sequencerMsg[41:]
	if len(blobHashes)%len(common.Hash{}) != 0 {
		return nil, nil, ErrInvalidBlobDataFormat
	}
	versionedHashes := make([]common.Hash, len(blobHashes)/len(common.Hash{}))
	for i := 0; i*32 < len(blobHashes); i += 1 {
		copy(versionedHashes[i][:], blobHashes[i*32:(i+1)*32])
	}
	kzgBlobs, err := b.blobReader.GetBlobs(ctx, batchBlockHash, versionedHashes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get blobs: %w", err)
	}

	var preimages PreimagesMap
	if needPreimages {
		preimages = make(PreimagesMap)
		preimageRecorder := RecordPreimagesTo(preimages)
		for i, blob := range kzgBlobs {
			// Prevent aliasing `blob` when slicing it, as for range loops overwrite the same variable
			// Won't be necessary after Go 1.22 with https://go.dev/blog/loopvar-preview
			b := blob
			preimageRecorder(versionedHashes[i], b[:], arbutil.EthVersionedHashPreimageType)
		}
	}

	var payload []byte
	if needPayload {
		payload, err = blobs.DecodeBlobs(kzgBlobs)
		if err != nil {
			log.Warn("Failed to decode blobs", "batchBlockHash", batchBlockHash, "versionedHashes", versionedHashes, "err", err)
			return nil, nil, nil
		}
	}

	return payload, preimages, nil
}

// RecoverPayload fetches the underlying payload from the DA provider
func (b *readerForBlobReader) RecoverPayload(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[PayloadResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (PayloadResult, error) {
		payload, _, err := b.recoverInternal(ctx, batchBlockHash, sequencerMsg, true, false)
		return PayloadResult{Payload: payload}, err
	})
}

// CollectPreimages collects preimages from the DA provider
func (b *readerForBlobReader) CollectPreimages(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[PreimagesResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (PreimagesResult, error) {
		_, preimages, err := b.recoverInternal(ctx, batchBlockHash, sequencerMsg, false, true)
		return PreimagesResult{Preimages: preimages}, err
	})
}
