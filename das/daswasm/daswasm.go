// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build js && wasm

// This is a small library intended to be built for WASM for generating the DAS keyset binary
// and BLS keypairs from the Orbit UI.

package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"syscall/js"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
)

type BackendConfig struct {
	PubKeyBytes []byte
	SignerMask  uint64
}

func serializeKeysetImpl(backends []BackendConfig, assumedHonest int) ([32]byte, []byte, error) {
	var aggSignersMask uint64
	pubKeys := []blsSignatures.PublicKey{}
	for _, d := range backends {
		if bits.OnesCount64(d.SignerMask) != 1 {
			return [32]byte{}, nil, fmt.Errorf("tried create keyset with a backend with invalid SignerMask %X", d.SignerMask)
		}
		pubKey, err := blsSignatures.PublicKeyFromBytes(d.PubKeyBytes, false)
		if err != nil {
			return [32]byte{}, nil, err
		}

		aggSignersMask |= d.SignerMask
		pubKeys = append(pubKeys, pubKey)
	}
	if bits.OnesCount64(aggSignersMask) != len(backends) {
		return [32]byte{}, nil, errors.New("at least two signers share a mask")
	}

	keyset := &arbstate.DataAvailabilityKeyset{
		AssumedHonest: uint64(assumedHonest),
		PubKeys:       pubKeys,
	}
	ksBuf := bytes.NewBuffer([]byte{})
	if err := keyset.Serialize(ksBuf); err != nil {
		return [32]byte{}, nil, err
	}
	keysetHash, err := keyset.Hash()
	if err != nil {
		return [32]byte{}, nil, err
	}

	return keysetHash, ksBuf.Bytes(), nil

}

// Serializes a keyset configuration into its binary representation, for use when
// calling the inbox contract.
// It takes a json object with top-level object with the following format:
//
//	{ "keyset": {
//	    "assumed-honest": integer,
//	    "backends": [{
//	      "pubkey": string of base64 encoding of committee member's bls public key,
//	      "signermask": integer bitmask for this committee member (should start from 1)
//	      }, {..
//	    }]
//	  }
//	}
//
// It returns a map that is converted to a JSON object with the following format
//
//	{
//	  keyset-hash: string of base64 encoding of the keyset's hash,
//	  keyset: string of base64 encoding of the keyset binary representation
//	}
func serializeKeyset(val []js.Value) any {
	keyset := val[0].Get("keyset")
	assumedHonest := keyset.Get("assumed-honest")
	backends := keyset.Get("backends")
	var backendConfigs []BackendConfig
	for i := 0; i < backends.Length(); i++ {
		pubKeyEncodedBytes := backends.Index(i).Get("pubkey")
		pubKeyDecoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader([]byte(pubKeyEncodedBytes.String())))
		pubKeyBytes, err := io.ReadAll(pubKeyDecoder)
		if err != nil {
			panic(err)
		}

		signerMask := backends.Index(i).Get("signermask")
		backendConfigs = append(backendConfigs, BackendConfig{
			PubKeyBytes: pubKeyBytes,
			SignerMask:  uint64(signerMask.Int()),
		})
	}

	keysetHash, keysetBytes, err := serializeKeysetImpl(backendConfigs, assumedHonest.Int())
	if err != nil {
		panic(err)
	}

	ret := make(map[string]interface{})
	ret["keyset-hash"] = encodeBase64(keysetHash[:])
	ret["keyset"] = encodeBase64(keysetBytes)
	return ret
}

// Generates a new BLS keypair, returns the result as a map which will be converted to a
// JSON object with fields "bls-public-key" and "bls-private-key" as strings of the base64
// encoded data.
func generateKey(val []js.Value) any {
	pubKey, privKey, err := blsSignatures.GenerateKeys()
	if err != nil {
		panic(err)
	}
	pubKeyBytes := blsSignatures.PublicKeyToBytes(pubKey)
	privKeyBytes := blsSignatures.PrivateKeyToBytes(privKey)

	ret := make(map[string]interface{})
	ret["bls-public-key"] = encodeBase64(pubKeyBytes)
	ret["bls-private-key"] = encodeBase64(privKeyBytes)
	return ret
}

func main() {
	js.Global().Set("serializeKeyset", js.FuncOf(func(_ js.Value, args []js.Value) any { return serializeKeyset(args) }))
	js.Global().Set("generateBLSKeypair", js.FuncOf(func(_ js.Value, args []js.Value) any { return generateKey(args) }))
	c := make(chan struct{}, 0)
	<-c // keep the program running so the callback can be called
}

func encodeBase64(b []byte) string {
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(b)))
	base64.StdEncoding.Encode(encoded, b)
	return string(encoded)
}
