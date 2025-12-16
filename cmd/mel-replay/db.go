package main

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/offchainlabs/nitro/arbutil"
)

// Ensures that the DB implementation satisfied the ethdb.Database interface.
var _ ethdb.Database = (*DB)(nil)

type DB struct {
	resolver preimageResolver
}

func (d *DB) Get(key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("expected 32 byte key query, but got %d bytes: %x", len(key), key)
	}
	preimage, err := d.resolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, common.BytesToHash(key))
	if err != nil {
		return nil, fmt.Errorf("error resolving preimage for %#x: %w", key, err)
	}
	return preimage, nil
}

func (d *DB) Has(key []byte) (bool, error) {
	return false, errors.New("unimplemented")
}

func (d *DB) Put(key []byte, value []byte) error {
	return errors.New("unimplemented")
}

func (p DB) Delete(key []byte) error {
	return errors.New("unimplemented")
}

func (d *DB) DeleteRange(start, end []byte) error {
	return errors.New("unimplemented")
}

func (p DB) Stat() (string, error) {
	return "", errors.New("unimplemented")
}

func (p DB) NewBatch() ethdb.Batch {
	panic("unimplemented")
}

func (p DB) NewBatchWithSize(size int) ethdb.Batch {
	panic("unimplemented")
}

func (p DB) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	panic("unimplemented")
}

func (p DB) SyncAncient() error {
	return nil // no-op
}

func (p DB) SyncKeyValue() error {
	return nil // no-op
}

func (p DB) Compact(start []byte, limit []byte) error {
	return nil // no-op
}

func (p DB) Close() error {
	return nil
}

func (d *DB) HasAncient(kind string, number uint64) (bool, error) {
	return false, errors.New("unimplemented")
}

func (d *DB) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (d *DB) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	return nil, errors.New("unimplemented")
}

func (d *DB) AncientBytes(kind string, id, offset, length uint64) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (d *DB) Ancients() (uint64, error) {
	return 0, errors.New("unimplemented")
}

func (d *DB) Tail() (uint64, error) {
	return 0, errors.New("unimplemented")
}

func (d *DB) AncientSize(kind string) (uint64, error) {
	return 0, errors.New("unimplemented")
}

func (d *DB) ReadAncients(fn func(ethdb.AncientReaderOp) error) (err error) {
	panic("unimplemented")
}

func (d *DB) ModifyAncients(f func(ethdb.AncientWriteOp) error) (int64, error) {
	return 0, errors.New("unimplemented")
}

func (d *DB) TruncateHead(n uint64) (uint64, error) {
	return 0, errors.New("unimplemented")
}

func (d *DB) TruncateTail(n uint64) (uint64, error) {
	return 0, errors.New("unimplemented")
}

func (d *DB) Sync() error {
	return errors.New("unimplemented")
}

func (d *DB) MigrateTable(s string, f func([]byte) ([]byte, error)) error {
	return errors.New("unimplemented")
}

func (d *DB) AncientDatadir() (string, error) {
	return "", errors.New("unimplemented")
}

func (d *DB) WasmDataBase() ethdb.KeyValueStore {
	panic("unimplemented")
}
