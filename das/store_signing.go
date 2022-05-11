// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func DasSignStore(message []byte, timeout uint64, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	return crypto.Sign(dasStoreHash(message, timeout), privateKey)
}

func DasRecoverSigner(message []byte, timeout uint64, sig []byte) (common.Address, error) {
	pk, err := crypto.SigToPub(dasStoreHash(message, timeout), sig)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*pk), nil
}

func dasStoreHash(message []byte, timeout uint64) []byte {
	return []byte{}
}
