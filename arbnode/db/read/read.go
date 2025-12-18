package read

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/db/schema"
	"github.com/offchainlabs/nitro/arbnode/mel"
)

// MELSequencerBatchCount returns the batch count corresponding to the head MEL state
func MELSequencerBatchCount(db ethdb.KeyValueStore) (uint64, error) {
	headStateBlockNum, err := Value[uint64](db, schema.HeadMelStateBlockNumKey)
	if err != nil {
		return 0, err
	}
	headState, err := Value[mel.State](db, Key(schema.MelStatePrefix, headStateBlockNum))
	if err != nil {
		return 0, err
	}
	return headState.BatchCount, nil
}

// SequencerBatchCount returns the pre-MEL sequencer batch count
func SequencerBatchCount(db ethdb.KeyValueStore) (uint64, error) {
	return Value[uint64](db, schema.SequencerBatchCountKey)
}

// MELBatchMetadata returns the BatchMetadata corresponding to the given batch sequence number
func MELBatchMetadata(db ethdb.KeyValueStore, seqNum uint64) (mel.BatchMetadata, error) {
	return Value[mel.BatchMetadata](db, Key(schema.MelSequencerBatchMetaPrefix, seqNum))
}

// BatchMetadata returns the pre-MEL BatchMetadata corresponding to the given batch sequence number
func BatchMetadata(db ethdb.KeyValueStore, seqNum uint64) (mel.BatchMetadata, error) {
	return Value[mel.BatchMetadata](db, Key(schema.SequencerBatchMetaPrefix, seqNum))
}

// Key returns appropriate database key for a given prefix and position, prefix generally picked
// from the db schema available at arbnode/db/schema/schema.go
func Key(prefix []byte, pos uint64) []byte {
	var key []byte
	key = append(key, prefix...)
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, pos)
	key = append(key, data...)
	return key
}

// Value given a ethdb KeyValueStore and a key returns the stored value corresponding to that key
func Value[T any](db ethdb.KeyValueStore, key []byte) (T, error) {
	var empty T
	data, err := db.Get(key)
	if err != nil {
		return empty, err
	}
	var val T
	err = rlp.DecodeBytes(data, &val)
	if err != nil {
		return empty, err
	}
	return val, nil
}
