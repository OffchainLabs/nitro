// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrecording

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/triedb"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

type TxsRecordingDatabase struct {
	underlying *triedb.Database
	recorder   daprovider.PreimageRecorder
}

func (rdb *TxsRecordingDatabase) Get(key []byte) ([]byte, error) {
	hash := common.BytesToHash(key)
	value, err := rdb.underlying.Node(hash)
	if err != nil {
		return nil, err
	}
	if rdb.recorder != nil {
		rdb.recorder(hash, value, arbutil.Keccak256PreimageType)
	}

	return value, nil
}
func (rdb *TxsRecordingDatabase) Has(key []byte) (bool, error) {
	hash := common.BytesToHash(key)
	_, err := rdb.underlying.Node(hash)
	return err == nil, nil
}
func (rdb *TxsRecordingDatabase) Put(key []byte, value []byte) error {
	return fmt.Errorf("Put not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) Delete(key []byte) error {
	return fmt.Errorf("Delete not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) DeleteRange(start, end []byte) error {
	return fmt.Errorf("DeleteRange not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) ReadAncients(fn func(ethdb.AncientReaderOp) error) (err error) {
	return fmt.Errorf("ReadAncients not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) ModifyAncients(func(ethdb.AncientWriteOp) error) (int64, error) {
	return 0, fmt.Errorf("ModifyAncients not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) SyncAncient() error {
	return fmt.Errorf("SyncAncient not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) TruncateHead(n uint64) (uint64, error) {
	return 0, fmt.Errorf("TruncateHead not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) TruncateTail(n uint64) (uint64, error) {
	return 0, fmt.Errorf("TruncateTail not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) Append(kind string, number uint64, item interface{}) error {
	return fmt.Errorf("Append not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) AppendRaw(kind string, number uint64, item []byte) error {
	return fmt.Errorf("AppendRaw not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) AncientDatadir() (string, error) {
	return "", fmt.Errorf("AncientDatadir not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, fmt.Errorf("Ancient not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	return nil, fmt.Errorf("AncientRange not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) AncientBytes(kind string, id, offset, length uint64) ([]byte, error) {
	return nil, fmt.Errorf("AncientBytes not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) Ancients() (uint64, error) {
	return 0, fmt.Errorf("Ancients not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) Tail() (uint64, error) {
	return 0, fmt.Errorf("Tail not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) AncientSize(kind string) (uint64, error) {
	return 0, fmt.Errorf("AncientSize not supported on recording DB")
}
func (rdb *TxsRecordingDatabase) Compact(start []byte, limit []byte) error {
	return nil
}
func (rdb *TxsRecordingDatabase) SyncKeyValue() error {
	return nil
}
func (rdb *TxsRecordingDatabase) Stat() (string, error) {
	return "", nil
}
func (rdb *TxsRecordingDatabase) WasmDataBase() ethdb.KeyValueStore {
	return &unsupportedKeyValueStore{}
}
func (rdb *TxsRecordingDatabase) NewBatch() ethdb.Batch {
	return &unsupportedBatch{}
}
func (rdb *TxsRecordingDatabase) NewBatchWithSize(size int) ethdb.Batch {
	return &unsupportedBatch{}
}
func (rdb *TxsRecordingDatabase) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return &emptyIterator{err: fmt.Errorf("NewIterator not supported on recording DB")}
}

// unsupportedBatch is a stub ethdb.Batch that returns errors on all write operations.
type unsupportedBatch struct{}

func (b *unsupportedBatch) Put(key []byte, value []byte) error {
	return fmt.Errorf("Put not supported on recording DB batch")
}
func (b *unsupportedBatch) Delete(key []byte) error {
	return fmt.Errorf("Delete not supported on recording DB batch")
}
func (b *unsupportedBatch) DeleteRange(start, end []byte) error {
	return fmt.Errorf("DeleteRange not supported on recording DB batch")
}
func (b *unsupportedBatch) ValueSize() int { return 0 }
func (b *unsupportedBatch) Write() error {
	return fmt.Errorf("Write not supported on recording DB batch")
}
func (b *unsupportedBatch) Reset() {}
func (b *unsupportedBatch) Replay(w ethdb.KeyValueWriter) error {
	return fmt.Errorf("Replay not supported on recording DB batch")
}

// emptyIterator is a stub ethdb.Iterator that reports an error and yields no results.
type emptyIterator struct{ err error }

func (it *emptyIterator) Next() bool    { return false }
func (it *emptyIterator) Error() error  { return it.err }
func (it *emptyIterator) Key() []byte   { return nil }
func (it *emptyIterator) Value() []byte { return nil }
func (it *emptyIterator) Release()      {}

// unsupportedKeyValueStore is a stub ethdb.KeyValueStore that returns errors on all operations.
type unsupportedKeyValueStore struct{}

func (s *unsupportedKeyValueStore) Has(key []byte) (bool, error) {
	return false, fmt.Errorf("Has not supported on recording DB WasmDataBase")
}
func (s *unsupportedKeyValueStore) Get(key []byte) ([]byte, error) {
	return nil, fmt.Errorf("Get not supported on recording DB WasmDataBase")
}
func (s *unsupportedKeyValueStore) Put(key []byte, value []byte) error {
	return fmt.Errorf("Put not supported on recording DB WasmDataBase")
}
func (s *unsupportedKeyValueStore) Delete(key []byte) error {
	return fmt.Errorf("Delete not supported on recording DB WasmDataBase")
}
func (s *unsupportedKeyValueStore) DeleteRange(start, end []byte) error {
	return fmt.Errorf("DeleteRange not supported on recording DB WasmDataBase")
}
func (s *unsupportedKeyValueStore) NewBatch() ethdb.Batch            { return &unsupportedBatch{} }
func (s *unsupportedKeyValueStore) NewBatchWithSize(int) ethdb.Batch { return &unsupportedBatch{} }
func (s *unsupportedKeyValueStore) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return &emptyIterator{err: fmt.Errorf("NewIterator not supported on recording DB WasmDataBase")}
}
func (s *unsupportedKeyValueStore) Stat() (string, error)                    { return "", nil }
func (s *unsupportedKeyValueStore) SyncKeyValue() error                      { return nil }
func (s *unsupportedKeyValueStore) Compact(start []byte, limit []byte) error { return nil }
func (s *unsupportedKeyValueStore) Close() error                             { return nil }
func (rdb *TxsRecordingDatabase) Close() error {
	return nil
}
