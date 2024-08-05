// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package staker

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"runtime"
	"testing"

	"github.com/offchainlabs/nitro/arbstate/daprovider"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/client/redis"

	validatorclient "github.com/offchainlabs/nitro/validator/client"
)

type StatelessBlockValidator struct {
	config *BlockValidatorConfig

	execSpawners   []validator.ExecutionSpawner
	redisValidator *redis.ValidationClient

	recorder execution.ExecutionRecorder

	inboxReader  InboxReaderInterface
	inboxTracker InboxTrackerInterface
	streamer     TransactionStreamerInterface
	db           ethdb.Database
	dapReaders   []daprovider.Reader
}

type BlockValidatorRegistrer interface {
	SetBlockValidator(*BlockValidator)
}

type InboxTrackerInterface interface {
	BlockValidatorRegistrer
	GetDelayedMessageBytes(context.Context, uint64) ([]byte, error)
	GetBatchMessageCount(seqNum uint64) (arbutil.MessageIndex, error)
	GetBatchAcc(seqNum uint64) (common.Hash, error)
	GetBatchCount() (uint64, error)
	FindInboxBatchContainingMessage(pos arbutil.MessageIndex) (uint64, bool, error)
}

type TransactionStreamerInterface interface {
	BlockValidatorRegistrer
	GetProcessedMessageCount() (arbutil.MessageIndex, error)
	GetMessage(seqNum arbutil.MessageIndex) (*arbostypes.MessageWithMetadata, error)
	ResultAtCount(count arbutil.MessageIndex) (*execution.MessageResult, error)
	PauseReorgs()
	ResumeReorgs()
	ChainConfig() *params.ChainConfig
}

type InboxReaderInterface interface {
	GetSequencerMessageBytes(ctx context.Context, seqNum uint64) ([]byte, common.Hash, error)
	GetFinalizedMsgCount(ctx context.Context) (arbutil.MessageIndex, error)
}

type GlobalStatePosition struct {
	BatchNumber uint64
	PosInBatch  uint64
}

// return the globalState position before and after processing message at the specified count
// batch-number must be provided by caller
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

func FindBatchContainingMessageIndex(
	tracker InboxTrackerInterface, pos arbutil.MessageIndex, high uint64,
) (uint64, error) {
	var low uint64
	// Iteration preconditions:
	// - high >= low
	// - msgCount(low - 1) <= pos implies low <= target
	// - msgCount(high) > pos implies high >= target
	// Therefore, if low == high, then low == high == target
	for high > low {
		// Due to integer rounding, mid >= low && mid < high
		mid := (low + high) / 2
		count, err := tracker.GetBatchMessageCount(mid)
		if err != nil {
			return 0, err
		}
		if count < pos {
			// Must narrow as mid >= low, therefore mid + 1 > low, therefore newLow > oldLow
			// Keeps low precondition as msgCount(mid) < pos
			low = mid + 1
		} else if count == pos {
			return mid + 1, nil
		} else if count == pos+1 || mid == low { // implied: count > pos
			return mid, nil
		} else { // implied: count > pos + 1
			// Must narrow as mid < high, therefore newHigh < lowHigh
			// Keeps high precondition as msgCount(mid) > pos
			high = mid
		}
	}
	return low, nil
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
	ChainConfig   *params.ChainConfig
	// valid when created, removed after recording
	msg *arbostypes.MessageWithMetadata
	// Has batch when created - others could be added on record
	BatchInfo []validator.BatchInfo
	// Valid since Ready
	Preimages  map[arbutil.PreimageType]map[common.Hash][]byte
	UserWasms  state.UserWasms
	DelayedMsg []byte
}

func (e *validationEntry) ToInput(stylusArchs []string) (*validator.ValidationInput, error) {
	if e.Stage != Ready {
		return nil, errors.New("cannot create input from non-ready entry")
	}
	res := validator.ValidationInput{
		Id:            uint64(e.Pos),
		HasDelayedMsg: e.HasDelayedMsg,
		DelayedMsgNr:  e.DelayedMsgNr,
		Preimages:     e.Preimages,
		UserWasms:     make(map[string]map[common.Hash][]byte, len(e.UserWasms)),
		BatchInfo:     e.BatchInfo,
		DelayedMsg:    e.DelayedMsg,
		StartState:    e.Start,
		DebugChain:    e.ChainConfig.DebugMode(),
	}
	for _, stylusArch := range stylusArchs {
		res.UserWasms[stylusArch] = make(map[common.Hash][]byte)
	}
	for hash, info := range e.UserWasms {
		for _, stylusArch := range stylusArchs {
			if stylusArch == "wavm" {
				res.UserWasms[stylusArch][hash] = info.Module
			} else if stylusArch == runtime.GOARCH {
				res.UserWasms[stylusArch][hash] = info.Asm
			} else {
				return nil, fmt.Errorf("stylusArch not supported by block validator: %v", stylusArch)
			}
		}
	}
	return &res, nil
}

