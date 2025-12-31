// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package util

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	arbosutil "github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/anytrust/tree"
	"github.com/offchainlabs/nitro/util/containers"
)

type Reader interface {
	GetByHash(ctx context.Context, hash common.Hash) ([]byte, error)
	ExpirationPolicy(ctx context.Context) (ExpirationPolicy, error)
	fmt.Stringer
}

type Writer interface {
	// Store requests that the message be stored until timeout (UTC time in unix epoch seconds).
	Store(ctx context.Context, message []byte, timeout uint64) (*DataAvailabilityCertificate, error)
	fmt.Stringer
}

type KeysetFetcher interface {
	GetKeysetByHash(context.Context, common.Hash) ([]byte, error)
}

// NewReader is generally meant to be only used by nitro.
// DA Providers should implement methods in the Reader interface independently
func NewReader(anyTrustReader Reader, keysetFetcher KeysetFetcher, validationMode daprovider.KeysetValidationMode) *reader {
	return &reader{
		anyTrustReader: anyTrustReader,
		keysetFetcher:  keysetFetcher,
		validationMode: validationMode,
	}
}

type reader struct {
	anyTrustReader Reader
	keysetFetcher  KeysetFetcher
	validationMode daprovider.KeysetValidationMode
}

// recoverInternal is the shared implementation for both RecoverPayload and CollectPreimages
func (d *reader) recoverInternal(
	ctx context.Context,
	batchNum uint64,
	sequencerMsg []byte,
	needPayload bool,
	needPreimages bool,
) ([]byte, daprovider.PreimagesMap, error) {
	// Convert validation mode to boolean for the internal function
	validateSeqMsg := d.validationMode != daprovider.KeysetDontValidate
	return recoverPayloadFromBatchInternal(ctx, batchNum, sequencerMsg, d.anyTrustReader, d.keysetFetcher, validateSeqMsg, needPayload, needPreimages)
}

// RecoverPayload fetches the underlying payload from the DA provider
func (d *reader) RecoverPayload(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PayloadResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PayloadResult, error) {
		payload, _, err := d.recoverInternal(ctx, batchNum, sequencerMsg, true, false)
		return daprovider.PayloadResult{Payload: payload}, err
	})
}

// CollectPreimages collects preimages from the DA provider
func (d *reader) CollectPreimages(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PreimagesResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PreimagesResult, error) {
		_, preimages, err := d.recoverInternal(ctx, batchNum, sequencerMsg, false, true)
		return daprovider.PreimagesResult{Preimages: preimages}, err
	})
}

// RecoverPayloadAndPreimages fetches the underlying payload and collects preimages from the DA provider given the batch header information
func (d *reader) RecoverPayloadAndPreimages(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PayloadAndPreimagesResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PayloadAndPreimagesResult, error) {
		payload, preimages, err := d.recoverInternal(ctx, batchNum, sequencerMsg, true, true)
		return daprovider.PayloadAndPreimagesResult{
			Payload:   payload,
			Preimages: preimages,
		}, err
	})
}

// NewWriter is generally meant to be only used by nitro.
// DA Providers should implement methods in the DAProviderWriter interface independently
func NewWriter(anyTrustWriter Writer, maxMessageSize int) *writer {
	return &writer{
		anyTrustWriter: anyTrustWriter,
		maxMessageSize: maxMessageSize,
	}
}

type writer struct {
	anyTrustWriter Writer
	maxMessageSize int
}

func (d *writer) Store(message []byte, timeout uint64) containers.PromiseInterface[[]byte] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) ([]byte, error) {
		cert, err := d.anyTrustWriter.Store(ctx, message, timeout)
		if err != nil {
			// If the aggregator failed due to insufficient backends, signal explicit fallback
			// to allow batch poster to try the next DA provider in the chain. The Aggregator
			// has a loud ERROR if the threshold of failing committee members is approaching,
			// which should give time to correct any errors to avoid fallback. Otherwise
			// the operator can set disable-dap-fallback-store-data-on-chain to totally
			// disable automatic fallback to EthDA.
			if errors.Is(err, ErrBatchFailed) {
				return nil, fmt.Errorf("%w: %w", daprovider.ErrFallbackRequested, err)
			}
			return nil, err
		}
		return Serialize(cert), nil
	})
}

