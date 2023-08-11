// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbstate

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/das/celestia"
	"github.com/offchainlabs/nitro/das/dastree"
)

type DataAvailabilityReader interface {
	GetByHash(ctx context.Context, hash common.Hash) ([]byte, error)
	ExpirationPolicy(ctx context.Context) (ExpirationPolicy, error)
}

type CelestiaDataAvailabilityReader interface {
	celestia.DataAvailabilityReader
}

var ErrHashMismatch = errors.New("result does not match expected hash")

// DASMessageHeaderFlag indicates that this data is a certificate for the data availability service,
// which will retrieve the full batch data.
const DASMessageHeaderFlag byte = 0x80

// CelestiaMessageHeaderFlag indicates that this data is a Blob Pointer
// which will be used to retrieve data from Celestia
const CelestiaMessageHeaderFlag byte = 0x0c

// TreeDASMessageHeaderFlag indicates that this DAS certificate data employs the new merkelization strategy.
// Ignored when DASMessageHeaderFlag is not set.
const TreeDASMessageHeaderFlag byte = 0x08

// L1AuthenticatedMessageHeaderFlag indicates that this message was authenticated by L1. Currently unused.
const L1AuthenticatedMessageHeaderFlag byte = 0x40

// ZeroheavyMessageHeaderFlag indicates that this message is zeroheavy-encoded.
const ZeroheavyMessageHeaderFlag byte = 0x20

// BlobHashesHeaderFlag indicates that this message contains EIP 4844 versioned hashes of the committments calculated over the blob data for the batch data.
const BlobHashesHeaderFlag byte = L1AuthenticatedMessageHeaderFlag | 0x10 // 0x50

// BrotliMessageHeaderByte indicates that the message is brotli-compressed.
const BrotliMessageHeaderByte byte = 0

// KnownHeaderBits is all header bits with known meaning to this nitro version
const KnownHeaderBits byte = DASMessageHeaderFlag | CelestiaMessageHeaderFlag | TreeDASMessageHeaderFlag | L1AuthenticatedMessageHeaderFlag | ZeroheavyMessageHeaderFlag | BlobHashesHeaderFlag | BrotliMessageHeaderByte

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

func IsCelestiaMessageHeaderByte(b uint8) bool {
	return hasBits(header, CelestiaMessageHeaderFlag)
}

func IsBrotliMessageHeaderByte(b uint8) bool {
	return b == BrotliMessageHeaderByte
}

// IsKnownHeaderByte returns true if the supplied header byte has only known bits
func IsKnownHeaderByte(b uint8) bool {
	return b&^KnownHeaderBits == 0
}

type DataAvailabilityCertificate struct {
	KeysetHash  [32]byte
	DataHash    [32]byte
	Timeout     uint64
	SignersMask uint64
	Sig         blsSignatures.Signature
	Version     uint8
}

func DeserializeDASCertFrom(rd io.Reader) (c *DataAvailabilityCertificate, err error) {
	r := bufio.NewReader(rd)
	c = &DataAvailabilityCertificate{}

	header, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if !IsDASMessageHeaderByte(header) {
		return nil, errors.New("tried to deserialize a message that doesn't have the DAS header")
	}

	_, err = io.ReadFull(r, c.KeysetHash[:])
	if err != nil {
		return nil, err
	}

	_, err = io.ReadFull(r, c.DataHash[:])
	if err != nil {
		return nil, err
	}

	var timeoutBuf [8]byte
	_, err = io.ReadFull(r, timeoutBuf[:])
	if err != nil {
		return nil, err
	}
	c.Timeout = binary.BigEndian.Uint64(timeoutBuf[:])

	if IsTreeDASMessageHeaderByte(header) {
		var versionBuf [1]byte
		_, err = io.ReadFull(r, versionBuf[:])
		if err != nil {
			return nil, err
		}
		c.Version = versionBuf[0]
	}

	var signersMaskBuf [8]byte
	_, err = io.ReadFull(r, signersMaskBuf[:])
	if err != nil {
		return nil, err
	}
	c.SignersMask = binary.BigEndian.Uint64(signersMaskBuf[:])

	var blsSignaturesBuf [96]byte
	_, err = io.ReadFull(r, blsSignaturesBuf[:])
	if err != nil {
		return nil, err
	}
	c.Sig, err = blsSignatures.SignatureFromBytes(blsSignaturesBuf[:])
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *DataAvailabilityCertificate) SerializeSignableFields() []byte {
	buf := make([]byte, 0, 32+9)
	buf = append(buf, c.DataHash[:]...)

	var intData [8]byte
	binary.BigEndian.PutUint64(intData[:], c.Timeout)
	buf = append(buf, intData[:]...)

	if c.Version != 0 {
		buf = append(buf, c.Version)
	}

	return buf
}

func (c *DataAvailabilityCertificate) RecoverKeyset(
	ctx context.Context,
	da DataAvailabilityReader,
	assumeKeysetValid bool,
) (*DataAvailabilityKeyset, error) {
	keysetBytes, err := da.GetByHash(ctx, c.KeysetHash)
	if err != nil {
		return nil, err
	}
	if !dastree.ValidHash(c.KeysetHash, keysetBytes) {
		return nil, errors.New("keyset hash does not match cert")
	}
	return DeserializeKeyset(bytes.NewReader(keysetBytes), assumeKeysetValid)
}

