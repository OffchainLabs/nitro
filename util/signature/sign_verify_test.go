package signature

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestSignVerifyModes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	signingAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := DataSignerFromPrivateKey(privateKey)

	config := TestSignVerifyConfig
	config.SymmetricFallback = false
	config.SymmetricSign = false
	config.ECDSA.AcceptSequencer = false
	config.ECDSA.AllowedAddresses = []string{signingAddr.Hex()}
	signVerifyECDSA, err := NewSignVerify(&config, dataSigner, nil)
	Require(t, err)

	configSymmetric := TestSignVerifyConfig
	configSymmetric.SymmetricFallback = true
	configSymmetric.SymmetricSign = true
	configSymmetric.ECDSA.AcceptSequencer = false
	signVerifySymmetric, err := NewSignVerify(&configSymmetric, nil, nil)
	Require(t, err)

	configFallback := TestSignVerifyConfig
	configFallback.SymmetricFallback = true
	configFallback.SymmetricSign = false
	configFallback.ECDSA.AllowedAddresses = []string{signingAddr.Hex()}
	configFallback.ECDSA.AcceptSequencer = false
	signVerifyFallback, err := NewSignVerify(&configFallback, dataSigner, nil)
	Require(t, err)

	data := []byte{0, 1, 2, 3, 4, 5, 6, 7}

	ecdsaSig, err := signVerifyECDSA.SignMessage(data)
	Require(t, err, "error signing data")

	err = signVerifyECDSA.VerifySignature(ctx, ecdsaSig, data)
	Require(t, err, "error verifying data")

	err = signVerifyFallback.VerifySignature(ctx, ecdsaSig, data)
	Require(t, err, "error verifying data")

	err = signVerifySymmetric.VerifySignature(ctx, ecdsaSig, data)
	if !errors.Is(err, ErrSignatureNotVerified) {
		t.Error("unexpected error", err)
	}

	symSig, err := signVerifySymmetric.SignMessage(data)
	Require(t, err, "error signing data")

	err = signVerifySymmetric.VerifySignature(ctx, symSig, data)
	Require(t, err, "error verifying data")

	err = signVerifyFallback.VerifySignature(ctx, symSig, data)
	Require(t, err, "error verifying data")

	err = signVerifyECDSA.VerifySignature(ctx, symSig, data)
	if !errors.Is(err, ErrSignatureNotVerified) {
		t.Error("unexpected error", err)
	}

	fallbackSig, err := signVerifyFallback.SignMessage(data)
	Require(t, err, "error signing data")

	err = signVerifyECDSA.VerifySignature(ctx, fallbackSig, data)
	Require(t, err, "error verifying data")
}
