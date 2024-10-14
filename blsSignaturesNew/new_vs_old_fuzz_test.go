// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package blsSignaturesNew

import (
	"bytes"
	"testing"

	"github.com/offchainlabs/nitro/blsSignatures"
)

func FuzzSignatureSerialization(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		oldSig, oldErr := blsSignatures.SignatureFromBytes(data)
		newSig, newErr := SignatureFromBytes(data)

		if (oldErr != nil) != (newErr != nil) {
			t.Fatalf("error mismatch: %v vs %v", oldErr, newErr)
		}
		if oldErr == nil {
			oldSigBytes := blsSignatures.SignatureToBytes(oldSig)
			newSigBytes := SignatureToBytes(newSig)
			if !bytes.Equal(oldSigBytes, newSigBytes) {
				t.Fatalf("signature mismatch: %x vs %x", oldSigBytes, newSigBytes)
			}
		}
	})
}

func FuzzSignatureValidation(f *testing.F) {
	// use the zero key so fuzzing can create valid signatures
	newPrivateKey, err := PrivateKeyFromBytes([]byte{})
	if err != nil {
		f.Fatalf("failed to generate new private key: %v", err)
	}
	newPublicKey, err := PublicKeyFromPrivateKey(newPrivateKey)
	if err != nil {
		f.Fatalf("failed to generate new public key: %v", err)
	}
	oldPrivateKey, err := blsSignatures.PrivateKeyFromBytes([]byte{})
	if err != nil {
		f.Fatalf("failed to generate old private key: %v", err)
	}
	oldPublicKey, err := blsSignatures.PublicKeyFromPrivateKey(oldPrivateKey)
	if err != nil {
		f.Fatalf("failed to generate old public key: %v", err)
	}
	message := []byte("hello world")
	f.Fuzz(func(t *testing.T, data []byte) {
		oldSig, oldErr := blsSignatures.SignatureFromBytes(data)
		newSig, newErr := SignatureFromBytes(data)

		if (oldErr != nil) != (newErr != nil) {
			t.Fatalf("signature deserialization error mismatch: %v vs %v", oldErr, newErr)
		}
		if oldErr == nil {
			oldValid, oldErr := blsSignatures.VerifySignature(oldSig, message, oldPublicKey)
			oldValid = oldErr == nil && oldValid
			newValid, newErr := VerifySignature(newSig, message, newPublicKey)
			newValid = newErr == nil && newValid
			if oldValid != newValid {
				f.Fatalf("validity mismatch: %v vs %v", oldValid, newValid)
			}
		}
	})
}
