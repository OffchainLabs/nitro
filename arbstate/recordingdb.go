//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbstate

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbos"
)

type RecordingDb struct {
	inner         ethdb.KeyValueStore
	readDbEntries map[common.Hash][]byte
}

func NewRecordingDb(inner ethdb.KeyValueStore) *RecordingDb {
	return &RecordingDb{inner, make(map[common.Hash][]byte)}
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

func (db RecordingDb) GetRecordedEntries() map[common.Hash][]byte {
	return db.readDbEntries
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

type RecordingChainContext struct {
	bc                     core.ChainContext
	minBlockNumberAccessed uint64
	initialBlockNumber     uint64
}

func NewRecordingChainContext(inner core.ChainContext, blocknumber uint64) *RecordingChainContext {
	return &RecordingChainContext{
		bc:                     inner,
		minBlockNumberAccessed: blocknumber,
		initialBlockNumber:     blocknumber,
	}
}

func (r *RecordingChainContext) Engine() consensus.Engine {
	return r.bc.Engine()
}

func (r *RecordingChainContext) GetHeader(hash common.Hash, num uint64) *types.Header {
	if num == 0 {
		return nil
	}
	if num < r.minBlockNumberAccessed {
		r.minBlockNumberAccessed = num
	}
	return r.bc.GetHeader(hash, num)
}

func (r *RecordingChainContext) GetMinBlockNumberAccessed() uint64 {
	return r.minBlockNumberAccessed
}

func CreateBlockBuilder(blockchain *core.BlockChain, lastblockHash common.Hash, recording bool) (*arbos.BlockBuilder, error) {
	lastBlockHeader := blockchain.GetHeaderByHash(lastblockHash)
	if lastBlockHeader == nil {
		return nil, errors.New("block header not found")
	}
	statedb, err := blockchain.StateAt(lastBlockHeader.Root)
	if err != nil {
		return nil, err
	}
	if !recording {
		return arbos.NewBlockBuilder(lastBlockHeader, statedb, blockchain, nil, nil, nil), nil
	}
	rawKeyValue := blockchain.StateCache().TrieDB().DiskDB()
	recordingKeyValue := NewRecordingDb(rawKeyValue)
	recordingStateDatabase := state.NewDatabase(rawdb.NewDatabase(recordingKeyValue))
	recordingStateDb, err := state.New(lastBlockHeader.Root, recordingStateDatabase, nil)
	if err != nil {
		return nil, err
	}
	if !lastBlockHeader.Number.IsUint64() {
		return nil, errors.New("block number not uint64")
	}
	recordingChainContext := NewRecordingChainContext(blockchain, lastBlockHeader.Number.Uint64())
	return arbos.NewBlockBuilder(lastBlockHeader, statedb, blockchain, recordingStateDb, recordingChainContext, recordingKeyValue), nil
}

func GetRecordsFromBuilder(builder *arbos.BlockBuilder) (map[common.Hash][]byte, error) {
	recordingStateDb := builder.RecordingStateDB()
	if recordingStateDb == nil {
		return nil, errors.New("no recording statedb")
	}
	recordingKeyValue := builder.RecordingKeyValue()
	recordingDb, ok := recordingKeyValue.(*RecordingDb)
	if !ok {
		return nil, errors.New("statedb does not have valid records")
	}
	entries := recordingDb.GetRecordedEntries()
	chainContextIf := builder.RecordingChainContext()
	recordingChainContext, ok := chainContextIf.(*RecordingChainContext)
	if (recordingChainContext == nil) || (!ok) {
		return nil, errors.New("recordingChainContext invalid")
	}
	blockchain, ok := recordingChainContext.bc.(*core.BlockChain)
	if (blockchain == nil) || (!ok) {
		return nil, errors.New("blockchain invalid")
	}
	for i := recordingChainContext.GetMinBlockNumberAccessed(); i <= recordingChainContext.initialBlockNumber; i++ {
		header := blockchain.GetHeaderByNumber(i)
		hash := header.Hash()
		bytes, err := rlp.EncodeToBytes(header)
		if err != nil {
			panic(fmt.Sprintf("Error RLP encoding header: %v\n", err))
		}
		entries[hash] = bytes
	}
	return entries, nil
}
