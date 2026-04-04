// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrunner

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/db/read"
	"github.com/offchainlabs/nitro/arbnode/db/schema"
	"github.com/offchainlabs/nitro/arbnode/mel"
)

// initialBoundary holds the cached boundary from the initial MEL state (set
// during legacy migration). Indices below these thresholds are read from
// legacy (pre-MEL) schema keys.
type initialBoundary struct {
	blockNum     uint64
	delayedCount uint64
	batchCount   uint64
}

// Database wraps an ethdb.KeyValueStore and provides MEL state persistence,
// batch metadata and delayed message storage, verified delayed message reads
// (via ReadDelayedMessage, which validates against the outbox accumulator),
// and raw unverified fetches (via FetchDelayedMessage).
// For legacy-migrated nodes, it dispatches reads below the initial MEL boundary
// to pre-MEL schema keys.
type Database struct {
	db ethdb.KeyValueStore

	// Set once during initialization via cacheInitialBoundary; read concurrently
	// by delayed sequencer, block validator, staker, and RPC handlers.
	// Using atomic.Pointer provides the memory barrier between the initializing
	// goroutine and concurrent readers.
	boundary atomic.Pointer[initialBoundary]
}

func NewDatabase(db ethdb.KeyValueStore) (*Database, error) {
	d := &Database{db: db}
	if err := d.loadInitialBoundary(); err != nil {
		return nil, fmt.Errorf("failed to load initial MEL state boundary: %w", err)
	}
	return d, nil
}

// cacheInitialBoundary atomically sets the cached boundary used for legacy key dispatch.
func (d *Database) cacheInitialBoundary(blockNum uint64, delayedCount, batchCount uint64) {
	d.boundary.Store(&initialBoundary{
		blockNum:     blockNum,
		delayedCount: delayedCount,
		batchCount:   batchCount,
	})
}

// loadInitialBoundary loads the initial MEL state boundary counts from the DB.
// If InitialMelStateBlockNumKey exists, it reads the initial state to cache
// the delayed message and batch count thresholds for legacy key dispatch.
// Returns nil if the key does not exist (fresh node with no legacy boundary).
func (d *Database) loadInitialBoundary() error {
	blockNum, err := read.Value[uint64](d.db, schema.InitialMelStateBlockNumKey)
	if err != nil {
		if rawdb.IsDbErrNotFound(err) {
			return nil
		}
		return fmt.Errorf("error reading InitialMelStateBlockNumKey: %w", err)
	}
	state, err := d.State(blockNum)
	if err != nil {
		return fmt.Errorf("error reading initial MEL state at block %d: %w", blockNum, err)
	}
	d.cacheInitialBoundary(blockNum, state.DelayedMessagesSeen, state.BatchCount)
	return nil
}

func (d *Database) SaveInitialMelState(initialState *mel.State) error {
	if d.boundary.Load() != nil {
		return errors.New("initial MEL state already set; cannot re-initialize legacy boundary")
	}
	dbBatch := d.db.NewBatch()
	encoded, err := rlp.EncodeToBytes(initialState.ParentChainBlockNumber)
	if err != nil {
		return fmt.Errorf("encoding initial block number: %w", err)
	}
	if err := dbBatch.Put(schema.InitialMelStateBlockNumKey, encoded); err != nil {
		return fmt.Errorf("writing initial block number key: %w", err)
	}
	if err := d.setMelState(dbBatch, initialState.ParentChainBlockNumber, *initialState); err != nil {
		return fmt.Errorf("writing initial MEL state at block %d: %w", initialState.ParentChainBlockNumber, err)
	}
	if err := d.setHeadMelStateBlockNum(dbBatch, initialState.ParentChainBlockNumber); err != nil {
		return fmt.Errorf("writing head block number: %w", err)
	}
	if err := dbBatch.Write(); err != nil {
		return fmt.Errorf("committing initial MEL state batch: %w", err)
	}
	d.cacheInitialBoundary(initialState.ParentChainBlockNumber, initialState.DelayedMessagesSeen, initialState.BatchCount)
	return nil
}

func (d *Database) GetHeadMelState() (*mel.State, error) {
	headMelStateBlockNum, err := d.GetHeadMelStateBlockNum()
	if err != nil {
		return nil, fmt.Errorf("error getting HeadMelStateBlockNum from database: %w", err)
	}
	return d.State(headMelStateBlockNum)
}

// SaveState persists just the MEL state and head pointer. Used during node
// initialization and in tests. Production block processing should use
// SaveProcessedBlock for atomic writes.
func (d *Database) SaveState(state *mel.State) error {
	dbBatch := d.db.NewBatch()
	if err := d.setMelState(dbBatch, state.ParentChainBlockNumber, *state); err != nil {
		return fmt.Errorf("SaveState: writing MEL state at block %d: %w", state.ParentChainBlockNumber, err)
	}
	if err := d.setHeadMelStateBlockNum(dbBatch, state.ParentChainBlockNumber); err != nil {
		return fmt.Errorf("SaveState: writing head block num: %w", err)
	}
	if err := dbBatch.Write(); err != nil {
		return fmt.Errorf("SaveState: committing batch for block %d: %w", state.ParentChainBlockNumber, err)
	}
	return nil
}