func newValidationEntry(
	pos arbutil.MessageIndex,
	start validator.GoGlobalState,
	end validator.GoGlobalState,
	msg *arbostypes.MessageWithMetadata,
	batch []byte,
	batchBlockHash common.Hash,
	prevDelayed uint64,
	chainConfig *params.ChainConfig,
) (*validationEntry, error) {
	batchInfo := validator.BatchInfo{
		Number:    start.Batch,
		BlockHash: batchBlockHash,
		Data:      batch,
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
		ChainConfig:   chainConfig,
	}, nil
}

func NewStatelessBlockValidator(
	inboxReader InboxReaderInterface,
	inbox InboxTrackerInterface,
	streamer TransactionStreamerInterface,
	recorder execution.ExecutionRecorder,
	arbdb ethdb.Database,
	dapReaders []daprovider.Reader,
	config func() *BlockValidatorConfig,
	stack *node.Node,
) (*StatelessBlockValidator, error) {
	var executionSpawners []validator.ExecutionSpawner
	var redisValClient *redis.ValidationClient

	if config().RedisValidationClientConfig.Enabled() {
		var err error
		redisValClient, err = redis.NewValidationClient(&config().RedisValidationClientConfig)
		if err != nil {
			return nil, fmt.Errorf("creating new redis validation client: %w", err)
		}
	}
	configs := config().ValidationServerConfigs
	for i := range configs {
		i := i
		confFetcher := func() *rpcclient.ClientConfig { return &config().ValidationServerConfigs[i] }
		executionSpawners = append(executionSpawners, validatorclient.NewExecutionClient(confFetcher, stack))
	}

	if len(executionSpawners) == 0 {
		return nil, errors.New("no enabled execution servers")
	}

	return &StatelessBlockValidator{
		config:         config(),
		recorder:       recorder,
		redisValidator: redisValClient,
		inboxReader:    inboxReader,
		inboxTracker:   inbox,
		streamer:       streamer,
		db:             arbdb,
		dapReaders:     dapReaders,
		execSpawners:   executionSpawners,
	}, nil
}

func (v *StatelessBlockValidator) readBatch(ctx context.Context, batchNum uint64) (bool, []byte, common.Hash, arbutil.MessageIndex, error) {
	batchCount, err := v.inboxTracker.GetBatchCount()
	if err != nil {
		return false, nil, common.Hash{}, 0, err
	}
	if batchCount <= batchNum {
		return false, nil, common.Hash{}, 0, nil
	}
	batchMsgCount, err := v.inboxTracker.GetBatchMessageCount(batchNum)
	if err != nil {
		return false, nil, common.Hash{}, 0, err
	}
	batch, batchBlockHash, err := v.inboxReader.GetSequencerMessageBytes(ctx, batchNum)
	if err != nil {
		return false, nil, common.Hash{}, 0, err
	}
	return true, batch, batchBlockHash, batchMsgCount, nil
}

