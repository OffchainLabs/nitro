// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"

	"github.com/offchainlabs/nitro/blsSignatures"
)

// Note for Decode functions
// Ethereum's BLS library doesn't like the byte slice containing the BLS keys to be
// any larger than necessary, so we need to create a Decoder to avoid returning any padding.

func DecodeBase64BLSPublicKey(pubKeyEncodedBytes []byte) (*blsSignatures.PublicKey, error) {
	pubKeyDecoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(pubKeyEncodedBytes))
	pubKeyBytes, err := ioutil.ReadAll(pubKeyDecoder)
	if err != nil {
		return nil, err
	}
	pubKey, err := blsSignatures.PublicKeyFromBytes(pubKeyBytes, true)
	if err != nil {
		return nil, err
	}
	return &pubKey, nil
}

func DecodeBase64BLSPrivateKey(privKeyEncodedBytes []byte) (*blsSignatures.PrivateKey, error) {
	privKeyDecoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(privKeyEncodedBytes))
	privKeyBytes, err := ioutil.ReadAll(privKeyDecoder)
	if err != nil {
		return nil, err
	}
	privKey, err := blsSignatures.PrivateKeyFromBytes(privKeyBytes)
	if err != nil {
		return nil, err
	}
	return &privKey, nil
}
