// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/offchainlabs/nitro/arbutil"
)

func TestMessagePrunerWithPruningEligibleMessagePresent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagesCount := uint64(2 * 100 * 1024)
	inboxTrackerDb, transactionStreamerDb := setupDatabase(t, 2*100*1024, 2*100*1024)
	err := deleteOldMessageFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount, inboxTrackerDb, transactionStreamerDb)
	Require(t, err)

	checkDbKeys(t, messagesCount, transactionStreamerDb, messagePrefix)
	checkDbKeys(t, messagesCount, inboxTrackerDb, rlpDelayedMessagePrefix)

}

func TestMessagePrunerTraverseEachMessageOnlyOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagesCount := uint64(10)
	inboxTrackerDb, transactionStreamerDb := setupDatabase(t, messagesCount, messagesCount)
	// In first iteration message till messagesCount are tried to be deleted.
	err := deleteOldMessageFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount, inboxTrackerDb, transactionStreamerDb)
	Require(t, err)
	// After first iteration messagesCount/2 is reinserted in inbox db
	err = inboxTrackerDb.Put(dbKey(messagePrefix, messagesCount/2), []byte{})
	Require(t, err)
	// In second iteration message till messagesCount are again tried to be deleted.
	err = deleteOldMessageFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount, inboxTrackerDb, transactionStreamerDb)
	Require(t, err)
	// In second iteration all the message till messagesCount are deleted again.
	checkDbKeys(t, messagesCount, transactionStreamerDb, messagePrefix)
}

func TestMessagePrunerPruneTillLessThenEqualTo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagesCount := uint64(10)
	inboxTrackerDb, transactionStreamerDb := setupDatabase(t, 2*messagesCount, 20)
	err := inboxTrackerDb.Delete(dbKey(messagePrefix, 9))
	Require(t, err)
	err = deleteOldMessageFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount, inboxTrackerDb, transactionStreamerDb)
	Require(t, err)
	hasKey, err := transactionStreamerDb.Has(dbKey(messagePrefix, messagesCount))
	Require(t, err)
	if !hasKey {
		Fail(t, "Key", 10, "with prefix", string(messagePrefix), "should be present after pruning")
	}
}

func TestMessagePrunerWithNoPruningEligibleMessagePresent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagesCount := uint64(10)
	inboxTrackerDb, transactionStreamerDb := setupDatabase(t, messagesCount, messagesCount)
	err := deleteOldMessageFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount, inboxTrackerDb, transactionStreamerDb)
	Require(t, err)

	checkDbKeys(t, uint64(messagesCount), transactionStreamerDb, messagePrefix)
	checkDbKeys(t, messagesCount, inboxTrackerDb, rlpDelayedMessagePrefix)

}

func setupDatabase(t *testing.T, messageCount, delayedMessageCount uint64) (ethdb.Database, ethdb.Database) {

	transactionStreamerDb := rawdb.NewMemoryDatabase()
	for i := uint64(0); i < uint64(messageCount); i++ {
		err := transactionStreamerDb.Put(dbKey(messagePrefix, i), []byte{})
		Require(t, err)
	}

	inboxTrackerDb := rawdb.NewMemoryDatabase()
	for i := uint64(0); i < delayedMessageCount; i++ {
		err := inboxTrackerDb.Put(dbKey(rlpDelayedMessagePrefix, i), []byte{})
		Require(t, err)
	}

	return inboxTrackerDb, transactionStreamerDb
}

func checkDbKeys(t *testing.T, endCount uint64, db ethdb.Database, prefix []byte) {
	t.Helper()
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
