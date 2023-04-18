// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package staker

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/validator/server_api"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/validator"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/pkg/errors"
)

type StatelessBlockValidator struct {
	config *BlockValidatorConfig

	execSpawner        validator.ExecutionSpawner
	validationSpawners []validator.ValidationSpawner

	recorder execution.ExecutionRecorder

	inboxReader  InboxReaderInterface
	inboxTracker InboxTrackerInterface
	streamer     TransactionStreamerInterface
	db           ethdb.Database
	daService    arbstate.DataAvailabilityReader

	moduleMutex           sync.Mutex
	currentWasmModuleRoot common.Hash
	pendingWasmModuleRoot common.Hash
}

type BlockValidatorRegistrer interface {
	SetBlockValidator(*BlockValidator)
}

type InboxTrackerInterface interface {
	BlockValidatorRegistrer
	GetDelayedMessageBytes(uint64) ([]byte, error)
	GetBatchMessageCount(seqNum uint64) (arbutil.MessageIndex, error)
	GetBatchAcc(seqNum uint64) (common.Hash, error)
	GetBatchCount() (uint64, error)
	FindL1BatchForMessage(pos arbutil.MessageIndex) (uint64, error)
}

type TransactionStreamerInterface interface {
	BlockValidatorRegistrer
	GetProcessedMessageCount() (arbutil.MessageIndex, error)
	GetMessage(seqNum arbutil.MessageIndex) (*arbostypes.MessageWithMetadata, error)
	ResultAtCount(count arbutil.MessageIndex) (*execution.MessageResult, error)
	PauseReorgs()
	ResumeReorgs()
}

type InboxReaderInterface interface {
	GetSequencerMessageBytes(seqNum uint64) containers.PromiseInterface[[]byte]
}

type L1ReaderInterface interface {
	Client() arbutil.L1Interface
	Subscribe(bool) (<-chan *types.Header, func())
	WaitForTxApproval(tx *types.Transaction) containers.PromiseInterface[*types.Receipt]
}

type GlobalStatePosition struct {
	BatchNumber uint64
	PosInBatch  uint64
}

func GlobalStatePositionsAtCount(
	tracker InboxTrackerInterface,
	count arbutil.MessageIndex,
	batch uint64,
) (GlobalStatePosition, GlobalStatePosition, error) {
	msgCountInBatch, err := tracker.GetBatchMessageCount(batch)
	if err != nil {
		return GlobalStatePosition{}, GlobalStatePosition{}, err
	}
	var firstInBatch arbutil.MessageIndex
	if batch > 0 {
		firstInBatch, err = tracker.GetBatchMessageCount(batch - 1)
		if err != nil {
			return GlobalStatePosition{}, GlobalStatePosition{}, err
		}
	}
	if msgCountInBatch < count {
		return GlobalStatePosition{}, GlobalStatePosition{}, fmt.Errorf("batch %d has msgCount %d, failed getting for %d", batch, msgCountInBatch-1, count)
	}
	if firstInBatch >= count {
		return GlobalStatePosition{}, GlobalStatePosition{}, fmt.Errorf("batch %d starts from %d, failed getting for %d", batch, firstInBatch, count)
	}
	posInBatch := uint64(count - firstInBatch - 1)
	startPos := GlobalStatePosition{batch, posInBatch}
	if msgCountInBatch == count {
		return startPos, GlobalStatePosition{batch + 1, 0}, nil
	}
	return startPos, GlobalStatePosition{batch, posInBatch + 1}, nil
}

type ValidationEntryStage uint32

const (
	Empty ValidationEntryStage = iota
	ReadyForRecord
	Ready
)

type validationEntry struct {
	Stage ValidationEntryStage
	// Valid since ReadyforRecord:
	Pos           arbutil.MessageIndex
	Start         validator.GoGlobalState
	End           validator.GoGlobalState
	HasDelayedMsg bool
	DelayedMsgNr  uint64
	// valid when created, removed after recording
	msg *arbostypes.MessageWithMetadata
	// Has batch when created - others could be added on record
	BatchInfo []validator.BatchInfo
	// Valid since Recorded
	Preimages  map[common.Hash][]byte
	DelayedMsg []byte
}

func (e *validationEntry) ToInput() (*validator.ValidationInput, error) {
	if e.Stage != Ready {
		return nil, errors.New("cannot create input from non-ready entry")
	}
	return &validator.ValidationInput{
		Id:            uint64(e.Pos),
		HasDelayedMsg: e.HasDelayedMsg,
		DelayedMsgNr:  e.DelayedMsgNr,
		Preimages:     e.Preimages,
		BatchInfo:     e.BatchInfo,
		DelayedMsg:    e.DelayedMsg,
		StartState:    e.Start,
	}, nil
}

