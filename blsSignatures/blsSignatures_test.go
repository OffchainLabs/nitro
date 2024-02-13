// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package blsSignatures

import (
	"math/rand"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/util/testhelpers"
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
	pubKeys := []PublicKey{}
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

	verified, err = VerifyAggregatedSignatureSameMessage(AggregateSignatures(sigs), message, pubKeys)
	Require(t, err)
	if !verified {
		Fail(t, "Second aggregated signature check failed")
	}
}

func TestSignatureAggregationAnyOrder(t *testing.T) {
	message := []byte("The quick brown fox jumped over the lazy dog.")
	pubKeys := []PublicKey{}
	sigs := []Signature{}
	for i := 0; i < NumSignaturesToAggregate; i++ {
		pub, priv, err := GenerateKeys()
		Require(t, err)
		pubKeys = append(pubKeys, pub)
		sig, err := SignMessage(priv, message)
		Require(t, err)
		sigs = append(sigs, sig)
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(NumSignaturesToAggregate, func(i, j int) { sigs[i], sigs[j] = sigs[j], sigs[i] })
	rand.Shuffle(NumSignaturesToAggregate, func(i, j int) { pubKeys[i], pubKeys[j] = pubKeys[j], pubKeys[i] })

	verified, err := VerifySignature(AggregateSignatures(sigs), message, AggregatePublicKeys(pubKeys))
	Require(t, err)
	if !verified {
		Fail(t, "First aggregated signature check failed")
	}

	rand.Shuffle(NumSignaturesToAggregate, func(i, j int) { sigs[i], sigs[j] = sigs[j], sigs[i] })
	verified, err = VerifyAggregatedSignatureSameMessage(AggregateSignatures(sigs), message, pubKeys)
	Require(t, err)
	if !verified {
		Fail(t, "Second aggregated signature check failed")
	}
}

func TestSignatureAggregationDifferentMessages(t *testing.T) {
	messages := [][]byte{}
	pubKeys := []PublicKey{}
	sigs := []Signature{}

	for i := 0; i < NumSignaturesToAggregate; i++ {
		msg := []byte{byte(i)}
		pubKey, privKey, err := GenerateKeys()
		Require(t, err)
		sig, err := SignMessage(privKey, msg)
		Require(t, err)
		messages = append(messages, msg)
		pubKeys = append(pubKeys, pubKey)
		sigs = append(sigs, sig)
	}

	verified, err := VerifyAggregatedSignatureDifferentMessages(AggregateSignatures(sigs), messages, pubKeys)
	Require(t, err)
	if !verified {
		Fail(t, "First aggregated signature check failed")
	}
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