func (d *writer) GetMaxMessageSize() containers.PromiseInterface[int] {
	return containers.NewReadyPromise(d.maxMessageSize, nil)
}

var (
	ErrHashMismatch = errors.New("result does not match expected hash")
	// ErrBatchFailed keeps "DAS" for backward compatibility with external systems that may match on this string
	ErrBatchFailed = errors.New("unable to batch to DAS")
)

const MinLifetimeSecondsForDataAvailabilityCert = 7 * 24 * 60 * 60 // one week

// RecoverPayloadFromBatch is deprecated, use recoverPayloadFromBatchInternal
func RecoverPayloadFromBatch(
	ctx context.Context,
	batchNum uint64,
	sequencerMsg []byte,
	anyTrustReader Reader,
	keysetFetcher KeysetFetcher,
	preimages daprovider.PreimagesMap,
	validateSeqMsg bool,
) ([]byte, daprovider.PreimagesMap, error) {
	needPreimages := preimages != nil
	payload, recoveredPreimages, err := recoverPayloadFromBatchInternal(ctx, batchNum, sequencerMsg, anyTrustReader, keysetFetcher, validateSeqMsg, true, needPreimages)
	if err != nil {
		return nil, nil, err
	}
	// If preimages were passed in, copy recovered preimages into the provided map
	if preimages != nil && recoveredPreimages != nil {
		for piType, piMap := range recoveredPreimages {
			if preimages[piType] == nil {
				preimages[piType] = make(map[common.Hash][]byte)
			}
			for hash, preimage := range piMap {
				preimages[piType][hash] = preimage
			}
		}
		return payload, preimages, nil
	}
	return payload, recoveredPreimages, nil
}

