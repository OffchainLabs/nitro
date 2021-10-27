//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package utils

import (
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
)

func Keccak256(data ...[]byte) common.Hash {
	var ret common.Hash
	hash := sha3.NewLegacyKeccak256()
	for _, b := range data {
		_, err := hash.Write(b)
		if err != nil {
			// This code should never be reached
			panic("Error writing SoliditySHA3 data")
		}
	}
	hash.Sum(ret[:0])
	return ret
}
