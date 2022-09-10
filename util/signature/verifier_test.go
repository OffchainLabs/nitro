// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package signature

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/util/contracts"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestVerifier(t *testing.T) {
	ctx := context.Background()
	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	signingAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := DataSignerFromPrivateKey(privateKey)

	authorizedAddresses := make([]common.Address, 0)
	authorizedAddresses = append(authorizedAddresses, signingAddr)
	verifier := NewVerifier(true, authorizedAddresses, nil)

	data := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	hash := crypto.Keccak256Hash(data)

	signature, err := dataSigner(hash.Bytes())
	Require(t, err, "error signing data")

	verified, err := verifier.VerifyData(ctx, signature, data)
	Require(t, err, "error verifying data")
	if !verified {
		t.Error("signature not verified")
	}

	verified, err = verifier.VerifyHash(ctx, signature, hash)
	Require(t, err, "error verifying data")
	if !verified {
		t.Error("signature not verified")
	}

	badData := []byte{1, 1, 2, 3, 4, 5, 6, 7}
	verified, err = verifier.VerifyData(ctx, signature, badData)
	Require(t, err, "error verifying data")
	if verified {
		t.Error("signature unexpectedly verified")
	}
}

func TestMissingRequiredSignature(t *testing.T) {
	ctx := context.Background()
	verifier := NewVerifier(true, nil, nil)
	_, err := verifier.VerifyData(ctx, nil, nil)
	if !errors.Is(err, ErrMissingFeedSignature) {
		t.Error("didn't fail when missing feed signature")
	}
}

func TestMissingSignatureAllowed(t *testing.T) {
	ctx := context.Background()
	verifier := NewVerifier(false, nil, nil)
	verified, err := verifier.VerifyData(ctx, nil, nil)
	Require(t, err, "error verifying data")
	if !verified {
		t.Error("signature not verified")
	}
}

func TestVerifierBatchPoster(t *testing.T) {
	ctx := context.Background()
	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	signingAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := DataSignerFromPrivateKey(privateKey)

	bpVerifier := contracts.NewMockBatchPosterVerifier(signingAddr)
	verifier := NewVerifier(true, nil, bpVerifier)

	data := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	hash := crypto.Keccak256Hash(data)

	signature, err := dataSigner(hash.Bytes())
	Require(t, err, "error signing data")

	verified, err := verifier.VerifyData(ctx, signature, data)
	Require(t, err, "error verifying data")
	if !verified {
		t.Error("signature not verified")
	}

	verified, err = verifier.VerifyHash(ctx, signature, hash)
	Require(t, err, "error verifying data")
	if !verified {
		t.Error("signature not verified")
	}

	badData := []byte{1, 1, 2, 3, 4, 5, 6, 7}
	verified, err = verifier.VerifyData(ctx, signature, badData)
	Require(t, err, "error verifying data")
	if verified {
		t.Error("signature unexpectedly verified")
	}
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}
