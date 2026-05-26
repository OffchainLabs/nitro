// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package staker

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/validator"
)

type pruningTestRecorder struct {
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
	r.pruned = append(r.pruned, before)
	return containers.NewReadyPromise(struct{}{}, nil)
}

type pruningTestInboxTracker struct{}

func (*pruningTestInboxTracker) SetBlockValidator(*BlockValidator) {}

func (*pruningTestInboxTracker) GetDelayedMessageBytes(context.Context, uint64) ([]byte, error) {
	return nil, errors.New("unexpected delayed message request")
}

func (*pruningTestInboxTracker) GetBatchMessageCount(seqNum uint64) (arbutil.MessageIndex, error) {
	if seqNum != 0 {
		return 0, errors.New("unexpected batch number")
	}
	return 1, nil
}

func (*pruningTestInboxTracker) GetBatchAcc(uint64) (common.Hash, error) {
	return common.Hash{}, nil
}

func (*pruningTestInboxTracker) GetBatchCount() (uint64, error) {
	return 1, nil
}

func (*pruningTestInboxTracker) FindInboxBatchContainingMessage(arbutil.MessageIndex) (uint64, bool, error) {
	return 0, true, nil
}

type pruningTestInboxReader struct{}

func (*pruningTestInboxReader) GetSequencerMessageBytes(context.Context, uint64) ([]byte, common.Hash, error) {
	return []byte{0}, common.Hash{}, nil
}

func (*pruningTestInboxReader) GetFinalizedMsgCount(context.Context) (arbutil.MessageIndex, error) {
	return 0, nil
}

type pruningTestStreamer struct {
	result execution.MessageResult
}

func (*pruningTestStreamer) SetBlockValidator(*BlockValidator) {}

func (*pruningTestStreamer) GetProcessedMessageCount() (arbutil.MessageIndex, error) {
	return 1, nil
}

func (*pruningTestStreamer) GetMessage(arbutil.MessageIndex) (*arbostypes.MessageWithMetadata, error) {
	msg := arbostypes.EmptyTestMessageWithMetadata
	return &msg, nil
}

func (s *pruningTestStreamer) ResultAtMessageIndex(arbutil.MessageIndex) (*execution.MessageResult, error) {
	return &s.result, nil
}

func (*pruningTestStreamer) PauseReorgs()  {}
func (*pruningTestStreamer) ResumeReorgs() {}

func (*pruningTestStreamer) ChainConfig() *params.ChainConfig {
	return params.TestChainConfig
}

type pruningTestValidationRun struct {
	containers.PromiseInterface[validator.GoGlobalState]
	wasmModuleRoot common.Hash
}

func (r *pruningTestValidationRun) WasmModuleRoot() common.Hash {
	return r.wasmModuleRoot
}

type pruningTestExecutionSpawner struct {
	wasmModuleRoot common.Hash
	result         validator.GoGlobalState
	err            error
}

func (s *pruningTestExecutionSpawner) Launch(*validator.ValidationInput, common.Hash) validator.ValidationRun {
	return &pruningTestValidationRun{
		PromiseInterface: containers.NewReadyPromise(s.result, s.err),
		wasmModuleRoot:   s.wasmModuleRoot,
	}
}

func (s *pruningTestExecutionSpawner) WasmModuleRoots() ([]common.Hash, error) {
	return []common.Hash{s.wasmModuleRoot}, nil
}

func (*pruningTestExecutionSpawner) Start(context.Context) error { return nil }
func (*pruningTestExecutionSpawner) Stop()                       {}
func (*pruningTestExecutionSpawner) Name() string                { return "pruning-test" }
func (*pruningTestExecutionSpawner) StylusArchs() []rawdb.WasmTarget {
	return nil
}
func (*pruningTestExecutionSpawner) Capacity() int { return 1 }
func (*pruningTestExecutionSpawner) CreateExecutionRun(common.Hash, *validator.ValidationInput, bool) containers.PromiseInterface[validator.ExecutionRun] {
	return containers.NewReadyPromise[validator.ExecutionRun](nil, nil)
}

func newPruningTestStatelessValidator(validationResult validator.GoGlobalState, validationErr error) (*StatelessBlockValidator, *pruningTestRecorder, common.Hash) {
	moduleRoot := common.BytesToHash([]byte{1})
	recordingResult := execution.MessageResult{
		BlockHash: common.BytesToHash([]byte{2}),
		SendRoot:  common.BytesToHash([]byte{3}),
	}
	recorder := &pruningTestRecorder{}
	validator := &StatelessBlockValidator{
		recorder:     recorder,
		inboxReader:  &pruningTestInboxReader{},
		inboxTracker: &pruningTestInboxTracker{},
		streamer: &pruningTestStreamer{
			result: recordingResult,
		},
		execSpawners: []validator.ExecutionSpawner{&pruningTestExecutionSpawner{
			wasmModuleRoot: moduleRoot,
			result:         validationResult,
			err:            validationErr,
		}},
	}
	return validator, recorder, moduleRoot
}

func TestValidateResultPrunesBlockRecordingsAfterSuccessfulValidation(t *testing.T) {
	validationResult := validator.GoGlobalState{
		BlockHash:  common.BytesToHash([]byte{2}),
		SendRoot:   common.BytesToHash([]byte{3}),
		Batch:      1,
		PosInBatch: 0,
	}
	validator, recorder, moduleRoot := newPruningTestStatelessValidator(validationResult, nil)

	valid, _, err := validator.ValidateResult(context.Background(), 0, false, moduleRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("expected validation to succeed")
	}
	if len(recorder.pruned) != 1 || recorder.pruned[0] != 1 {
		t.Fatalf("expected pruning before message 1, got %v", recorder.pruned)
	}
}

func TestValidateResultDoesNotPruneBlockRecordingsAfterFailedValidation(t *testing.T) {
	validator, recorder, moduleRoot := newPruningTestStatelessValidator(validator.GoGlobalState{}, errors.New("validation failed"))

	valid, _, err := validator.ValidateResult(context.Background(), 0, false, moduleRoot)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if valid {
		t.Fatal("expected validation to fail")
	}
	if len(recorder.pruned) != 0 {
		t.Fatalf("expected no pruning after failed validation, got %v", recorder.pruned)
	}
}
