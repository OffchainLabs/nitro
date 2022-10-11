// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/ethereum/go-ethereum/ethdb"
	"io"
)

type mockDB struct {
	ethdb.Reader
	ethdb.Writer
	ethdb.Batcher
	ethdb.Iteratee
	ethdb.Stater
	ethdb.Compacter
	ethdb.Snapshotter
	io.Closer

	messages map[string][]byte
}

func newMockDB() *mockDB {
	return &mockDB{
		messages: make(map[string][]byte),
	}
}

func (m *mockDB) Has(key []byte) (bool, error) {
	_, ok := m.messages[hex.EncodeToString(key)]
	return ok, nil
}

func (m *mockDB) Delete(key []byte) error {
	delete(m.messages, hex.EncodeToString(key))

	return nil
}

func (m *mockDB) Get(key []byte) ([]byte, error) {
	val, ok := m.messages[hex.EncodeToString(key)]
	if !ok {
		return nil, errors.New("not found")
	}

	return val, nil
}

func (m *mockDB) Put(key []byte, value []byte) error {
	m.messages[hex.EncodeToString(key)] = value

	return nil
}

func (m *mockDB) NewIterator(prefix []byte, minKey []byte) ethdb.Iterator {
	return newMockIter(m, prefix, minKey)
}

func (m *mockDB) NewBatch() ethdb.Batch {
	return newMockBatch(m)
}

type mockBatch struct {
	db *mockDB

	messages map[string][]byte
}

func newMockBatch(db *mockDB) *mockBatch {
	b := mockBatch{
		db:       db,
		messages: make(map[string][]byte),
	}

	return &b
}

func (m *mockBatch) Put(key []byte, value []byte) error {
	m.messages[hex.EncodeToString(key)] = value

	return nil
}

func (m *mockBatch) Delete(key []byte) error {
	return m.db.Delete(key)
}

func (m *mockBatch) ValueSize() int {
	return 0
}

func (m *mockBatch) Write() error {
	for encodedKey, value := range m.messages {
		key, err := hex.DecodeString(encodedKey)
		if err != nil {
			return err
		}
		if err := m.db.Put(key, value); err != nil {
			return err
		}
	}

	return nil
}

func (m *mockBatch) Reset() {
}

func (m *mockBatch) Replay(_ ethdb.KeyValueWriter) error {
	return nil
}

func (m *mockBatch) NetBatchWithSize() ethdb.Batcher {
	return newMockBatch(m.db)
}

func (m mockBatch) NewBatch() ethdb.Batch {
	return newMockBatch(m.db)
}

func (m mockBatch) NewBatchWithSize(_ int) ethdb.Batch {
	return newMockBatch(m.db)
}

type mockIter struct {
	db           *mockDB
	prefix       []byte
	currentKey   uint64
	currentValue []byte
	err          error
}

func newMockIter(db *mockDB, prefix []byte, minKey []byte) *mockIter {
	key := binary.BigEndian.Uint64(minKey)
	return &mockIter{
		db:         db,
		prefix:     prefix,
		currentKey: key,
	}
}

func (i *mockIter) Release() {
}

func (i *mockIter) Error() error {
	return i.err
}

func (i *mockIter) Next() bool {
	currentKey := dbKey(i.prefix, i.currentKey)
	has, err := i.db.Has(currentKey)
	if err != nil {
		i.err = err
		return false
	}
	if !has {
		return false
	}

	value, err := i.db.Get(currentKey)
	if err != nil {
		i.err = err
		return false
	}
	i.currentValue = value

	i.currentKey++

	return true
}

func (i *mockIter) Key() []byte {
	return dbKey(i.prefix, i.currentKey)
}

func (i *mockIter) Value() []byte {
	return i.currentValue
}