func newValidationEntry(
	pos arbutil.MessageIndex,
	start validator.GoGlobalState,
	end validator.GoGlobalState,
	msg *arbostypes.MessageWithMetadata,
	batch []byte,
	prevDelayed uint64,
) (*validationEntry, error) {
	batchInfo := validator.BatchInfo{
		Number: start.Batch,
		Data:   batch,
	}
	hasDelayed := false
	var delayedNum uint64
	if msg.DelayedMessagesRead == prevDelayed+1 {
		hasDelayed = true
		delayedNum = prevDelayed
	} else if msg.DelayedMessagesRead != prevDelayed {
		return nil, fmt.Errorf("illegal validation entry delayedMessage %d, previous %d", msg.DelayedMessagesRead, prevDelayed)
	}
	return &validationEntry{
		Stage:         ReadyForRecord,
		Pos:           pos,
		Start:         start,
		End:           end,
		HasDelayedMsg: hasDelayed,
		DelayedMsgNr:  delayedNum,
		msg:           msg,
		BatchInfo:     []validator.BatchInfo{batchInfo},
	}, nil
}

func NewStatelessBlockValidator(
	inboxReader InboxReaderInterface,
	inbox InboxTrackerInterface,
	streamer TransactionStreamerInterface,
	recorder execution.ExecutionRecorder,
	arbdb ethdb.Database,
	das arbstate.DataAvailabilityReader,
	config *BlockValidatorConfig,
	stack *node.Node,
) (*StatelessBlockValidator, error) {
	valClient := server_api.NewValidationClient(&config.ValidationServer, stack)
	execClient := server_api.NewExecutionClient(&config.ValidationServer, stack)
	validator := &StatelessBlockValidator{
		config:             config,
		execSpawner:        execClient,
		recorder:           recorder,
		validationSpawners: []validator.ValidationSpawner{valClient},
		inboxReader:        inboxReader,
		inboxTracker:       inbox,
		streamer:           streamer,
		db:                 arbdb,
		daService:          das,
	}
	return validator, nil
}

func (v *StatelessBlockValidator) GetModuleRootsToValidate() []common.Hash {
	v.moduleMutex.Lock()
	defer v.moduleMutex.Unlock()

	validatingModuleRoots := []common.Hash{v.currentWasmModuleRoot}
	if (v.currentWasmModuleRoot != v.pendingWasmModuleRoot && v.pendingWasmModuleRoot != common.Hash{}) {
		validatingModuleRoots = append(validatingModuleRoots, v.pendingWasmModuleRoot)
	}
	return validatingModuleRoots
}

func (v *StatelessBlockValidator) ValidationEntryRecord(ctx context.Context, e *validationEntry) error {
	if e.Stage != ReadyForRecord {
		return errors.Errorf("validation entry should be ReadyForRecord, is: %v", e.Stage)
	}
	// nothing to record for genesis
	if e.Pos == 0 {
		e.Stage = Ready
		return nil
	}
	recording, err := v.recorder.RecordBlockCreation(e.Pos, e.msg).Await(ctx)
	if err != nil {
		return err
	}
	if recording.BlockHash != e.End.BlockHash {
		return fmt.Errorf("recording failed: pos %d, hash expected %v, got %v", e.Pos, e.End.BlockHash, recording.BlockHash)
	}
	if e.HasDelayedMsg {
		delayedMsg, err := v.inboxTracker.GetDelayedMessageBytes(e.DelayedMsgNr)
		if err != nil {
			log.Error(
				"error while trying to read delayed msg for proving",
				"err", err, "seq", e.DelayedMsgNr, "pos", e.Pos,
			)
			return fmt.Errorf("error while trying to read delayed msg for proving: %w", err)
		}
		e.DelayedMsg = delayedMsg
	}

	e.BatchInfo = append(e.BatchInfo, recording.BatchInfo...)

	if recording.Preimages != nil {
		e.Preimages = recording.Preimages
	} else {
		e.Preimages = make(map[common.Hash][]byte)
	}
	for _, batch := range e.BatchInfo {
		if len(batch.Data) <= 40 {
			continue
		}
		if !arbstate.IsDASMessageHeaderByte(batch.Data[40]) {
			continue
		}
		if v.daService == nil {
			log.Warn("No DAS configured, but sequencer message found with DAS header")
		} else {
			_, err := arbstate.RecoverPayloadFromDasBatch(
				ctx, batch.Number, batch.Data, v.daService, e.Preimages, arbstate.KeysetValidate,
			)
			if err != nil {
				return err
			}
		}
	}

	e.msg = nil // no longer needed
	e.Stage = Ready
	return nil
}

