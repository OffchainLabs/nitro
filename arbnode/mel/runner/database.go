// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrunner

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/db/read"
	"github.com/offchainlabs/nitro/arbnode/db/schema"
	"github.com/offchainlabs/nitro/arbnode/mel"
)

// Database holds an ethdb.KeyValueStore underneath and implements reading of
// delayed messages in native mode by verifying the read delayed message
// against the outbox accumulator.
type Database struct {
	db ethdb.KeyValueStore

	// Cached boundary counts from the initial MEL state (set during legacy migration).
	// Indices below these thresholds are read from legacy (pre-MEL) schema keys.
	hasInitialState     bool
	initialDelayedCount uint64
	initialBatchCount   uint64
}

func NewDatabase(db ethdb.KeyValueStore) (*Database, error) {
	d := &Database{db: db}
	if err := d.loadInitialBoundary(); err != nil {
		return nil, fmt.Errorf("failed to load initial MEL state boundary: %w", err)
	}
	return d, nil
}

// loadInitialBoundary loads the initial MEL state boundary counts from the DB.
// If InitialMelStateBlockNumKey exists, it reads the initial state to cache
// the delayed message and batch count thresholds for legacy key dispatch.
// Returns nil if the key does not exist (fresh node with no legacy boundary).
func (d *Database) loadInitialBoundary() error {
	blockNum, err := read.Value[uint64](d.db, schema.InitialMelStateBlockNumKey)
	if err != nil {
		if rawdb.IsDbErrNotFound(err) {
			d.hasInitialState = false
			return nil
		}
		return fmt.Errorf("error reading InitialMelStateBlockNumKey: %w", err)
	}
	state, err := d.State(blockNum)
	if err != nil {
		return fmt.Errorf("error reading initial MEL state at block %d: %w", blockNum, err)
	}
	d.hasInitialState = true
	d.initialDelayedCount = state.DelayedMessagesSeen
	d.initialBatchCount = state.BatchCount
	return nil
}

func (d *Database) SaveInitialMelState(initialState *mel.State) error {
	dbBatch := d.db.NewBatch()
	encoded, err := rlp.EncodeToBytes(initialState.ParentChainBlockNumber)
	if err != nil {
		return err
	}
	if err := dbBatch.Put(schema.InitialMelStateBlockNumKey, encoded); err != nil {
		return err
	}
	if err := d.setMelState(dbBatch, initialState.ParentChainBlockNumber, *initialState); err != nil {
		return err
	}
	if err := d.setHeadMelStateBlockNum(dbBatch, initialState.ParentChainBlockNumber); err != nil {
		return err
	}
	d.hasInitialState = true
	d.initialDelayedCount = initialState.DelayedMessagesSeen
	d.initialBatchCount = initialState.BatchCount
	return dbBatch.Write()
}

func (d *Database) GetHeadMelState() (*mel.State, error) {
	headMelStateBlockNum, err := d.GetHeadMelStateBlockNum()
	if err != nil {
		return nil, fmt.Errorf("error getting HeadMelStateBlockNum from database: %w", err)
	}
	return d.State(headMelStateBlockNum)
}

// SaveState should exclusively be called for saving the recently generated "head" MEL state
func (d *Database) SaveState(state *mel.State) error {
	dbBatch := d.db.NewBatch()
	if err := d.setMelState(dbBatch, state.ParentChainBlockNumber, *state); err != nil {
		return err
	}
	if err := d.setHeadMelStateBlockNum(dbBatch, state.ParentChainBlockNumber); err != nil {
		return err
	}
	return dbBatch.Write()
}

func (d *Database) setMelState(batch ethdb.KeyValueWriter, parentChainBlockNumber uint64, state mel.State) error {
	key := read.Key(schema.MelStatePrefix, parentChainBlockNumber)
	melStateBytes, err := rlp.EncodeToBytes(state)
	if err != nil {
		return err
	}
	melStateSizeBytesGauge.Update(int64(len(melStateBytes)))
	if err := batch.Put(key, melStateBytes); err != nil {
		return err
	}
	return nil
}

func (d *Database) setHeadMelStateBlockNum(batch ethdb.KeyValueWriter, parentChainBlockNumber uint64) error {
	parentChainBlockNumberBytes, err := rlp.EncodeToBytes(parentChainBlockNumber)
	if err != nil {
		return err
	}
	err = batch.Put(schema.HeadMelStateBlockNumKey, parentChainBlockNumberBytes)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) GetHeadMelStateBlockNum() (uint64, error) {
	return read.Value[uint64](d.db, schema.HeadMelStateBlockNumKey)
}

func (d *Database) State(parentChainBlockNumber uint64) (*mel.State, error) {
	state, err := read.Value[mel.State](d.db, read.Key(schema.MelStatePrefix, parentChainBlockNumber))
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (d *Database) SaveBatchMetas(state *mel.State, batchMetas []*mel.BatchMetadata) error {
	dbBatch := d.db.NewBatch()
	if state.BatchCount < uint64(len(batchMetas)) {
		return fmt.Errorf("mel state's BatchCount: %d is lower than number of batchMetadata: %d queued to be added", state.BatchCount, len(batchMetas))
	}
	firstPos := state.BatchCount - uint64(len(batchMetas))
	for i, batchMetadata := range batchMetas {
		key := read.Key(schema.MelSequencerBatchMetaPrefix, firstPos+uint64(i)) // #nosec G115
		batchMetadataBytes, err := rlp.EncodeToBytes(*batchMetadata)
		if err != nil {
			return err
		}
		err = dbBatch.Put(key, batchMetadataBytes)
		if err != nil {
			return err
		}

	}
	return dbBatch.Write()
}

func (d *Database) fetchBatchMetadata(seqNum uint64) (*mel.BatchMetadata, error) {
	if d.hasInitialState && seqNum < d.initialBatchCount {
		return legacyFetchBatchMetadata(d.db, seqNum)
	}
	batchMetadata, err := read.Value[mel.BatchMetadata](d.db, read.Key(schema.MelSequencerBatchMetaPrefix, seqNum))
	if err != nil {
		return nil, err
	}
	return &batchMetadata, nil
}

func (d *Database) SaveDelayedMessages(state *mel.State, delayedMessages []*mel.DelayedInboxMessage) error {
	dbBatch := d.db.NewBatch()
	if state.DelayedMessagesSeen < uint64(len(delayedMessages)) {
		return fmt.Errorf("mel state's DelayedMessagesSeen: %d is lower than number of delayed messages: %d queued to be added", state.DelayedMessagesSeen, len(delayedMessages))
	}
	firstPos := state.DelayedMessagesSeen - uint64(len(delayedMessages))
	for i, msg := range delayedMessages {
		key := read.Key(schema.MelDelayedMessagePrefix, firstPos+uint64(i)) // #nosec G115
		delayedBytes, err := rlp.EncodeToBytes(*msg)
		if err != nil {
			return err
		}
		err = dbBatch.Put(key, delayedBytes)
		if err != nil {
			return err
		}

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
	if d.hasInitialState && index < d.initialDelayedCount {
		return legacyFetchDelayedMessage(d.db, index)
	}
	delayed, err := read.Value[mel.DelayedInboxMessage](d.db, read.Key(schema.MelDelayedMessagePrefix, index))
	if err != nil {
		return nil, err
	}
	return &delayed, nil
}
