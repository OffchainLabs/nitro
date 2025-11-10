// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package daprovider

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
)

type BlobReader interface {
	GetBlobs(
		ctx context.Context,
		batchBlockHash common.Hash,
		versionedHashes []common.Hash,
	) ([]kzg4844.Blob, error)
	Initialize(ctx context.Context) error
}

type PreimagesMap map[arbutil.PreimageType]map[common.Hash][]byte

// PreimageRecorder is used to add (key,value) pair to the map accessed by key = ty of a bigger map, preimages.
// If ty doesn't exist as a key in the preimages map, then it is intialized to map[common.Hash][]byte and then (key,value) pair is added
type PreimageRecorder func(key common.Hash, value []byte, ty arbutil.PreimageType)

// RecordPreimagesTo takes in preimages map and returns a function that can be used
// In recording (hash,preimage) key value pairs into preimages map, when fetching payload through RecoverPayloadFromBatch
func RecordPreimagesTo(preimages PreimagesMap) PreimageRecorder {
	if preimages == nil {
		return nil
	}
	return func(key common.Hash, value []byte, ty arbutil.PreimageType) {
		if preimages[ty] == nil {
			preimages[ty] = make(map[common.Hash][]byte)
		}
		preimages[ty][key] = value
	}
}

var (
	ErrNoBlobReader          = errors.New("blob batch payload was encountered but no BlobReader was configured")
	ErrInvalidBlobDataFormat = errors.New("blob batch data is not a list of hashes as expected")
	ErrSeqMsgValidation      = errors.New("error validating recovered payload from batch")
)

type KeysetValidationMode uint8

const KeysetValidate KeysetValidationMode = 0
const KeysetPanicIfInvalid KeysetValidationMode = 1
const KeysetDontValidate KeysetValidationMode = 2

// DASMessageHeaderFlag indicates that this data is a certificate for the data availability service,
// which will retrieve the full batch data.
const DASMessageHeaderFlag byte = 0x80

// TreeDASMessageHeaderFlag indicates that this DAS certificate data employs the new merkelization strategy.
// Ignored when DASMessageHeaderFlag is not set.
const TreeDASMessageHeaderFlag byte = 0x08

// L1AuthenticatedMessageHeaderFlag indicates that this message was authenticated by L1. Currently unused.
const L1AuthenticatedMessageHeaderFlag byte = 0x40

// ZeroheavyMessageHeaderFlag indicates that this message is zeroheavy-encoded.
const ZeroheavyMessageHeaderFlag byte = 0x20

// BlobHashesHeaderFlag indicates that this message contains EIP 4844 versioned hashes of the commitments calculated over the blob data for the batch data.
const BlobHashesHeaderFlag byte = L1AuthenticatedMessageHeaderFlag | 0x10 // 0x50

// BrotliMessageHeaderByte indicates that the message is brotli-compressed.
const BrotliMessageHeaderByte byte = 0

// DACertificateMessageHeaderFlag indicates that this message uses a custom data availability system.
// Anytrust uses the legacy TreeDASMessageHeaderFlag instead despite also having a certificate.
const DACertificateMessageHeaderFlag byte = 0x01

// KnownHeaderBits is all header bits with known meaning to this nitro version
const KnownHeaderBits byte = DASMessageHeaderFlag | TreeDASMessageHeaderFlag | L1AuthenticatedMessageHeaderFlag | ZeroheavyMessageHeaderFlag | BlobHashesHeaderFlag | BrotliMessageHeaderByte

var DefaultDASRetentionPeriod time.Duration = time.Hour * 24 * 15

// hasBits returns true if `checking` has all `bits`
func hasBits(checking byte, bits byte) bool {
	return (checking & bits) == bits
}

func IsL1AuthenticatedMessageHeaderByte(header byte) bool {
	return hasBits(header, L1AuthenticatedMessageHeaderFlag)
}

func IsDASMessageHeaderByte(header byte) bool {
	return hasBits(header, DASMessageHeaderFlag)
}

func IsTreeDASMessageHeaderByte(header byte) bool {
	return hasBits(header, TreeDASMessageHeaderFlag)
}

func IsZeroheavyEncodedHeaderByte(header byte) bool {
	return hasBits(header, ZeroheavyMessageHeaderFlag)
}

func IsBlobHashesHeaderByte(header byte) bool {
	return hasBits(header, BlobHashesHeaderFlag)
}

func IsDACertificateMessageHeaderByte(header byte) bool {
	return header == DACertificateMessageHeaderFlag
}

func IsBrotliMessageHeaderByte(b uint8) bool {
	return b == BrotliMessageHeaderByte
}

// IsKnownHeaderByte returns true if the supplied header byte has only known bits
func IsKnownHeaderByte(b uint8) bool {
	return b&^KnownHeaderBits == 0
}

func TriggerDAPayload(ctx context.Context, dapReaders *ReaderRegistry, seqNum uint64, batchBlockHash common.Hash, dataPayload []byte, keysetValidationMode KeysetValidationMode) error {
	payload := dataPayload[40:]
	if len(payload) > 0 && dapReaders != nil {
		if dapReader, found := dapReaders.GetByHeaderByte(payload[0]); found {
			promise := dapReader.RecoverPayload(seqNum, batchBlockHash, dataPayload)
			_, err := promise.Await(ctx)
			if err != nil {
				// Matches the way keyset validation was done inside DAS readers i.e logging the error
				// But other daproviders might just want to return the error
				if strings.Contains(err.Error(), ErrSeqMsgValidation.Error()) && IsDASMessageHeaderByte(payload[0]) {
					if keysetValidationMode == KeysetPanicIfInvalid {
						panic(err.Error())
					} else {
						log.Error(err.Error())
					}
				} else {
					return err
				}
			}
		}
	}

	return nil
}
