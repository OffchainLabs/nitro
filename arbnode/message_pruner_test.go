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

func TestMessagePrunerTraverseEachMessageOnlyOnce(t *testing.T) {
	endBatchCount := uint64(10)
	endBatchMetadata := BatchMetadata{}
	inboxTrackerDb, transactionStreamerDb := setupDatabase(t, endBatchCount, endBatchMetadata)
	// In first iteration message till endBatchCount are tried to be deleted.
	deleteOldMessageFromDB(endBatchCount, endBatchMetadata, inboxTrackerDb, transactionStreamerDb)
	// In first iteration all the message till endBatchCount are deleted.
	checkDbKeys(t, endBatchCount, inboxTrackerDb, sequencerBatchMetaPrefix)
	// After first iteration endBatchCount/2 is reinserted in inbox db
	err := inboxTrackerDb.Put(dbKey(sequencerBatchMetaPrefix, endBatchCount/2), []byte{})
	Require(t, err)
	// In second iteration message till endBatchCount are again tried to be deleted.
	deleteOldMessageFromDB(endBatchCount, endBatchMetadata, inboxTrackerDb, transactionStreamerDb)
	// In second iteration message endBatchCount/2 is not deleted because it was reinserted after first iteration
	// and since each message is only traversed once during the pruning cycle and endBatchCount/2 was deleted in first
	// iteration, it will not be deleted again.
	hasKey, err := inboxTrackerDb.Has(dbKey(sequencerBatchMetaPrefix, endBatchCount/2))
	Require(t, err)
	if !hasKey {
		Fail(t, "Key", endBatchCount/2, "with prefix", string(sequencerBatchMetaPrefix), "should be present after pruning")
	}
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
