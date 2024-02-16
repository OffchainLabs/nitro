// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package dastree

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/pretty"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestDASTree(t *testing.T) {
	store := make(map[bytes32][]byte)
	tests := [][]byte{
		{}, {0x32}, crypto.Keccak256(),
		make([]byte, BinSize), make([]byte, BinSize+1), make([]byte, 4*BinSize),
	}
	for i := 0; i < 64; i++ {
		large := make([]byte, rand.Intn(12*BinSize))
		tests = append(tests, large)
	}

	record := func(key bytes32, value []byte) {
		colors.PrintGrey("storing ", key, " ", pretty.PrettyBytes(value))
		store[key] = value
		if crypto.Keccak256Hash(value) != key {
			Fail(t, "key not the hash of value")
		}
	}
	oracle := func(key bytes32) ([]byte, error) {
		preimage, ok := store[key]
		if !ok {
			Fail(t, "no preimage for key", key)
		}
		if crypto.Keccak256Hash(preimage) != key {
			Fail(t, "key not the hash of preimage")
		}
		colors.PrintBlue("loading ", key, " ", pretty.PrettyBytes(preimage))
		return preimage, nil
	}

	hashes := map[bytes32][]byte{}
	for _, test := range tests {
		hash := RecordHash(record, test)
		hashes[hash] = test
	}

	for key, value := range hashes {
		colors.PrintMint("testing ", key)
		preimage, err := Content(key, oracle)
		Require(t, err, key)

		if !bytes.Equal(preimage, value) || !ValidHash(key, preimage) {
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
