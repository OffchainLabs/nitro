package signature

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestSignVerifyModes(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	signingAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := DataSignerFromPrivateKey(privateKey)

	config := DefaultSignVerifyConfig
	config.SymmetricFallback = false
	config.SymmetricSign = false
	config.ECDSA.AcceptBatchPosters = false
	config.ECDSA.AllowedAddresses = []string{signingAddr.Hex()}
	signVerifyECDSA, err := NewSignVerify(&config, dataSigner, nil)
	Require(t, err)

	configSymmetric := DefaultSignVerifyConfig
	configSymmetric.SymmetricFallback = true
	configSymmetric.SymmetricSign = true
	configSymmetric.ECDSA.AcceptBatchPosters = false
	signVerifySymmetric, err := NewSignVerify(&configSymmetric, nil, nil)
	Require(t, err)

	configFallback := DefaultSignVerifyConfig
	configFallback.SymmetricFallback = true
	configFallback.SymmetricSign = false
	configFallback.ECDSA.AllowedAddresses = []string{signingAddr.Hex()}
	configFallback.ECDSA.AcceptBatchPosters = false
	signVerifyFallback, err := NewSignVerify(&configFallback, dataSigner, nil)
	Require(t, err)

	data := []byte{0, 1, 2, 3, 4, 5, 6, 7}

	ecdsaSig, err := signVerifyECDSA.SignMessage(data)
	Require(t, err, "error signing data")

	verified, err := signVerifyECDSA.VerifySignature(ecdsaSig, data)
	Require(t, err, "error verifying data")
	if !verified {
		t.Error("signature not verified")
	}

	verified, err = signVerifyFallback.VerifySignature(ecdsaSig, data)
	Require(t, err, "error verifying data")
	if !verified {
		t.Error("signature not verified")
	}

	verified, _ = signVerifySymmetric.VerifySignature(ecdsaSig, data)
	if verified {
		t.Error("wrong signature verified")
	}

	symSig, err := signVerifySymmetric.SignMessage(data)
	Require(t, err, "error signing data")

	verified, err = signVerifySymmetric.VerifySignature(symSig, data)
	Require(t, err, "error verifying data")
	if !verified {
		t.Error("signature not verified")
	}

	verified, err = signVerifyFallback.VerifySignature(symSig, data)
	Require(t, err, "error verifying data")
	if !verified {
		t.Error("signature not verified")
	}

	verified, _ = signVerifyECDSA.VerifySignature(symSig, data)
	if verified {
		t.Error("wrong signature verified")
	}

	fallbackSig, err := signVerifyFallback.SignMessage(data)
	Require(t, err, "error signing data")

	verified, err = signVerifyECDSA.VerifySignature(fallbackSig, data)
	Require(t, err, "error verifying data")
	if !verified {
		t.Error("signature not verified")
	}
}