func putBatchMetas(batch ethdb.KeyValueWriter, state *mel.State, batchMetas []*mel.BatchMetadata) error {
	if state.BatchCount < uint64(len(batchMetas)) {
		return fmt.Errorf("mel state's BatchCount: %d is lower than number of batchMetadata: %d queued to be added", state.BatchCount, len(batchMetas))
	}
	firstPos := state.BatchCount - uint64(len(batchMetas))
	for i, batchMetadata := range batchMetas {
		seqNum := firstPos + uint64(i) // #nosec G115
		key := read.Key(schema.MelSequencerBatchMetaPrefix, seqNum)
		batchMetadataBytes, err := rlp.EncodeToBytes(*batchMetadata)
		if err != nil {
			return fmt.Errorf("encoding batch metadata at seqNum %d: %w", seqNum, err)
		}
		if err := batch.Put(key, batchMetadataBytes); err != nil {
			return fmt.Errorf("writing batch metadata at seqNum %d: %w", seqNum, err)
		}
	}
	return nil
}

func putDelayedMessages(batch ethdb.KeyValueWriter, state *mel.State, delayedMessages []*mel.DelayedInboxMessage) error {
	if state.DelayedMessagesSeen < uint64(len(delayedMessages)) {
		return fmt.Errorf("mel state's DelayedMessagesSeen: %d is lower than number of delayed messages: %d queued to be added", state.DelayedMessagesSeen, len(delayedMessages))
	}
	firstPos := state.DelayedMessagesSeen - uint64(len(delayedMessages))
	for i, msg := range delayedMessages {
		index := firstPos + uint64(i) // #nosec G115
		key := read.Key(schema.MelDelayedMessagePrefix, index)
		delayedBytes, err := rlp.EncodeToBytes(*msg)
		if err != nil {
			return fmt.Errorf("encoding delayed message at index %d: %w", index, err)
		}
		if err := batch.Put(key, delayedBytes); err != nil {
			return fmt.Errorf("writing delayed message at index %d: %w", index, err)
		}
	}
	return nil
}

// SaveProcessedBlock atomically writes batch metadata, delayed messages, and
// the new head MEL state in a single database batch. This ensures crash-safe
// consistency: either all data from a processed block is persisted, or none is.
func (d *Database) SaveProcessedBlock(state *mel.State, batchMetas []*mel.BatchMetadata, delayedMessages []*mel.DelayedInboxMessage) error {
	dbBatch := d.db.NewBatch()
	if err := putBatchMetas(dbBatch, state, batchMetas); err != nil {
		return fmt.Errorf("SaveProcessedBlock: %w", err)
	}
	if err := putDelayedMessages(dbBatch, state, delayedMessages); err != nil {
		return fmt.Errorf("SaveProcessedBlock: %w", err)
	}
	if err := d.setMelState(dbBatch, state.ParentChainBlockNumber, *state); err != nil {
		return fmt.Errorf("SaveProcessedBlock: writing MEL state at block %d: %w", state.ParentChainBlockNumber, err)
	}
	if err := d.setHeadMelStateBlockNum(dbBatch, state.ParentChainBlockNumber); err != nil {
		return fmt.Errorf("SaveProcessedBlock: writing head block num %d: %w", state.ParentChainBlockNumber, err)
	}
	if err := dbBatch.Write(); err != nil {
		return fmt.Errorf("SaveProcessedBlock: committing batch for block %d: %w", state.ParentChainBlockNumber, err)
	}
	return nil
}

func (d *Database) setMelState(batch ethdb.KeyValueWriter, parentChainBlockNumber uint64, state mel.State) error {
	key := read.Key(schema.MelStatePrefix, parentChainBlockNumber)
	melStateBytes, err := rlp.EncodeToBytes(state)
	if err != nil {
		return err
	}
	return batch.Put(key, melStateBytes)
}

func (d *Database) setHeadMelStateBlockNum(batch ethdb.KeyValueWriter, parentChainBlockNumber uint64) error {
	parentChainBlockNumberBytes, err := rlp.EncodeToBytes(parentChainBlockNumber)
	if err != nil {
		return err
	}
	return batch.Put(schema.HeadMelStateBlockNumKey, parentChainBlockNumberBytes)
}

func (d *Database) GetHeadMelStateBlockNum() (uint64, error) {
	return read.Value[uint64](d.db, schema.HeadMelStateBlockNumKey)
}

func (d *Database) InitialBlockNum() (uint64, bool) {
	b := d.boundary.Load()
	if b == nil {
		return 0, false
	}
	return b.blockNum, true
}

