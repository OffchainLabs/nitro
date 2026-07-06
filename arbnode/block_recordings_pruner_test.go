// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/validator"
)

type pruningTestRecorder struct {
	mutex  sync.Mutex
	pruned []arbutil.MessageIndex
}

func (r *pruningTestRecorder) RecordBlockCreation(
	pos arbutil.MessageIndex,
	_ *arbostypes.MessageWithMetadata,
	_ []rawdb.WasmTarget,
) containers.PromiseInterface[*execution.RecordResult] {
	return containers.NewReadyPromise(&execution.RecordResult{Pos: pos}, nil)
}

func (r *pruningTestRecorder) PrepareForRecord(_, _ arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	return containers.NewReadyPromise(struct{}{}, nil)
}

func (r *pruningTestRecorder) PruneBlockRecordings(before arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.pruned = append(r.pruned, before)
	return containers.NewReadyPromise(struct{}{}, nil)
}

func (r *pruningTestRecorder) prunedCalls() []arbutil.MessageIndex {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return append([]arbutil.MessageIndex{}, r.pruned...)
}

func newTestBlockRecordingsPruner(config BlockRecordingsPrunerConfig) (*BlockRecordingsPruner, *pruningTestRecorder) {
	recorder := &pruningTestRecorder{}
	pruner := NewBlockRecordingsPruner(recorder, func() *BlockRecordingsPrunerConfig { return &config })
	return pruner, recorder
}

func TestBlockRecordingsPrunerPrunesBelowLatestConfirmed(t *testing.T) {
	pruner, recorder := newTestBlockRecordingsPruner(DefaultBlockRecordingsPrunerConfig)
	pruner.Start(context.Background())
	defer pruner.StopAndWait()

	pruner.UpdateLatestConfirmed(42, validator.GoGlobalState{})

	pruner.StopAndWait()
	pruned := recorder.prunedCalls()
	if len(pruned) != 1 || pruned[0] != 42 {
		t.Fatalf("expected pruning before message 42, got %v", pruned)
	}
}

func TestBlockRecordingsPrunerThrottlesByPruneInterval(t *testing.T) {
	config := DefaultBlockRecordingsPrunerConfig
	config.PruneInterval = time.Hour
	pruner, recorder := newTestBlockRecordingsPruner(config)
	pruner.Start(context.Background())
	defer pruner.StopAndWait()

	pruner.UpdateLatestConfirmed(1, validator.GoGlobalState{})
	for start := time.Now(); len(recorder.prunedCalls()) == 0; {
		if time.Since(start) > 10*time.Second {
			t.Fatal("timed out waiting for first prune")
		}
		time.Sleep(time.Millisecond)
	}
	pruner.UpdateLatestConfirmed(2, validator.GoGlobalState{})

	pruner.StopAndWait()
	pruned := recorder.prunedCalls()
	if len(pruned) != 1 || pruned[0] != 1 {
		t.Fatalf("expected only the first prune within the prune interval, got %v", pruned)
	}
}
