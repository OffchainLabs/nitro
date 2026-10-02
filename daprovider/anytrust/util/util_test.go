package util

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/anytrust/tree"
)

type unexpectedReader struct {
	getByHashCalled bool
}

func (r *unexpectedReader) GetByHash(context.Context, common.Hash) ([]byte, error) {
	r.getByHashCalled = true
	return nil, errors.New("GetByHash should not be called for discarded batches")
}

func (r *unexpectedReader) ExpirationPolicy(context.Context) (ExpirationPolicy, error) {
	return KeepForever, nil
}

func (r *unexpectedReader) String() string {
	return "unexpectedReader"
}

type staticKeysetFetcher struct {
	expectedHash common.Hash
	keysetBytes  []byte
	called       bool
}

func (f *staticKeysetFetcher) GetKeysetByHash(_ context.Context, hash common.Hash) ([]byte, error) {
	f.called = true
	if hash != f.expectedHash {
		return nil, errors.New("requested unexpected keyset hash")
	}
	return f.keysetBytes, nil
}

func TestRecoverPayloadFromBatch_DoesNotDropKeysetPreimagesOnDiscardedBatch(t *testing.T) {
	t.Parallel()

	pubKey, privKey, err := blsSignatures.GenerateKeys()
	if err != nil {
		t.Fatalf("GenerateKeys: %v", err)
	}

	keyset := &DataAvailabilityKeyset{
		AssumedHonest: 1,
		PubKeys:       []blsSignatures.PublicKey{pubKey},
	}
	keysetBuf := new(bytes.Buffer)
	if err := keyset.Serialize(keysetBuf); err != nil {
		t.Fatalf("Serialize keyset: %v", err)
	}
	keysetBytes := keysetBuf.Bytes()
	keysetHash := tree.Hash(keysetBytes)

	cert := &DataAvailabilityCertificate{
		KeysetHash:  keysetHash,
		DataHash:    crypto.Keccak256Hash([]byte("test data hash")),
		Timeout:     123456789,
		SignersMask: 1,
		Version:     0,
	}
	// Intentionally sign the wrong message so signature verification fails and the batch is discarded.
	cert.Sig, err = blsSignatures.SignMessage(privKey, []byte("not the signable fields"))
	if err != nil {
		t.Fatalf("SignMessage: %v", err)
	}

	sequencerMsg := make([]byte, 40)
	binary.BigEndian.PutUint64(sequencerMsg[8:16], 0)
	sequencerMsg = append(sequencerMsg, Serialize(cert)...)

	reader := &unexpectedReader{}
	keysetFetcher := &staticKeysetFetcher{
		expectedHash: keysetHash,
		keysetBytes:  keysetBytes,
	}

	preimages := make(daprovider.PreimagesMap)
	payload, gotPreimages, err := RecoverPayloadFromBatch(
		context.Background(),
		42,
		sequencerMsg,
		reader,
		keysetFetcher,
		preimages,
		false,
	)
	if err != nil {
		t.Fatalf("RecoverPayloadFromBatch: %v", err)
	}
	if payload != nil {
		t.Fatalf("expected payload to be nil for a discarded batch")
	}
	if reader.getByHashCalled {
		t.Fatalf("unexpected payload fetch for discarded batch")
	}
	if !keysetFetcher.called {
		t.Fatalf("expected keyset fetch")
	}
	if gotPreimages == nil {
		t.Fatalf("expected preimages map to be returned on discard")
	}
	if len(gotPreimages) == 0 {
		t.Fatalf("expected preimages to be recorded before discard")
	}

	expectedKeysetBinHash := crypto.Keccak256Hash(keysetBytes)
	keysetPreimages := gotPreimages[arbutil.Keccak256PreimageType]
	if keysetPreimages == nil {
		t.Fatalf("expected keccak preimages to be recorded")
	}
	gotKeysetBinPreimage, ok := keysetPreimages[expectedKeysetBinHash]
	if !ok {
		t.Fatalf("expected keyset bin preimage to be recorded")
	}
	if !bytes.Equal(gotKeysetBinPreimage, keysetBytes) {
		t.Fatalf("recorded keyset bin preimage mismatch")
	}
}

func TestRecoverPayloadFromBatch_DoesNotDropKeysetPreimagesOnExpiresTooSoonBatch(t *testing.T) {
	t.Parallel()

	pubKey, privKey, err := blsSignatures.GenerateKeys()
	if err != nil {
		t.Fatalf("GenerateKeys: %v", err)
	}

	keyset := &DataAvailabilityKeyset{
		AssumedHonest: 1,
		PubKeys:       []blsSignatures.PublicKey{pubKey},
	}
	keysetBuf := new(bytes.Buffer)
	if err := keyset.Serialize(keysetBuf); err != nil {
		t.Fatalf("Serialize keyset: %v", err)
	}
	keysetBytes := keysetBuf.Bytes()
	keysetHash := tree.Hash(keysetBytes)

	maxTimestamp := uint64(1234)
	timeout := maxTimestamp + MinLifetimeSecondsForDataAvailabilityCert - 1

	cert := &DataAvailabilityCertificate{
		KeysetHash:  keysetHash,
		DataHash:    crypto.Keccak256Hash([]byte("test data hash")),
		Timeout:     timeout,
		SignersMask: 1,
		Version:     0,
	}
	cert.Sig, err = blsSignatures.SignMessage(privKey, cert.SerializeSignableFields())
	if err != nil {
		t.Fatalf("SignMessage: %v", err)
	}

	sequencerMsg := make([]byte, 40)
	binary.BigEndian.PutUint64(sequencerMsg[8:16], maxTimestamp)
	sequencerMsg = append(sequencerMsg, Serialize(cert)...)

	reader := &unexpectedReader{}
	keysetFetcher := &staticKeysetFetcher{
		expectedHash: keysetHash,
		keysetBytes:  keysetBytes,
	}

	preimages := make(daprovider.PreimagesMap)
	payload, gotPreimages, err := RecoverPayloadFromBatch(
		context.Background(),
		42,
		sequencerMsg,
		reader,
		keysetFetcher,
		preimages,
		false,
	)
	if err != nil {
		t.Fatalf("RecoverPayloadFromBatch: %v", err)
	}
	if payload != nil {
		t.Fatalf("expected payload to be nil for a discarded batch")
	}
	if reader.getByHashCalled {
		t.Fatalf("unexpected payload fetch for discarded batch")
	}
	if !keysetFetcher.called {
		t.Fatalf("expected keyset fetch")
	}
	if gotPreimages == nil {
		t.Fatalf("expected preimages map to be returned on discard")
	}
	if len(gotPreimages) == 0 {
		t.Fatalf("expected preimages to be recorded before discard")
	}

	expectedKeysetBinHash := crypto.Keccak256Hash(keysetBytes)
	keysetPreimages := gotPreimages[arbutil.Keccak256PreimageType]
	if keysetPreimages == nil {
		t.Fatalf("expected keccak preimages to be recorded")
	}
	gotKeysetBinPreimage, ok := keysetPreimages[expectedKeysetBinHash]
	if !ok {
		t.Fatalf("expected keyset bin preimage to be recorded")
	}
	if !bytes.Equal(gotKeysetBinPreimage, keysetBytes) {
		t.Fatalf("recorded keyset bin preimage mismatch")
	}
}
