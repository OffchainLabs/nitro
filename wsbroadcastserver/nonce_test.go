// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package wsbroadcastserver

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestNonceHashing(t *testing.T) {
	var nonceA common.Hash
	var nonceB common.Hash
	nonceB[0]++

	nonceAHash := computeNonceHash(nonceA)
	recomputed := computeNonceHash(nonceA)
	if nonceAHash != recomputed {
		Fail(t, "nonce hash is non-deterministic; got", nonceAHash, "and then", recomputed)
	}

	nonceBHash := computeNonceHash(nonceB)
	if nonceAHash == nonceBHash {
		Fail(t, "nonce hash is the same for A and B:", nonceAHash)
	}

	aLesser := lesserHash(nonceAHash, nonceBHash)
	if aLesser {
		if lesserHash(nonceBHash, nonceAHash) {
			Fail(t, "a < b && b < a")
		}
	} else {
		if lesserHash(nonceBHash, nonceAHash) {
			Fail(t, "a >= b && b < a")
		}
	}

	if lesserHash(nonceAHash, nonceAHash) {
		Fail(t, "a < a")
	}
}

func TestNonceOrdering(t *testing.T) {
	nonces := []common.Hash{
		common.HexToHash("0x00"),
		common.HexToHash("0x01"),
		common.HexToHash("0x02"),
		common.HexToHash("0x100"),
		common.HexToHash("0x101"),
		common.HexToHash("0x102"),
		common.HexToHash("0x10000"),
		common.HexToHash("0x10001"),
		common.HexToHash("0x10002"),
		common.HexToHash("0x20000"),
		common.HexToHash("0x20001"),
		common.HexToHash("0x20002"),
	}
	for i, nonceA := range nonces {
		for j, nonceB := range nonces {
			less := lesserHash(nonceA, nonceB)
			expected := i < j
			if less != expected {
				Fail(t, "expected lesserHash(", nonceA, ",", nonceB, ") to be", expected, "but got", less)
			}
		}
	}
}
