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

func TestBlockRecordingsPrunerThrottlesByMinPruneInterval(t *testing.T) {
	config := DefaultBlockRecordingsPrunerConfig
	config.MinPruneInterval = time.Hour
	pruner, recorder := newTestBlockRecordingsPruner(config)
	pruner.Start(context.Background())
	defer pruner.StopAndWait()

	pruner.UpdateLatestConfirmed(1, validator.GoGlobalState{})
	for start := time.Now(); len(recorder.prunedCalls()) == 0; {
		if time.Since(start) > 10*time.Second {
			t.Fatal("timed out waiting for first prune")
		}
		time.Sleep(10 * time.Millisecond)
	}
	pruner.UpdateLatestConfirmed(2, validator.GoGlobalState{})

	pruner.StopAndWait()
	pruned := recorder.prunedCalls()
	if len(pruned) != 1 || pruned[0] != 1 {
		t.Fatalf("expected only the first prune within the prune interval, got %v", pruned)
	}
}

func TestBlockRecordingsPrunerPrunesRepeatedly(t *testing.T) {
	config := DefaultBlockRecordingsPrunerConfig
	config.MinPruneInterval = 0
	pruner, recorder := newTestBlockRecordingsPruner(config)
	pruner.Start(context.Background())
	defer pruner.StopAndWait()

	for pos := arbutil.MessageIndex(1); pos <= 5; pos++ {
		for start := time.Now(); ; {
			pruner.UpdateLatestConfirmed(pos, validator.GoGlobalState{})
			pruned := recorder.prunedCalls()
			if len(pruned) > 0 && pruned[len(pruned)-1] == pos {
				break
			}
			if time.Since(start) > 10*time.Second {
				t.Fatalf("timed out waiting for prune at %d, got %v", pos, pruned)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	pruner.StopAndWait()
	pruned := recorder.prunedCalls()
	seen := make(map[arbutil.MessageIndex]bool)
	for i, p := range pruned {
		if i > 0 && p < pruned[i-1] {
			t.Fatalf("expected prune positions to be non-decreasing, got %v", pruned)
		}
		seen[p] = true
	}
	for pos := arbutil.MessageIndex(1); pos <= 5; pos++ {
		if !seen[pos] {
			t.Fatalf("expected a prune at %d, got %v", pos, pruned)
		}
	}
}
