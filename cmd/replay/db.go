// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/wavmio"
)

type PreimageDb struct{}

func (db PreimageDb) Has(key []byte) (bool, error) {
	if len(key) != 32 {
		return false, nil
	}
	return false, errors.New("preimage DB doesn't support Has")
}

func (db PreimageDb) Get(key []byte) ([]byte, error) {
	var hash [32]byte
	copy(hash[:], key)
	if len(key) == 32 {
		copy(hash[:], key)
	} else if len(key) == len(rawdb.CodePrefix)+32 && bytes.HasPrefix(key, rawdb.CodePrefix) {
		// Retrieving code
		copy(hash[:], key[len(rawdb.CodePrefix):])
	} else {
		return nil, fmt.Errorf("preimage DB attempted to access non-hash key %v", hex.EncodeToString(key))
	}
	return wavmio.ResolveTypedPreimage(arbutil.Keccak256PreimageType, hash)
}

func (db PreimageDb) Put(key []byte, value []byte) error {
	return errors.New("preimage DB doesn't support Put")
}

func (db PreimageDb) Delete(key []byte) error {
	return errors.New("preimage DB doesn't support Delete")
}

func (db PreimageDb) NewBatch() ethdb.Batch {
	return NopBatcher{db}
}

func (db PreimageDb) NewBatchWithSize(size int) ethdb.Batch {
	return NopBatcher{db}
}

func (db PreimageDb) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return ErrorIterator{}
}

func (db PreimageDb) Stat(property string) (string, error) {
	return "", errors.New("preimage DB doesn't support Stat")
}

func (db PreimageDb) Compact(start []byte, limit []byte) error {
	return nil
}

func (db PreimageDb) NewSnapshot() (ethdb.Snapshot, error) {
	// This is fine as PreimageDb doesn't support mutation
	return db, nil
}

func (db PreimageDb) Close() error {
	return nil
}

func (db PreimageDb) Release() {
}

type NopBatcher struct {
	ethdb.KeyValueStore
}

func (b NopBatcher) ValueSize() int {
	return 0
}

func (b NopBatcher) Write() error {
	return nil
}

func (b NopBatcher) Reset() {}

func (b NopBatcher) Replay(w ethdb.KeyValueWriter) error {
	return nil
}

type ErrorIterator struct{}

func (i ErrorIterator) Next() bool {
	return false
}

func (i ErrorIterator) Error() error {
	return errors.New("preimage DB doesn't support iterators")
}

func (i ErrorIterator) Key() []byte {
	return []byte{}
}

func (i ErrorIterator) Value() []byte {
	return []byte{}
}

func (i ErrorIterator) Release() {}
