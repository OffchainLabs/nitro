// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/cmd/daprovider/das/dasutil"
	"github.com/offchainlabs/nitro/util/signature"
)

func checkSig(keyDir string, message []byte, timeout uint64, sig []byte) (*dasutil.DataAvailabilityCertificate, error) {
	pubkeyEncoded, err := ioutil.ReadFile(keyDir + "/ecdsa.pub")
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

func TestExtraSignatureCheck(t *testing.T) {
	keyDir := t.TempDir()
	err := GenerateAndStoreECDSAKeys(keyDir)
	Require(t, err)

	privateKey, err := crypto.LoadECDSA(keyDir + "/ecdsa")
	Require(t, err)
	signer := signature.DataSignerFromPrivateKey(privateKey)

	msg := []byte("Hello world")
	timeout := uint64(1234)
	sig, err := applyDasSigner(signer, msg, timeout)
	Require(t, err)
	_, err = checkSig(keyDir, msg, timeout, sig)
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
