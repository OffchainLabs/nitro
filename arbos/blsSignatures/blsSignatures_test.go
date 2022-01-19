//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package blsSignatures

import (
	"github.com/offchainlabs/arbstate/util/testhelpers"
	"testing"
)

func TestValidSignature(t *testing.T) {
	pub, priv, err := GenerateKeys()
	Require(t, err)

	message := []byte("The quick brown fox jumped over the lazy dog.")
	sig, err := SignMessage(priv, message)
	Require(t, err)

	verified, err := VerifySignature(sig, message, pub)
	Require(t, err)
	if !verified {
		Fail(t, "valid signature failed to verify")
	}
}

func TestWrongMessageSignature(t *testing.T) {
	pub, priv, err := GenerateKeys()
	Require(t, err)

	message := []byte("The quick brown fox jumped over the lazy dog.")
	sig, err := SignMessage(priv, message)
	Require(t, err)

	verified, err := VerifySignature(sig, append(message, 3), pub)
	Require(t, err)
	if verified {
		Fail(t, "signature check on wrong message didn't fail")
	}
}

func TestWrongKeySignature(t *testing.T) {
	_, priv, err := GenerateKeys()
	Require(t, err)
	pub, _, err := GenerateKeys()
	Require(t, err)

	message := []byte("The quick brown fox jumped over the lazy dog.")
	sig, err := SignMessage(priv, message)
	Require(t, err)

	verified, err := VerifySignature(sig, message, pub)
	Require(t, err)
	if verified {
		Fail(t, "signature check with wrong public key didn't fail")
	}
}

const NumSignaturesToAggregate = 12

func TestSignatureAggregation(t *testing.T) {
	message := []byte("The quick brown fox jumped over the lazy dog.")
	pubKeys := []*PublicKey{}
	sigs := []Signature{}
	for i := 0; i < NumSignaturesToAggregate; i++ {
		pub, priv, err := GenerateKeys()
		Require(t, err)
		pubKeys = append(pubKeys, pub)
		sig, err := SignMessage(priv, message)
		Require(t, err)
		sigs = append(sigs, sig)
	}

	verified, err := VerifySignature(AggregateSignatures(sigs), message, AggregatePublicKeys(pubKeys))
	Require(t, err)
	if !verified {
		Fail(t, "First aggregated signature check failed")
	}

	verified, err = VerifyAggregatedSignature(AggregateSignatures(sigs), message, pubKeys)
	Require(t, err)
	if !verified {
		Fail(t, "Second aggregated signature check failed")
	}
}

func Require(t *testing.T, err error, text ...string) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
