// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dastree

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/util/pretty"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestDASTree(t *testing.T) {
	store := make(map[bytes32][]byte)
	tests := [][]byte{{}, {0x32}, crypto.Keccak256(), crypto.Keccak256([]byte{0x32})}
	for i := 0; i < 8; i++ {
		large := make([]byte, rand.Intn(8*binSize))
		tests = append(tests, large)
	}

	record := func(key bytes32, value []byte) {
		store[key] = value
	}
	oracle := func(key bytes32) []byte {
		preimage, ok := store[key]
		if !ok {
			t.Error("no preimage for key", key)
			return []byte{}
		}
		return preimage
	}

	for _, test := range tests {
		hash := RecordHash(record, test)
		store[hash] = test
	}

	for key, value := range store {
		preimage, err := Content(key, oracle)
		Require(t, err, key)

		if !bytes.Equal(preimage, value) {
			Fail(t, "incorrect preimage", pretty.FirstFewBytes(preimage), pretty.FirstFewBytes(value))
		}
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
