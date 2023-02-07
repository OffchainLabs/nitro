// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package signature

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
)

type DataSignerFunc func([]byte) ([]byte, error)

func DataSignerFromPrivateKey(privateKey *ecdsa.PrivateKey) DataSignerFunc {
	return func(data []byte) ([]byte, error) {
		return crypto.Sign(data, privateKey)
	}
}
