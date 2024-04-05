// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package daprovider

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/blobs"
)

type Reader interface {
	// IsValidHeaderByte returns true if the given headerByte has bits corresponding to the DA provider
	IsValidHeaderByte(headerByte byte) bool

	// RecoverPayloadFromBatch fetches the underlying payload from the DA provider given the batch header information
	RecoverPayloadFromBatch(
		ctx context.Context,
		batchNum uint64,
		batchBlockHash common.Hash,
		sequencerMsg []byte,
		preimageRecorder PreimageRecorder,
		validateSeqMsg bool,
	) ([]byte, error)
}

// NewReaderForDAS is generally meant to be only used by nitro.
// DA Providers should implement methods in the Reader interface independently
func NewReaderForDAS(dasReader DASReader) *readerForDAS {
	return &readerForDAS{dasReader: dasReader}
}

type readerForDAS struct {
	dasReader DASReader
}

func (d *readerForDAS) IsValidHeaderByte(headerByte byte) bool {
	return IsDASMessageHeaderByte(headerByte)
}

func (d *readerForDAS) RecoverPayloadFromBatch(
	ctx context.Context,
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
	preimageRecorder PreimageRecorder,
	validateSeqMsg bool,
) ([]byte, error) {
	return RecoverPayloadFromDasBatch(ctx, batchNum, sequencerMsg, d.dasReader, preimageRecorder, validateSeqMsg)
}

// NewReaderForBlobReader is generally meant to be only used by nitro.
// DA Providers should implement methods in the Reader interface independently
func NewReaderForBlobReader(blobReader BlobReader) *readerForBlobReader {
	return &readerForBlobReader{blobReader: blobReader}
}

type readerForBlobReader struct {
	blobReader BlobReader
}

func (b *readerForBlobReader) IsValidHeaderByte(headerByte byte) bool {
	return IsBlobHashesHeaderByte(headerByte)
}

func (b *readerForBlobReader) RecoverPayloadFromBatch(
	ctx context.Context,
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
	preimageRecorder PreimageRecorder,
	validateSeqMsg bool,
) ([]byte, error) {
	blobHashes := sequencerMsg[41:]
	if len(blobHashes)%len(common.Hash{}) != 0 {
		return nil, ErrInvalidBlobDataFormat
	}
	versionedHashes := make([]common.Hash, len(blobHashes)/len(common.Hash{}))
	for i := 0; i*32 < len(blobHashes); i += 1 {
		copy(versionedHashes[i][:], blobHashes[i*32:(i+1)*32])
	}
	kzgBlobs, err := b.blobReader.GetBlobs(ctx, batchBlockHash, versionedHashes)
	if err != nil {
		return nil, fmt.Errorf("failed to get blobs: %w", err)
	}
	if preimageRecorder != nil {
		for i, blob := range kzgBlobs {
			// Prevent aliasing `blob` when slicing it, as for range loops overwrite the same variable
			// Won't be necessary after Go 1.22 with https://go.dev/blog/loopvar-preview
			b := blob
			preimageRecorder(versionedHashes[i], b[:], arbutil.EthVersionedHashPreimageType)
		}
	}
	payload, err := blobs.DecodeBlobs(kzgBlobs)
	if err != nil {
		log.Warn("Failed to decode blobs", "batchBlockHash", batchBlockHash, "versionedHashes", versionedHashes, "err", err)
		return nil, nil
	}
	return payload, nil
}
