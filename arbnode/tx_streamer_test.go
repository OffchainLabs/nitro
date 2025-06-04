package arbnode

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/rlp"

	dbschema "github.com/offchainlabs/nitro/arbnode/db-schema"
	"github.com/offchainlabs/nitro/arbutil"
)

func TestTimeboostBackfillingsTrackersForMissingBlockMetadata(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messageCount := uint64(20)

	// Create arbDB with fragmented blockMetadata across blocks
	arbDb := rawdb.NewMemoryDatabase()
	countBytes, err := rlp.EncodeToBytes(messageCount)
	Require(t, err)
	Require(t, arbDb.Put(dbschema.MessageCountKey, countBytes))
	addKeys := func(start, end uint64, prefix []byte) {
		for i := start; i <= end; i++ {
			Require(t, arbDb.Put(dbKey(prefix, i), []byte{}))
		}
	}
	// 12, 13, 14, 18 have block metadata
	addKeys(12, 14, dbschema.BlockMetadataInputFeedPrefix)
	addKeys(18, 18, dbschema.BlockMetadataInputFeedPrefix)
	// 15, 16, 17, 19 are missing
	addKeys(15, 17, dbschema.MissingBlockMetadataInputFeedPrefix)
	addKeys(19, 19, dbschema.MissingBlockMetadataInputFeedPrefix)

	// Create tx streamer
	txStreamer := &TransactionStreamer{db: arbDb}
	txStreamer.StopWaiter.Start(ctx, txStreamer)

	backfillAndVerifyCorrectness := func(trackBlockMetadataFrom arbutil.MessageIndex, missingTrackers []uint64) {
		txStreamer.trackBlockMetadataFrom = trackBlockMetadataFrom
		txStreamer.backfillTrackersForMissingBlockMetadata(ctx)
		iter := arbDb.NewIterator([]byte("x"), nil)
		pos := 0
		for iter.Next() {
			keyBytes := bytes.TrimPrefix(iter.Key(), dbschema.MissingBlockMetadataInputFeedPrefix)
			if binary.BigEndian.Uint64(keyBytes) != missingTrackers[pos] {
				t.Fatalf("unexpected presence of blockMetadata. msgSeqNum: %d, expectedMsgSeqNum: %d", binary.BigEndian.Uint64(keyBytes), missingTrackers[pos])
			}
			pos++
		}
		if pos != len(missingTrackers) {
			t.Fatalf("number of keys with dbschema.BlockMetadataInputFeedPrefix doesn't match expected value. Want: %d, Got: %d", len(missingTrackers), pos)
		}
		iter.Release()
	}

	// Backfill trackers for missing data and verify that 10, 11 get added to already existing 16, 17, 18, 19 keys
	backfillAndVerifyCorrectness(10, []uint64{10, 11, 15, 16, 17, 19})

	// Backfill trackers for missing data and verify that 5, 6, 7, 8, 9 get added to already existing 10, 11, 16, 17, 18, 19 keys
	backfillAndVerifyCorrectness(5, []uint64{5, 6, 7, 8, 9, 10, 11, 15, 16, 17, 19})
}
