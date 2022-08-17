// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package hashing

import (
	"golang.org/x/crypto/sha3"

	"github.com/ethereum/go-ethereum/common"
)

func SoliditySHA3(data ...[]byte) common.Hash {
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
