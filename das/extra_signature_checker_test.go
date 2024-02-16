// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/util/signature"
)

type StubSignatureCheckDAS struct {
	keyDir string
}

func (s *StubSignatureCheckDAS) Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (*arbstate.DataAvailabilityCertificate, error) {
	pubkeyEncoded, err := ioutil.ReadFile(s.keyDir + "/ecdsa.pub")
	if err != nil {
		return nil, err
	}
	pubkey, err := hex.DecodeString(string(pubkeyEncoded))
	if err != nil {
		return nil, err
	}

	verified := crypto.VerifySignature(pubkey, dasStoreHash(message, timeout), sig[:64])
	if !verified {
		return nil, errors.New("signature verification failed")
	}
	return nil, nil
}

func (s *StubSignatureCheckDAS) ExpirationPolicy(ctx context.Context) (arbstate.ExpirationPolicy, error) {
	return arbstate.KeepForever, nil
}

func (s *StubSignatureCheckDAS) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	return []byte{}, nil
}

func (s *StubSignatureCheckDAS) HealthCheck(ctx context.Context) error {
	return nil
}

func (s *StubSignatureCheckDAS) String() string {
	return "StubSignatureCheckDAS"
}

func TestExtraSignatureCheck(t *testing.T) {
	keyDir := t.TempDir()
	err := GenerateAndStoreECDSAKeys(keyDir)
	Require(t, err)

	privateKey, err := crypto.LoadECDSA(keyDir + "/ecdsa")
	Require(t, err)
	signer := signature.DataSignerFromPrivateKey(privateKey)

	var da DataAvailabilityServiceWriter = &StubSignatureCheckDAS{keyDir}
	da, err = NewStoreSigningDAS(da, signer)
	Require(t, err)

	_, err = da.Store(context.Background(), []byte("Hello world"), 1234, []byte{})
	Require(t, err)
}

func TestSimpleSignatureCheck(t *testing.T) {
	keyDir := t.TempDir()
	err := GenerateAndStoreECDSAKeys(keyDir)
	Require(t, err)
	privateKey, err := crypto.LoadECDSA(keyDir + "/ecdsa")
	Require(t, err)

	data := []byte("Hello World")
	dataHash := crypto.Keccak256(data)
	sig, err := crypto.Sign(dataHash, privateKey)
	Require(t, err)

	pubkeyEncoded, err := ioutil.ReadFile(keyDir + "/ecdsa.pub")
	Require(t, err)

	pubkey, err := hex.DecodeString(string(pubkeyEncoded))
	Require(t, err)

	verified := crypto.VerifySignature(pubkey, dataHash, sig[:64])
	if !verified {
		Fail(t, "Signature not verified")
	}
}

func TestEvenSimplerSignatureCheck(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	Require(t, err)

	data := []byte("Hello World")
	dataHash := crypto.Keccak256(data)
	sig, err := crypto.Sign(dataHash, privateKey)
	Require(t, err)

	pubkey, err := crypto.SigToPub(dataHash, sig)
	Require(t, err)
	if !bytes.Equal(crypto.FromECDSAPub(pubkey), crypto.FromECDSAPub(&privateKey.PublicKey)) {
		Fail(t, "Derived pubkey doesn't match pubkey")
	}

	verified := crypto.VerifySignature(crypto.FromECDSAPub(&privateKey.PublicKey), dataHash, sig[:64])
	if !verified {
		Fail(t, "Signature not verified")
	}
}
