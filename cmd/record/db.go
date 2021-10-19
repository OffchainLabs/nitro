//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
)

type RecordingDb struct {
	inner         ethdb.Database
	readDbEntries map[common.Hash][]byte
}

func (db RecordingDb) Has(key []byte) (bool, error) {
	if len(key) != 32 {
		return false, nil
	}
	return false, errors.New("Recording DB doesn't support Has")
}

func (db RecordingDb) Get(key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("Recording DB attempted to access non-hash key %v", hex.EncodeToString(key))
	}
	var hash common.Hash
	copy(hash[:], key)
	res, err := db.inner.Get(key)
	if err != nil {
		return nil, err
	}
	if crypto.Keccak256Hash(res) != hash {
		return nil, fmt.Errorf("Recording DB attempted to access non-hash key %v", hash)
	}
	db.readDbEntries[hash] = res
	return res, nil
}

func (db RecordingDb) Put(key []byte, value []byte) error {
	return errors.New("Recording DB doesn't support Put")
}

func (db RecordingDb) Delete(key []byte) error {
	return errors.New("Recording DB doesn't support Delete")
}

func (db RecordingDb) NewBatch() ethdb.Batch {
	return NopBatcher{db}
}

func (db RecordingDb) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return ErrorIterator{}
}

func (db RecordingDb) Stat(property string) (string, error) {
	return "", errors.New("Recording DB doesn't support Stat")
}

func (db RecordingDb) Compact(start []byte, limit []byte) error {
	return nil
}

func (db RecordingDb) Close() error {
	return nil
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
	return errors.New("Recording DB doesn't support iterators")
}

func (i ErrorIterator) Key() []byte {
	return []byte{}
}

func (i ErrorIterator) Value() []byte {
	return []byte{}
}

func (i ErrorIterator) Release() {}