// recoverPayloadFromBatchInternal is the shared implementation
func recoverPayloadFromBatchInternal(
	ctx context.Context,
	batchNum uint64,
	sequencerMsg []byte,
	anyTrustReader Reader,
	keysetFetcher KeysetFetcher,
	validateSeqMsg bool,
	needPayload bool,
	needPreimages bool,
) ([]byte, daprovider.PreimagesMap, error) {
	var preimages daprovider.PreimagesMap
	var preimageRecorder daprovider.PreimageRecorder
	if needPreimages {
		preimages = make(daprovider.PreimagesMap)
		preimageRecorder = daprovider.RecordPreimagesTo(preimages)
	}
	cert, err := DeserializeCertFrom(bytes.NewReader(sequencerMsg[40:]))
	if err != nil {
		log.Error("Failed to deserialize AnyTrust message", "err", err)
		return nil, nil, nil
	}
	version := cert.Version

	if version >= 2 {
		log.Error("Your node software is probably out of date", "certificateVersion", version)
		return nil, nil, nil
	}

	getByHash := func(ctx context.Context, hash common.Hash) ([]byte, error) {
		newHash := hash
		if version == 0 {
			newHash = tree.FlatHashToTreeHash(hash)
		}

		preimage, err := anyTrustReader.GetByHash(ctx, newHash)
		if err != nil && hash != newHash {
			log.Debug("error fetching new style hash, trying old", "new", newHash, "old", hash, "err", err)
			preimage, err = anyTrustReader.GetByHash(ctx, hash)
		}
		if err != nil {
			return nil, err
		}

		switch {
		case version == 0 && crypto.Keccak256Hash(preimage) != hash:
			fallthrough
		case version == 1 && tree.Hash(preimage) != hash:
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
		return nil, nil, err
	}
	if preimageRecorder != nil {
		tree.RecordHash(preimageRecorder, keysetPreimage)
	}

	keyset, err := DeserializeKeyset(bytes.NewReader(keysetPreimage), !validateSeqMsg)
	if err != nil {
		return nil, nil, fmt.Errorf("%w. Couldn't deserialize keyset, err: %w, keyset hash: %x batch num: %d", daprovider.ErrSeqMsgValidation, err, cert.KeysetHash, batchNum)
	}
	err = keyset.VerifySignature(cert.SignersMask, cert.SerializeSignableFields(), cert.Sig)
	if err != nil {
		log.Error("Bad signature on AnyTrust batch", "err", err)
		return nil, nil, nil
	}

	maxTimestamp := binary.BigEndian.Uint64(sequencerMsg[8:16])
	if cert.Timeout < maxTimestamp+MinLifetimeSecondsForDataAvailabilityCert {
		log.Error("Data availability cert expires too soon", "err", "")
		return nil, nil, nil
	}

	dataHash := cert.DataHash
	var payload []byte
	// We need to fetch the payload if either we need to return it or need to record preimages
	if needPayload || needPreimages {
		payload, err = getByHash(ctx, dataHash)
		if err != nil {
			log.Error("Couldn't fetch AnyTrust batch contents", "err", err)
			return nil, nil, err
		}
	}

	if preimageRecorder != nil && payload != nil {
		if version == 0 {
			treeLeaf := tree.FlatHashToTreeLeaf(dataHash)
			preimageRecorder(dataHash, payload, arbutil.Keccak256PreimageType)
			preimageRecorder(crypto.Keccak256Hash(treeLeaf), treeLeaf, arbutil.Keccak256PreimageType)
		} else {
			tree.RecordHash(preimageRecorder, payload)
		}
	}

	return payload, preimages, nil
}

type DataAvailabilityCertificate struct {
	KeysetHash  [32]byte
	DataHash    [32]byte
	Timeout     uint64
	SignersMask uint64
	Sig         blsSignatures.Signature
	Version     uint8
}

func DeserializeCertFrom(rd io.Reader) (c *DataAvailabilityCertificate, err error) {
	r := bufio.NewReader(rd)
	c = &DataAvailabilityCertificate{}

	header, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if !daprovider.IsAnyTrustMessageHeaderByte(header) {
		return nil, errors.New("tried to deserialize a message that doesn't have the AnyTrust header")
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

	if daprovider.IsAnyTrustTreeMessageHeaderByte(header) {
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
	da Reader,
	assumeKeysetValid bool,
) (*DataAvailabilityKeyset, error) {
	keysetBytes, err := da.GetByHash(ctx, c.KeysetHash)
	if err != nil {
		return nil, err
	}
	if !tree.ValidHash(c.KeysetHash, keysetBytes) {
		return nil, errors.New("keyset hash does not match cert")
	}
	return DeserializeKeyset(bytes.NewReader(keysetBytes), assumeKeysetValid)
}

type DataAvailabilityKeyset struct {
	AssumedHonest uint64
	PubKeys       []blsSignatures.PublicKey
}

func (keyset *DataAvailabilityKeyset) Serialize(wr io.Writer) error {
	if err := arbosutil.Uint64ToWriter(keyset.AssumedHonest, wr); err != nil {
		return err
	}
	if err := arbosutil.Uint64ToWriter(uint64(len(keyset.PubKeys)), wr); err != nil {
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
	if wr.Len() > tree.BinSize {
		return common.Hash{}, errors.New("keyset too large")
	}
	return tree.Hash(wr.Bytes()), nil
}

func DeserializeKeyset(rd io.Reader, assumeKeysetValid bool) (*DataAvailabilityKeyset, error) {
	assumedHonest, err := arbosutil.Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}
	numKeys, err := arbosutil.Uint64FromReader(rd)
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

	flags := daprovider.AnyTrustMessageHeaderFlag
	if c.Version != 0 {
		flags |= daprovider.AnyTrustTreeMessageHeaderFlag
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