func (v *StatelessBlockValidator) ValidationEntryRecord(ctx context.Context, e *validationEntry) error {
	if e.Stage != ReadyForRecord {
		return fmt.Errorf("validation entry should be ReadyForRecord, is: %v", e.Stage)
	}
	e.Preimages = make(map[arbutil.PreimageType]map[common.Hash][]byte)
	if e.Pos != 0 {
		recording, err := v.recorder.RecordBlockCreation(ctx, e.Pos, e.msg)
		if err != nil {
			return err
		}
		if recording.BlockHash != e.End.BlockHash {
			return fmt.Errorf("recording failed: pos %d, hash expected %v, got %v", e.Pos, e.End.BlockHash, recording.BlockHash)
		}
		// record any additional batch fetching
		batchFetcher := func(batchNum uint64) ([]byte, error) {
			found, data, hash, _, err := v.readBatch(ctx, batchNum)
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, errors.New("batch not found")
			}
			e.BatchInfo = append(e.BatchInfo, validator.BatchInfo{
				Number:    batchNum,
				BlockHash: hash,
				Data:      data,
			})
			return data, nil
		}
		e.msg.Message.BatchGasCost = nil
		err = e.msg.Message.FillInBatchGasCost(batchFetcher)
		if err != nil {
			return err
		}

		if recording.Preimages != nil {
			e.Preimages[arbutil.Keccak256PreimageType] = recording.Preimages
		}
		e.UserWasms = recording.UserWasms
	}
	if e.HasDelayedMsg {
		delayedMsg, err := v.inboxTracker.GetDelayedMessageBytes(ctx, e.DelayedMsgNr)
		if err != nil {
			log.Error(
				"error while trying to read delayed msg for proving",
				"err", err, "seq", e.DelayedMsgNr, "pos", e.Pos,
			)
			return fmt.Errorf("error while trying to read delayed msg for proving: %w", err)
		}
		e.DelayedMsg = delayedMsg
	}
	for _, batch := range e.BatchInfo {
		if len(batch.Data) <= 40 {
			continue
		}
		foundDA := false
		for _, dapReader := range v.dapReaders {
			if dapReader != nil && dapReader.IsValidHeaderByte(batch.Data[40]) {
				preimageRecorder := daprovider.RecordPreimagesTo(e.Preimages)
				_, err := dapReader.RecoverPayloadFromBatch(ctx, batch.Number, batch.BlockHash, batch.Data, preimageRecorder, true)
				if err != nil {
					// Matches the way keyset validation was done inside DAS readers i.e logging the error
					//  But other daproviders might just want to return the error
					if errors.Is(err, daprovider.ErrSeqMsgValidation) && daprovider.IsDASMessageHeaderByte(batch.Data[40]) {
						log.Error(err.Error())
					} else {
						return err
					}
				}
				foundDA = true
				break
			}
		}
		if !foundDA {
			if daprovider.IsDASMessageHeaderByte(batch.Data[40]) {
				log.Error("No DAS Reader configured, but sequencer message found with DAS header")
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

// return the globalState position before and after processing message at the specified count
func (v *StatelessBlockValidator) GlobalStatePositionsAtCount(count arbutil.MessageIndex) (GlobalStatePosition, GlobalStatePosition, error) {
	if count == 0 {
		return GlobalStatePosition{}, GlobalStatePosition{}, errors.New("no initial state for count==0")
	}
	if count == 1 {
		return GlobalStatePosition{}, GlobalStatePosition{1, 0}, nil
	}
	batch, found, err := v.inboxTracker.FindInboxBatchContainingMessage(count - 1)
	if err != nil {
		return GlobalStatePosition{}, GlobalStatePosition{}, err
	}
	if !found {
		return GlobalStatePosition{}, GlobalStatePosition{}, errors.New("batch not found on L1 yet")
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
	seqMsg, batchBlockHash, err := v.inboxReader.GetSequencerMessageBytes(ctx, startPos.BatchNumber)
	if err != nil {
		return nil, err
	}
	entry, err := newValidationEntry(pos, start, end, msg, seqMsg, batchBlockHash, prevDelayed, v.streamer.ChainConfig())
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
	var run validator.ValidationRun
	if !useExec {
		if v.redisValidator != nil {
			if validator.SpawnerSupportsModule(v.redisValidator, moduleRoot) {
				input, err := entry.ToInput(v.redisValidator.StylusArchs())
				if err != nil {
					return false, nil, err
				}
				run = v.redisValidator.Launch(input, moduleRoot)
			}
		}
	}
	if run == nil {
		for _, spawner := range v.execSpawners {
			if validator.SpawnerSupportsModule(spawner, moduleRoot) {
				input, err := entry.ToInput(spawner.StylusArchs())
				if err != nil {
					return false, nil, err
				}
				run = spawner.Launch(input, moduleRoot)
				break
			}
		}
	}
	if run == nil {
		return false, nil, fmt.Errorf("validation with WasmModuleRoot %v not supported by node", moduleRoot)
	}
	defer run.Cancel()
	gsEnd, err := run.Await(ctx)
	if err != nil || gsEnd != entry.End {
		return false, &gsEnd, err
	}
	return true, &entry.End, nil
}

func (v *StatelessBlockValidator) OverrideRecorder(t *testing.T, recorder execution.ExecutionRecorder) {
	v.recorder = recorder
}

func (v *StatelessBlockValidator) GetLatestWasmModuleRoot(ctx context.Context) (common.Hash, error) {
	var lastErr error
	for _, spawner := range v.execSpawners {
		var latest common.Hash
		latest, lastErr = spawner.LatestWasmModuleRoot().Await(ctx)
		if latest != (common.Hash{}) && lastErr == nil {
			return latest, nil
		}
		if ctx.Err() != nil {
			return common.Hash{}, ctx.Err()
		}
	}
	return common.Hash{}, fmt.Errorf("couldn't detect latest WasmModuleRoot: %w", lastErr)
}

func (v *StatelessBlockValidator) Start(ctx_in context.Context) error {
	if v.redisValidator != nil {
		if err := v.redisValidator.Start(ctx_in); err != nil {
			return fmt.Errorf("starting execution spawner: %w", err)
		}
	}
	for i, spawner := range v.execSpawners {
		if err := spawner.Start(ctx_in); err != nil {
			if u, parseErr := url.Parse(v.config.ValidationServerConfigs[i].URL); parseErr == nil {
				if u.Scheme != "ws" && u.Scheme != "wss" {
					return fmt.Errorf("validation server's url scheme is unsupported, it should either be ws or wss, url:%s err: %w", v.config.ValidationServerConfigs[i].URL, err)
				}
			}
			return err
		}
	}
	return nil
}

func (v *StatelessBlockValidator) Stop() {
	for _, spawner := range v.execSpawners {
		spawner.Stop()
	}
	if v.redisValidator != nil {
		v.redisValidator.Stop()
	}
}