// RewriteHeadBlockNum overwrites the head MEL state block number pointer.
// Used by ReorgTo to rewind the head without a full state save.
// Returns an error if no MEL state exists at the target block.
func (d *Database) RewriteHeadBlockNum(parentChainBlockNumber uint64) error {
	if _, err := d.State(parentChainBlockNumber); err != nil {
		return fmt.Errorf("cannot reorg to block %d: no MEL state found: %w", parentChainBlockNumber, err)
	}
	dbBatch := d.db.NewBatch()
	if err := d.setHeadMelStateBlockNum(dbBatch, parentChainBlockNumber); err != nil {
		return fmt.Errorf("RewriteHeadBlockNum: encoding head block num %d: %w", parentChainBlockNumber, err)
	}
	if err := dbBatch.Write(); err != nil {
		return fmt.Errorf("RewriteHeadBlockNum: committing batch for block %d: %w", parentChainBlockNumber, err)
	}
	return nil
}

func (d *Database) State(parentChainBlockNumber uint64) (*mel.State, error) {
	state, err := read.Value[mel.State](d.db, read.Key(schema.MelStatePrefix, parentChainBlockNumber))
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// StateAtOrBelowHead returns the MEL state for a given block, but only if the
// block is at or below the current head. This prevents callers from reading stale
// MEL state entries that may remain in the DB above the head after a reorg.
// Returns an error if the block is above the head (descriptive message) or if no
// state exists at that block number (DB not-found error).
func (d *Database) StateAtOrBelowHead(parentChainBlockNumber uint64) (*mel.State, error) {
	headBlockNum, err := d.GetHeadMelStateBlockNum()
	if err != nil {
		return nil, fmt.Errorf("StateAtOrBelowHead: failed to read head block num: %w", err)
	}
	if parentChainBlockNumber > headBlockNum {
		return nil, fmt.Errorf("requested MEL state at block %d is above current head %d (possible stale data after reorg)", parentChainBlockNumber, headBlockNum)
	}
	return d.State(parentChainBlockNumber)
}

// saveBatchMetas is a test-only helper. Production code uses SaveProcessedBlock for atomic writes.
func (d *Database) saveBatchMetas(state *mel.State, batchMetas []*mel.BatchMetadata) error {
	dbBatch := d.db.NewBatch()
	if err := putBatchMetas(dbBatch, state, batchMetas); err != nil {
		return err
	}
	return dbBatch.Write()
}

func (d *Database) fetchBatchMetadata(seqNum uint64) (*mel.BatchMetadata, error) {
	if b := d.boundary.Load(); b != nil && seqNum < b.batchCount {
		return legacyFetchBatchMetadata(d.db, seqNum)
	}
	batchMetadata, err := read.Value[mel.BatchMetadata](d.db, read.Key(schema.MelSequencerBatchMetaPrefix, seqNum))
	if err != nil {
		return nil, err
	}
	return &batchMetadata, nil
}

// saveDelayedMessages is a test-only helper. Production code uses SaveProcessedBlock for atomic writes.
func (d *Database) saveDelayedMessages(state *mel.State, delayedMessages []*mel.DelayedInboxMessage) error {
	dbBatch := d.db.NewBatch()
	if err := putDelayedMessages(dbBatch, state, delayedMessages); err != nil {
		return err
	}
	return dbBatch.Write()
}

// ReadDelayedMessage reads a delayed message by index from the database.
// It pops from the outbox accumulator (pouring inbox first if needed)
// and verifies the fetched message's hash matches the expected hash.
func (d *Database) ReadDelayedMessage(state *mel.State, index uint64) (*mel.DelayedInboxMessage, error) {
	expectedMsgHash, err := state.PopDelayedOutbox()
	if err != nil {
		return nil, fmt.Errorf("error popping delayed outbox for index %d: %w", index, err)
	}
	// The init message (index 0) is accumulated and read in the same block,
	// so it may not be in the DB yet. Use the in-memory copy from state.
	var delayed *mel.DelayedInboxMessage
	if index == 0 && state.InitMsg() != nil {
		delayed = state.InitMsg()
	} else {
		delayed, err = d.FetchDelayedMessage(index)
		if err != nil {
			return nil, err
		}
	}
	delayedBytes, err := rlp.EncodeToBytes(delayed)
	if err != nil {
		return nil, err
	}
	actualHash := crypto.Keccak256Hash(delayedBytes)
	if actualHash != expectedMsgHash {
		return nil, fmt.Errorf("delayed message hash mismatch at index %d: expected %s, got %s", index, expectedMsgHash.Hex(), actualHash.Hex())
	}
	return delayed, nil
}

func (d *Database) FetchDelayedMessage(index uint64) (*mel.DelayedInboxMessage, error) {
	if b := d.boundary.Load(); b != nil && index < b.delayedCount {
		return legacyFetchDelayedMessage(d.db, index)
	}
	delayed, err := read.Value[mel.DelayedInboxMessage](d.db, read.Key(schema.MelDelayedMessagePrefix, index))
	if err != nil {
		return nil, err
	}
	return &delayed, nil
}
