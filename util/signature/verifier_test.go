// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package signature

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/util/contracts"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestVerifier(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	signingAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := DataSignerFromPrivateKey(privateKey)

	config := TestingFeedVerifierConfig
	config.AllowedAddresses = []string{signingAddr.Hex()}
	verifier, err := NewVerifier(&config, nil)
	Require(t, err)

	data := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	hash := crypto.Keccak256Hash(data)

	signature, err := dataSigner(hash.Bytes())
	Require(t, err, "error signing data")

	err = verifier.VerifyData(ctx, signature, data)
	Require(t, err, "error verifying data")

	err = verifier.VerifyHash(ctx, signature, hash)
	Require(t, err, "error verifying data")

	badData := []byte{1, 1, 2, 3, 4, 5, 6, 7}
	err = verifier.VerifyData(ctx, signature, badData)
	if !errors.Is(err, ErrSignatureNotVerified) {
		t.Error("unexpected error", err)
	}
}

func TestMissingRequiredSignature(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := TestingFeedVerifierConfig
	config.Dangerous.AcceptMissing = false
	verifier, err := NewVerifier(&config, nil)
	Require(t, err)
	err = verifier.VerifyData(ctx, nil, nil)
	if !errors.Is(err, ErrMissingSignature) {
		t.Error("didn't fail when missing feed signature")
	}
}

func TestMissingSignatureAllowed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := TestingFeedVerifierConfig
	config.Dangerous.AcceptMissing = true
	verifier, err := NewVerifier(&config, nil)
	Require(t, err)
	err = verifier.VerifyData(ctx, nil, nil)
	Require(t, err, "error verifying data")
}

func TestVerifierBatchPoster(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	signingAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := DataSignerFromPrivateKey(privateKey)

	bpVerifier := contracts.NewMockBatchPosterVerifier(signingAddr)
	config := TestingFeedVerifierConfig
	config.AcceptSequencer = true
	verifier, err := NewVerifier(&config, bpVerifier)
	Require(t, err)

	data := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	hash := crypto.Keccak256Hash(data)

	signature, err := dataSigner(hash.Bytes())
	Require(t, err, "error signing data")

	err = verifier.VerifyData(ctx, signature, data)
	Require(t, err, "error verifying data")

	err = verifier.VerifyHash(ctx, signature, hash)
	Require(t, err, "error verifying data")

	badKey, err := crypto.GenerateKey()
	Require(t, err)
	badDataSigner := DataSignerFromPrivateKey(badKey)
	badSignature, err := badDataSigner(hash.Bytes())
	Require(t, err, "error signing data")

	err = verifier.VerifyData(ctx, badSignature, data)
	if !errors.Is(err, ErrSignerNotApproved) {
		t.Error("unexpected error", err)
	}
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}