type DataAvailabilityKeyset struct {
	AssumedHonest uint64
	PubKeys       []blsSignatures.PublicKey
}

func (keyset *DataAvailabilityKeyset) Serialize(wr io.Writer) error {
	if err := util.Uint64ToWriter(keyset.AssumedHonest, wr); err != nil {
		return err
	}
	if err := util.Uint64ToWriter(uint64(len(keyset.PubKeys)), wr); err != nil {
		return err
	}
	for _, pk := range keyset.PubKeys {
		pkBuf := blsSignatures.PublicKeyToBytes(pk)
		buf := []byte{byte(len(pkBuf) / 256), byte(len(pkBuf) % 256)}
		_, err := wr.Write(append(buf, pkBuf...))
		if err != nil {
			return err
		}
	}
	return nil
}

func (keyset *DataAvailabilityKeyset) Hash() (common.Hash, error) {
	wr := bytes.NewBuffer([]byte{})
	if err := keyset.Serialize(wr); err != nil {
		return common.Hash{}, err
	}
	if wr.Len() > dastree.BinSize {
		return common.Hash{}, errors.New("keyset too large")
	}
	return dastree.Hash(wr.Bytes()), nil
}

func DeserializeKeyset(rd io.Reader, assumeKeysetValid bool) (*DataAvailabilityKeyset, error) {
	assumedHonest, err := util.Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}
	numKeys, err := util.Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}
	if numKeys > 64 {
		return nil, errors.New("too many keys in serialized DataAvailabilityKeyset")
	}
	pubkeys := make([]blsSignatures.PublicKey, numKeys)
	buf2 := []byte{0, 0}
	for i := uint64(0); i < numKeys; i++ {
		if _, err := io.ReadFull(rd, buf2); err != nil {
			return nil, err
		}
		buf := make([]byte, int(buf2[0])*256+int(buf2[1]))
		if _, err := io.ReadFull(rd, buf); err != nil {
			return nil, err
		}
		pubkeys[i], err = blsSignatures.PublicKeyFromBytes(buf, assumeKeysetValid)
		if err != nil {
			return nil, err
		}
	}
	return &DataAvailabilityKeyset{
		AssumedHonest: assumedHonest,
		PubKeys:       pubkeys,
	}, nil
}

func (keyset *DataAvailabilityKeyset) VerifySignature(signersMask uint64, data []byte, sig blsSignatures.Signature) error {
	pubkeys := []blsSignatures.PublicKey{}
	numNonSigners := uint64(0)
	for i := 0; i < len(keyset.PubKeys); i++ {
		if (1<<i)&signersMask != 0 {
			pubkeys = append(pubkeys, keyset.PubKeys[i])
		} else {
			numNonSigners++
		}
	}
	if numNonSigners >= keyset.AssumedHonest {
		return errors.New("not enough signers")
	}
	aggregatedPubKey := blsSignatures.AggregatePublicKeys(pubkeys)
	success, err := blsSignatures.VerifySignature(sig, data, aggregatedPubKey)

	if err != nil {
		return err
	}
	if !success {
		return errors.New("bad signature")
	}
	return nil
}

type ExpirationPolicy int64

const (
	KeepForever                ExpirationPolicy = iota // Data is kept forever
	DiscardAfterArchiveTimeout                         // Data is kept till Archive timeout (Archive Timeout is defined by archiving node, assumed to be as long as minimum data timeout)
	DiscardAfterDataTimeout                            // Data is kept till aggregator provided timeout (Aggregator provides a timeout for data while making the put call)
	MixedTimeout                                       // Used for cases with mixed type of timeout policy(Mainly used for aggregators which have data availability services with multiply type of timeout policy)
	DiscardImmediately                                 // Data is never stored (Mainly used for empty/wrapper/placeholder classes)
	// Add more type of expiration policy.
)

func (ep ExpirationPolicy) String() (string, error) {
	switch ep {
	case KeepForever:
		return "KeepForever", nil
	case DiscardAfterArchiveTimeout:
		return "DiscardAfterArchiveTimeout", nil
	case DiscardAfterDataTimeout:
		return "DiscardAfterDataTimeout", nil
	case MixedTimeout:
		return "MixedTimeout", nil
	case DiscardImmediately:
		return "DiscardImmediately", nil
	default:
		return "", errors.New("unknown Expiration Policy")
	}
}

func StringToExpirationPolicy(s string) (ExpirationPolicy, error) {
	switch s {
	case "KeepForever":
		return KeepForever, nil
	case "DiscardAfterArchiveTimeout":
		return DiscardAfterArchiveTimeout, nil
	case "DiscardAfterDataTimeout":
		return DiscardAfterDataTimeout, nil
	case "MixedTimeout":
		return MixedTimeout, nil
	case "DiscardImmediately":
		return DiscardImmediately, nil
	default:
		return -1, fmt.Errorf("invalid Expiration Policy: %s", s)
	}
}
