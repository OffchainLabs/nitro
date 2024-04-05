// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package daprovider

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/das/dastree"
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
	cert, err := DeserializeDASCertFrom(bytes.NewReader(sequencerMsg[40:]))
	if err != nil {
		log.Error("Failed to deserialize DAS message", "err", err)
		return nil, nil
	}
	version := cert.Version

	if version >= 2 {
		log.Error("Your node software is probably out of date", "certificateVersion", version)
		return nil, nil
	}

	getByHash := func(ctx context.Context, hash common.Hash) ([]byte, error) {
		newHash := hash
		if version == 0 {
			newHash = dastree.FlatHashToTreeHash(hash)
		}

		preimage, err := d.dasReader.GetByHash(ctx, newHash)
		if err != nil && hash != newHash {
			log.Debug("error fetching new style hash, trying old", "new", newHash, "old", hash, "err", err)
			preimage, err = d.dasReader.GetByHash(ctx, hash)
		}
		if err != nil {
			return nil, err
		}

		switch {
		case version == 0 && crypto.Keccak256Hash(preimage) != hash:
			fallthrough
		case version == 1 && dastree.Hash(preimage) != hash:
			log.Error(
				"preimage mismatch for hash",
				"hash", hash, "err", ErrHashMismatch, "version", version,
			)
			return nil, ErrHashMismatch
		}
		return preimage, nil
	}

	keysetPreimage, err := getByHash(ctx, cert.KeysetHash)
	if err != nil {
		log.Error("Couldn't get keyset", "err", err)
		return nil, err
	}
	if preimageRecorder != nil {
		dastree.RecordHash(preimageRecorder, keysetPreimage)
	}

	keyset, err := DeserializeKeyset(bytes.NewReader(keysetPreimage), !validateSeqMsg)
	if err != nil {
		return nil, fmt.Errorf("%w. Couldn't deserialize keyset, err: %w, keyset hash: %x batch num: %d", ErrSeqMsgValidation, err, cert.KeysetHash, batchNum)
	}
	err = keyset.VerifySignature(cert.SignersMask, cert.SerializeSignableFields(), cert.Sig)
	if err != nil {
		log.Error("Bad signature on DAS batch", "err", err)
		return nil, nil
	}

	maxTimestamp := binary.BigEndian.Uint64(sequencerMsg[8:16])
	if cert.Timeout < maxTimestamp+MinLifetimeSecondsForDataAvailabilityCert {
		log.Error("Data availability cert expires too soon", "err", "")
		return nil, nil
	}

	dataHash := cert.DataHash
	payload, err := getByHash(ctx, dataHash)
	if err != nil {
		log.Error("Couldn't fetch DAS batch contents", "err", err)
		return nil, err
	}

	if preimageRecorder != nil {
		if version == 0 {
			treeLeaf := dastree.FlatHashToTreeLeaf(dataHash)
			preimageRecorder(dataHash, payload, arbutil.Keccak256PreimageType)
			preimageRecorder(crypto.Keccak256Hash(treeLeaf), treeLeaf, arbutil.Keccak256PreimageType)
		} else {
			dastree.RecordHash(preimageRecorder, payload)
		}
	}

	return payload, nil
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
