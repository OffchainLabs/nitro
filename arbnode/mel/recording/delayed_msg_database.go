// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrecording

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/db/read"
	"github.com/offchainlabs/nitro/arbnode/db/schema"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

// DelayedMsgDatabase is used for recording of preimages relating to delayed messages
// needed for MEL validation. It reads delayed messages from the DB and records preimages
// so that the replay binary can read them via preimage resolution.
type DelayedMsgDatabase struct {
	db        ethdb.KeyValueStore
	preimages map[common.Hash][]byte
}

// NewDelayedMsgDatabase returns DelayedMsgDatabase that records preimages related
// to the delayed messages needed for MEL validation into the given preimages map
func NewDelayedMsgDatabase(db ethdb.KeyValueStore, preimages daprovider.PreimagesMap) (*DelayedMsgDatabase, error) {
	if preimages == nil {
		return nil, errors.New("preimages recording destination cannot be nil")
	}
	if _, ok := preimages[arbutil.Keccak256PreimageType]; !ok {
		preimages[arbutil.Keccak256PreimageType] = make(map[common.Hash][]byte)
	}
	return &DelayedMsgDatabase{
		db:        db,
		preimages: preimages[arbutil.Keccak256PreimageType],
	}, nil
}

// ReadDelayedMessage reads a delayed message by index. It pops from the outbox
// (pouring inbox first if needed via the state's internal preimage map) and records
// the message content preimage for replay.
func (r *DelayedMsgDatabase) ReadDelayedMessage(state *mel.State, index uint64) (*mel.DelayedInboxMessage, error) {
	expectedMsgHash, err := state.PopDelayedOutbox()
	if err != nil {
		return nil, fmt.Errorf("error popping delayed outbox for index %d: %w", index, err)
	}
	delayed, err := fetchDelayedMessage(r.db, index)
	if err != nil {
		return nil, err
	}
	delayedBytes, err := rlp.EncodeToBytes(delayed)
	if err != nil {
		return nil, err
	}
	actualHash := crypto.Keccak256Hash(delayedBytes)
	if actualHash != expectedMsgHash {
		return nil, fmt.Errorf("delayed message hash mismatch at index %d: expected %s, got %s", index, expectedMsgHash.Hex(), actualHash.Hex())
	}
	// Record message content preimage for replay
	r.preimages[actualHash] = delayedBytes
	return delayed, nil
}

func fetchDelayedMessage(db ethdb.KeyValueStore, index uint64) (*mel.DelayedInboxMessage, error) {
	delayed, err := read.Value[mel.DelayedInboxMessage](db, read.Key(schema.MelDelayedMessagePrefix, index))
	if err != nil {
		return nil, err
	}
	return &delayed, nil
}
