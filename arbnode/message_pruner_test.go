// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

func TestMessagePrunerWithPruningEligibleMessagePresent(t *testing.T) {
	endBatchCount := uint64(2 * 100 * 1024)
	endBatchMetadata := BatchMetadata{
		MessageCount:        2 * 100 * 1024,
		DelayedMessageCount: 2 * 100 * 1024,
	}
	inboxTrackerDb, transactionStreamerDb := setupDatabase(t, endBatchCount, endBatchMetadata)
	deleteOldMessageFromDB(endBatchCount, endBatchMetadata, inboxTrackerDb, transactionStreamerDb)

	checkDbKeys(t, endBatchCount, inboxTrackerDb, sequencerBatchMetaPrefix)
	checkDbKeys(t, uint64(endBatchMetadata.MessageCount), transactionStreamerDb, messagePrefix)
	checkDbKeys(t, endBatchMetadata.DelayedMessageCount, inboxTrackerDb, rlpDelayedMessagePrefix)

}

func TestMessagePrunerWithNoPruningEligibleMessagePresent(t *testing.T) {
	endBatchCount := uint64(2)
	endBatchMetadata := BatchMetadata{
		MessageCount:        2,
		DelayedMessageCount: 2,
	}
	inboxTrackerDb, transactionStreamerDb := setupDatabase(t, endBatchCount, endBatchMetadata)
	deleteOldMessageFromDB(endBatchCount, endBatchMetadata, inboxTrackerDb, transactionStreamerDb)

	checkDbKeys(t, endBatchCount, inboxTrackerDb, sequencerBatchMetaPrefix)
	checkDbKeys(t, uint64(endBatchMetadata.MessageCount), transactionStreamerDb, messagePrefix)
	checkDbKeys(t, endBatchMetadata.DelayedMessageCount, inboxTrackerDb, rlpDelayedMessagePrefix)

}

func setupDatabase(t *testing.T, endBatchCount uint64, endBatchMetadata BatchMetadata) (ethdb.Database, ethdb.Database) {
	inboxTrackerDb := rawdb.NewMemoryDatabase()
	for i := uint64(0); i < endBatchCount; i++ {
		err := inboxTrackerDb.Put(dbKey(sequencerBatchMetaPrefix, i), []byte{})
		Require(t, err)
	}

	transactionStreamerDb := rawdb.NewMemoryDatabase()
	for i := uint64(0); i < uint64(endBatchMetadata.MessageCount); i++ {
		err := transactionStreamerDb.Put(dbKey(messagePrefix, i), []byte{})
		Require(t, err)
	}

	for i := uint64(0); i < endBatchMetadata.DelayedMessageCount; i++ {
		err := inboxTrackerDb.Put(dbKey(rlpDelayedMessagePrefix, i), []byte{})
		Require(t, err)
	}

	return inboxTrackerDb, transactionStreamerDb
}

func checkDbKeys(t *testing.T, endCount uint64, db ethdb.Database, prefix []byte) {
	for i := uint64(0); i < endCount; i++ {
		hasKey, err := db.Has(dbKey(prefix, i))
		Require(t, err)
		if i == 0 || i == endCount-1 {
			if !hasKey {
				Fail(t, "Key", i, "with prefix", string(prefix), "should be present after pruning")
			}
		} else {
			if hasKey {
				Fail(t, "Key", i, "with prefix", string(prefix), "should not be present after pruning")
			}
		}
	}
}