func buildGlobalState(res execution.MessageResult, pos GlobalStatePosition) validator.GoGlobalState {
	return validator.GoGlobalState{
		BlockHash:  res.BlockHash,
		SendRoot:   res.SendRoot,
		Batch:      pos.BatchNumber,
		PosInBatch: pos.PosInBatch,
	}
}

func (v *StatelessBlockValidator) GlobalStatePositionsAtCount(count arbutil.MessageIndex) (GlobalStatePosition, GlobalStatePosition, error) {
	if count == 0 {
		return GlobalStatePosition{}, GlobalStatePosition{1, 0}, nil
	}
	batch, err := v.inboxTracker.FindL1BatchForMessage(count - 1)
	if err != nil {
		return GlobalStatePosition{}, GlobalStatePosition{}, err
	}
	return GlobalStatePositionsAtCount(v.inboxTracker, count, batch)
}

func (v *StatelessBlockValidator) CreateReadyValidationEntry(ctx context.Context, pos arbutil.MessageIndex) (*validationEntry, error) {
	msg, err := v.streamer.GetMessage(pos)
	if err != nil {
		return nil, err
	}
	result, err := v.streamer.ResultAtCount(pos + 1)
	if err != nil {
		return nil, err
	}
	var prevDelayed uint64
	if pos > 0 {
		prev, err := v.streamer.GetMessage(pos - 1)
		if err != nil {
			return nil, err
		}
		prevDelayed = prev.DelayedMessagesRead
	}
	prevResult, err := v.streamer.ResultAtCount(pos)
	if err != nil {
		return nil, err
	}
	startPos, endPos, err := v.GlobalStatePositionsAtCount(pos + 1)
	if err != nil {
		return nil, fmt.Errorf("failed calculating position for validation: %w", err)
	}
	start := buildGlobalState(*prevResult, startPos)
	end := buildGlobalState(*result, endPos)
	seqMsg, err := v.inboxReader.GetSequencerMessageBytes(startPos.BatchNumber).Await(ctx)
	if err != nil {
		return nil, err
	}
	entry, err := newValidationEntry(pos, start, end, msg, seqMsg, prevDelayed)
	if err != nil {
		return nil, err
	}
	err = v.ValidationEntryRecord(ctx, entry)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (v *StatelessBlockValidator) ValidateResult(
	ctx context.Context, pos arbutil.MessageIndex, useExec bool, moduleRoot common.Hash,
) (bool, *validator.GoGlobalState, error) {
	entry, err := v.CreateReadyValidationEntry(ctx, pos)
	if err != nil {
		return false, nil, err
	}
	input, err := entry.ToInput()
	if err != nil {
		return false, nil, err
	}
	var spawners []validator.ValidationSpawner
	if useExec {
		spawners = append(spawners, v.execSpawner)
	} else {
		spawners = v.validationSpawners
	}
	if len(spawners) == 0 {
		return false, &entry.End, errors.New("no validation defined")
	}
	var runs []validator.ValidationRun
	for _, spawner := range spawners {
		run := spawner.Launch(input, moduleRoot)
		runs = append(runs, run)
	}
	defer func() {
		for _, run := range runs {
			run.Cancel()
		}
	}()
	for _, run := range runs {
		gsEnd, err := run.Await(ctx)
		if err != nil || gsEnd != entry.End {
			return false, &gsEnd, err
		}
	}
	return true, &entry.End, nil
}

func (v *StatelessBlockValidator) OverrideRecorder(t *testing.T, recorder execution.ExecutionRecorder) {
	v.recorder = recorder
}

func (v *StatelessBlockValidator) Start(ctx_in context.Context) error {
	err := v.execSpawner.Start(ctx_in)
	if err != nil {
		return err
	}
	for _, spawner := range v.validationSpawners {
		if err := spawner.Start(ctx_in); err != nil {
			return err
		}
	}
	if v.config.PendingUpgradeModuleRoot != "" {
		if v.config.PendingUpgradeModuleRoot == "latest" {
			latest, err := v.execSpawner.LatestWasmModuleRoot().Await(ctx_in)
			if err != nil {
				return err
			}
			v.pendingWasmModuleRoot = latest
		} else {
			v.pendingWasmModuleRoot = common.HexToHash(v.config.PendingUpgradeModuleRoot)
			if (v.pendingWasmModuleRoot == common.Hash{}) {
				return errors.New("pending-upgrade-module-root config value illegal")
			}
		}
	}
	return nil
}

func (v *StatelessBlockValidator) Stop() {
	v.execSpawner.Stop()
	for _, spawner := range v.validationSpawners {
		spawner.Stop()
	}
}
