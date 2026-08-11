// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"sync/atomic"

	"github.com/offchainlabs/nitro/arbnode/mel"
)

// testIngestLimit caps which sequencer batches this process will ingest: a
// batch is only added when its SequenceNumber is below the limit. A negative
// value (the default) disables the cap entirely, so production behaviour is
// unchanged.
//
// It exists so tests can hold a node partway through catching up on the parent
// chain inbox, which is otherwise a timing race that only shows up on loaded
// machines. See SetTestIngestLimit.
var testIngestLimit atomic.Int64

func init() {
	testIngestLimit.Store(-1)
}

func SetTestIngestLimit(limit int64) {
	testIngestLimit.Store(limit)
}

// truncateBatchesForTest applies the test ingestion cap to a run of batches,
// which AddSequencerBatches requires to be contiguous and ascending.
func truncateBatchesForTest(batches []*mel.SequencerInboxBatch) []*mel.SequencerInboxBatch {
	limit := testIngestLimit.Load()
	if limit < 0 {
		return batches
	}
	// #nosec G115 -- limit is non-negative here.
	for i, batch := range batches {
		if batch.SequenceNumber >= uint64(limit) {
			return batches[:i]
		}
	}
	return batches
}
