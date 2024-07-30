// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package daprovider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/das/dastree"
)

type DASReader interface {
	GetByHash(ctx context.Context, hash common.Hash) ([]byte, error)
	ExpirationPolicy(ctx context.Context) (ExpirationPolicy, error)
}

type DASWriter interface {
	// Store requests that the message be stored until timeout (UTC time in unix epoch seconds).
	Store(ctx context.Context, message []byte, timeout uint64) (*DataAvailabilityCertificate, error)
	fmt.Stringer
}

var DefaultDASRetentionPeriod time.Duration = time.Hour * 24 * 15

type DASKeysetFetcher interface {
	GetKeysetByHash(context.Context, common.Hash) ([]byte, error)
}

type BlobReader interface {
	GetBlobs(
		ctx context.Context,
		batchBlockHash common.Hash,
		versionedHashes []common.Hash,
	) ([]kzg4844.Blob, error)
	Initialize(ctx context.Context) error
}

// PreimageRecorder is used to add (key,value) pair to the map accessed by key = ty of a bigger map, preimages.
// If ty doesn't exist as a key in the preimages map, then it is intialized to map[common.Hash][]byte and then (key,value) pair is added
type PreimageRecorder func(key common.Hash, value []byte, ty arbutil.PreimageType)

// RecordPreimagesTo takes in preimages map and returns a function that can be used
// In recording (hash,preimage) key value pairs into preimages map, when fetching payload through RecoverPayloadFromBatch
func RecordPreimagesTo(preimages map[arbutil.PreimageType]map[common.Hash][]byte) PreimageRecorder {
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

// BlobHashesHeaderFlag indicates that this message contains EIP 4844 versioned hashes of the committments calculated over the blob data for the batch data.
const BlobHashesHeaderFlag byte = L1AuthenticatedMessageHeaderFlag | 0x10 // 0x50

// BrotliMessageHeaderByte indicates that the message is brotli-compressed.
const BrotliMessageHeaderByte byte = 0

// KnownHeaderBits is all header bits with known meaning to this nitro version
const KnownHeaderBits byte = DASMessageHeaderFlag | TreeDASMessageHeaderFlag | L1AuthenticatedMessageHeaderFlag | ZeroheavyMessageHeaderFlag | BlobHashesHeaderFlag | BrotliMessageHeaderByte

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

func IsBrotliMessageHeaderByte(b uint8) bool {
	return b == BrotliMessageHeaderByte
}

// IsKnownHeaderByte returns true if the supplied header byte has only known bits
func IsKnownHeaderByte(b uint8) bool {
	return b&^KnownHeaderBits == 0
}

const MinLifetimeSecondsForDataAvailabilityCert = 7 * 24 * 60 * 60 // one week
var (
	ErrHashMismatch          = errors.New("result does not match expected hash")
	ErrBatchToDasFailed      = errors.New("unable to batch to DAS")
	ErrNoBlobReader          = errors.New("blob batch payload was encountered but no BlobReader was configured")
	ErrInvalidBlobDataFormat = errors.New("blob batch data is not a list of hashes as expected")
	ErrSeqMsgValidation      = errors.New("error validating recovered payload from batch")
)

type KeysetValidationMode uint8

const KeysetValidate KeysetValidationMode = 0
const KeysetPanicIfInvalid KeysetValidationMode = 1
const KeysetDontValidate KeysetValidationMode = 2

func RecoverPayloadFromDasBatch(
	ctx context.Context,
	batchNum uint64,
	sequencerMsg []byte,
	dasReader DASReader,
	keysetFetcher DASKeysetFetcher,
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

		preimage, err := dasReader.GetByHash(ctx, newHash)
		if err != nil && hash != newHash {
			log.Debug("error fetching new style hash, trying old", "new", newHash, "old", hash, "err", err)
			preimage, err = dasReader.GetByHash(ctx, hash)
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

	keysetPreimage, err := keysetFetcher.GetKeysetByHash(ctx, cert.KeysetHash)
	if err != nil {
		log.Error("Couldn't get keyset", "err", err, "keysetHash", common.Bytes2Hex(cert.KeysetHash[:]))
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
	da DASReader,
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

func Serialize(c *DataAvailabilityCertificate) []byte {

	flags := DASMessageHeaderFlag
	if c.Version != 0 {
		flags |= TreeDASMessageHeaderFlag
	}

	buf := make([]byte, 0)
	buf = append(buf, flags)
	buf = append(buf, c.KeysetHash[:]...)
	buf = append(buf, c.SerializeSignableFields()...)

	var intData [8]byte
	binary.BigEndian.PutUint64(intData[:], c.SignersMask)
	buf = append(buf, intData[:]...)

	return append(buf, blsSignatures.SignatureToBytes(c.Sig)...)
}
