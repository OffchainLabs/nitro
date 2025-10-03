package arbnode

import (
	"bytes"
	"context"
	"encoding/binary"
	"log/slog"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/util/testhelpers"
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
	Require(t, arbDb.Put(messageCountKey, countBytes))
	addKeys := func(start, end uint64, prefix []byte) {
		for i := start; i <= end; i++ {
			Require(t, arbDb.Put(dbKey(prefix, i), []byte{}))
		}
	}
	// 12, 13, 14, 18 have block metadata
	addKeys(12, 14, blockMetadataInputFeedPrefix)
	addKeys(18, 18, blockMetadataInputFeedPrefix)
	// 15, 16, 17, 19 are missing
	addKeys(15, 17, missingBlockMetadataInputFeedPrefix)
	addKeys(19, 19, missingBlockMetadataInputFeedPrefix)

	// Create tx streamer
	txStreamer := &TransactionStreamer{db: arbDb}
	txStreamer.StopWaiter.Start(ctx, txStreamer)

	backfillAndVerifyCorrectness := func(trackBlockMetadataFrom arbutil.MessageIndex, missingTrackers []uint64) {
		txStreamer.trackBlockMetadataFrom = trackBlockMetadataFrom
		txStreamer.backfillTrackersForMissingBlockMetadata(ctx)
		iter := arbDb.NewIterator([]byte("x"), nil)
		pos := 0
		for iter.Next() {
			keyBytes := bytes.TrimPrefix(iter.Key(), missingBlockMetadataInputFeedPrefix)
			if binary.BigEndian.Uint64(keyBytes) != missingTrackers[pos] {
				t.Fatalf("unexpected presence of blockMetadata. msgSeqNum: %d, expectedMsgSeqNum: %d", binary.BigEndian.Uint64(keyBytes), missingTrackers[pos])
			}
			pos++
		}
		if pos != len(missingTrackers) {
			t.Fatalf("number of keys with blockMetadataInputFeedPrefix doesn't match expected value. Want: %d, Got: %d", len(missingTrackers), pos)
		}
		iter.Release()
	}

	// Backfill trackers for missing data and verify that 10, 11 get added to already existing 16, 17, 18, 19 keys
	backfillAndVerifyCorrectness(10, []uint64{10, 11, 15, 16, 17, 19})

	// Backfill trackers for missing data and verify that 5, 6, 7, 8, 9 get added to already existing 10, 11, 16, 17, 18, 19 keys
	backfillAndVerifyCorrectness(5, []uint64{5, 6, 7, 8, 9, 10, 11, 15, 16, 17, 19})
}

func TestLoggingMinRequiredNodeVersionMessage(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LvlTrace)

	txStreamer := &TransactionStreamer{}
	feedMessage := &message.BroadcastFeedMessage{}
	currentNodeVersion := chaininfo.NITRO_NODE_VERSION
	msg := "Node version is below the required minimum nitro version"

	minNitroVersions := [3]int{currentNodeVersion + 1, currentNodeVersion, currentNodeVersion} // Info log should be printed
	feedMessage.MinNitroVersions = &minNitroVersions
	Require(t, txStreamer.logMinRequiredNodeVersionMessage(feedMessage))
	if !logHandler.WasLoggedAtLevel(msg, slog.LevelInfo) || logHandler.WasLoggedAtLevel(msg, slog.LevelWarn) || logHandler.WasLoggedAtLevel(msg, slog.LevelError) { // Only Info shoul be printed
		t.Fatal("minimum required node version message was not logged at level Info")
	}

	atomicTimeWrite(&txStreamer.lastMinNitroVersionMsgLoggedTimestamp, time.Time{})
	minNitroVersions[1] += 2 // Warn log should be printed
	Require(t, txStreamer.logMinRequiredNodeVersionMessage(feedMessage))
	if !logHandler.WasLoggedAtLevel(msg, slog.LevelWarn) || logHandler.WasLoggedAtLevel(msg, slog.LevelError) { // Error log shouldn't be printed yet
		t.Fatal("minimum required node version message was not logged at level Warn")
	}

	atomicTimeWrite(&txStreamer.lastMinNitroVersionMsgLoggedTimestamp, time.Time{})
	minNitroVersions[2] += 3 // Error log should be printed
	Require(t, txStreamer.logMinRequiredNodeVersionMessage(feedMessage))
	if !logHandler.WasLoggedAtLevel(msg, slog.LevelError) {
		t.Fatal("minimum required node version message was not logged at level Error")
	}
}
