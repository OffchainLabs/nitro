// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package wsbroadcastserver

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestNonceHashing(t *testing.T) {
	now := time.Now()
	var nonceA common.Hash
	var nonceB common.Hash
	nonceB[0]++

	nonceAHash := computeNonceHash(nonceA, 0, now)
	recomputed := computeNonceHash(nonceA, 0, now)
	if nonceAHash != recomputed {
		Fail(t, "nonce hash is non-deterministic; got", nonceAHash, "and then", recomputed)
	}

	nonceBHash := computeNonceHash(nonceB, 0, now)
	if nonceAHash == nonceBHash {
		Fail(t, "nonce hash is the same for A and B:", nonceAHash)
	}

	nonceAScore := scoreNonceHash(nonceAHash)
	recomputedScore := scoreNonceHash(nonceAHash)
	if nonceAScore != recomputedScore {
		Fail(t, "nonce hash score is non-deterministic; got", nonceAScore, "and then", recomputedScore)
	}

	nonceBScore := scoreNonceHash(nonceBHash)
	if nonceAScore == nonceBScore {
		Fail(t, "nonce hash score is the same for A and B:", nonceAScore)
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
			better := scoreNonceHash(nonceA) > scoreNonceHash(nonceB)
			expected := i < j
			if better != expected {
				Fail(t, "expected (scoreNonceHash(", nonceA, ") > scoreNonceHash(", nonceB, ")) to be", expected, "but got", better)
			}
		}
	}
}
