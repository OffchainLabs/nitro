package data_streaming

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

var testBytes = []byte("test payload data")
var testExtras = []uint64{10, 20, 30}

func TestSigningSchemes(t *testing.T) {
	cryptoSigner, cryptVerifier := prepareCrypto(t)
	testSynergy(t, DefaultPayloadSigner(cryptoSigner), DefaultPayloadVerifier(cryptVerifier))
	testSynergy(t, PayloadCommiter(), PayloadCommitmentVerifier())
}

func testSynergy(t *testing.T, payloadSigner *PayloadSigner, payloadVerifier *PayloadVerifier) {
	sig, err := payloadSigner.signPayload(testBytes, testExtras...)
	testhelpers.RequireImpl(t, err)
	err = payloadVerifier.verifyPayload(context.Background(), sig, testBytes, testExtras...)
	testhelpers.RequireImpl(t, err)

	sig2, err := payloadSigner.signPayload(append(testBytes, 0), testExtras...)
	testhelpers.RequireImpl(t, err)
	err = payloadVerifier.verifyPayload(context.Background(), sig2, testBytes, testExtras...)
	require.Error(t, err)
}

func prepareCrypto(t *testing.T) (signature.DataSignerFunc, *signature.Verifier) {
	privateKey, err := crypto.GenerateKey()
	testhelpers.RequireImpl(t, err)

	signatureVerifierConfig := signature.VerifierConfig{
		AllowedAddresses: []string{crypto.PubkeyToAddress(privateKey.PublicKey).Hex()},
		AcceptSequencer:  false,
		Dangerous:        signature.DangerousVerifierConfig{AcceptMissing: false},
	}
	verifier, err := signature.NewVerifier(&signatureVerifierConfig, nil)
	testhelpers.RequireImpl(t, err)

	signer := signature.DataSignerFromPrivateKey(privateKey)
	return signer, verifier
}
