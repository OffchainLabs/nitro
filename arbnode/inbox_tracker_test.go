package arbnode

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/offchainlabs/nitro/util/containers"
)

func TestDeleteBatchMetadata(t *testing.T) {
	testBytes := []byte("bloop")

	tracker := &InboxTracker{
		db:        rawdb.NewMemoryDatabase(),
		batchMeta: containers.NewLruCache[uint64, BatchMetadata](100),
	}

	for i := uint64(0); i < 30; i += 1 {
		err := tracker.db.Put(dbKey(sequencerBatchMetaPrefix, i), testBytes)
		Require(t, err)
		if i%5 != 0 {
			tracker.batchMeta.Add(i, BatchMetadata{})
		}
	}

	batch := tracker.db.NewBatch()
	err := tracker.deleteBatchMetadataStartingAt(batch, 15)
	if err != nil {
		Fail(t, "deleteBatchMetadataStartingAt returned error: ", err)
	}
	err = batch.Write()
	Require(t, err)

	for i := uint64(0); i < 15; i += 1 {
		has, err := tracker.db.Has(dbKey(sequencerBatchMetaPrefix, i))
		Require(t, err)
		if !has {
			Fail(t, "value removed from db: ", i)
		}
		if i%5 != 0 {
			if !tracker.batchMeta.Contains(i) {
				Fail(t, "value removed from cache: ", i)
			}
		}
	}

	for i := uint64(15); i < 30; i += 1 {
		has, err := tracker.db.Has(dbKey(sequencerBatchMetaPrefix, i))
		Require(t, err)
		if has {
			Fail(t, "value not removed from db: ", i)
		}
		if tracker.batchMeta.Contains(i) {
			Fail(t, "value removed from cache: ", i)
		}
	}

}
