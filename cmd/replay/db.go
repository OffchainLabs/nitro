package main

import (
	"errors"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/offchainlabs/arbstate/wavmio"
)

type PreimageDb struct{}

func (db PreimageDb) Has(key []byte) (bool, error) {
	if len(key) != 32 {
		return false, nil
	}
	return false, errors.New("Preimage DB doesn't support Has")
}

func (db PreimageDb) Get(key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("Preimage DB keys must be 32 bytes long")
	}
	var hash [32]byte
	copy(hash[:], key)
	return wavmio.ResolvePreImage(hash), nil
}

func (db PreimageDb) Put(key []byte, value []byte) error {
	return errors.New("Preimage DB doesn't support Put")
}

func (db PreimageDb) Delete(key []byte) error {
	return errors.New("Preimage DB doesn't support Delete")
}

func (db PreimageDb) NewBatch() ethdb.Batch {
	return NopBatcher{db}
}

func (db PreimageDb) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return ErrorIterator{}
}

func (db PreimageDb) Stat(property string) (string, error) {
	return "", errors.New("Preimage DB doesn't support Stat")
}

func (db PreimageDb) Compact(start []byte, limit []byte) error {
	return nil
}

func (db PreimageDb) Close() error {
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
	return errors.New("Preimage DB doesn't support iterators")
}

func (i ErrorIterator) Key() []byte {
	return []byte{}
}

func (i ErrorIterator) Value() []byte {
	return []byte{}
}

func (i ErrorIterator) Release() {}
