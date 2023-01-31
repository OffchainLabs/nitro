// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package staker

import (
	"context"
	"fmt"
	"sync"

	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/validator/server_api"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/validator"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/pkg/errors"
)

type StatelessBlockValidator struct {
	config *BlockValidatorConfig

	execSpawner        validator.ExecutionSpawner
	validationSpawners []validator.ValidationSpawner

	inboxReader       InboxReaderInterface
	inboxTracker      InboxTrackerInterface
	streamer          TransactionStreamerInterface
	blockchain        *core.BlockChain
	db                ethdb.Database
	daService         arbstate.DataAvailabilityReader
	genesisBlockNum   uint64
	recordingDatabase *arbitrum.RecordingDatabase

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
}

type TransactionStreamerInterface interface {
	BlockValidatorRegistrer
	GetMessage(seqNum arbutil.MessageIndex) (*arbostypes.MessageWithMetadata, error)
	GetGenesisBlockNumber() (uint64, error)
	PauseReorgs()
	ResumeReorgs()
}

type InboxReaderInterface interface {
	GetSequencerMessageBytes(ctx context.Context, seqNum uint64) ([]byte, error)
}

type L1ReaderInterface interface {
	Client() arbutil.L1Interface
	Subscribe(bool) (<-chan *types.Header, func())
	WaitForTxApproval(ctx context.Context, tx *types.Transaction) (*types.Receipt, error)
}

type GlobalStatePosition struct {
	BatchNumber uint64
	PosInBatch  uint64
}

func GlobalStatePositionsFor(
	tracker InboxTrackerInterface,
	pos arbutil.MessageIndex,
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
	if msgCountInBatch <= pos {
		return GlobalStatePosition{}, GlobalStatePosition{}, fmt.Errorf("batch %d has up to message %d, failed getting for %d", batch, msgCountInBatch-1, pos)
	}
	if firstInBatch > pos {
		return GlobalStatePosition{}, GlobalStatePosition{}, fmt.Errorf("batch %d starts from %d, failed getting for %d", batch, firstInBatch, pos)
	}
	startPos := GlobalStatePosition{batch, uint64(pos - firstInBatch)}
	if msgCountInBatch == pos+1 {
		return startPos, GlobalStatePosition{batch + 1, 0}, nil
	}
	return startPos, GlobalStatePosition{batch, uint64(pos + 1 - firstInBatch)}, nil
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
	Recorded
	Ready
)

type validationEntry struct {
	Stage ValidationEntryStage
	// Valid since ReadyforRecord:
	BlockNumber     uint64
	PrevBlockHash   common.Hash
	PrevBlockHeader *types.Header
	BlockHash       common.Hash
	BlockHeader     *types.Header
	HasDelayedMsg   bool
	DelayedMsgNr    uint64
	msg             *arbostypes.MessageWithMetadata
	// Valid since Recorded:
	Preimages  map[common.Hash][]byte
	BatchInfo  []validator.BatchInfo
	DelayedMsg []byte
	// Valid since Ready:
	StartPosition GlobalStatePosition
	EndPosition   GlobalStatePosition
}

func (v *validationEntry) start() (validator.GoGlobalState, error) {
	start := v.StartPosition
	prevExtraInfo, err := types.DeserializeHeaderExtraInformation(v.PrevBlockHeader)
	if err != nil {
		return validator.GoGlobalState{}, err
	}
	return validator.GoGlobalState{
		Batch:      start.BatchNumber,
		PosInBatch: start.PosInBatch,
		BlockHash:  v.PrevBlockHash,
		SendRoot:   prevExtraInfo.SendRoot,
	}, nil
}

func (v *validationEntry) expectedEnd() (validator.GoGlobalState, error) {
	extraInfo, err := types.DeserializeHeaderExtraInformation(v.BlockHeader)
	if err != nil {
		return validator.GoGlobalState{}, err
	}
	end := v.EndPosition
	return validator.GoGlobalState{
		Batch:      end.BatchNumber,
		PosInBatch: end.PosInBatch,
		BlockHash:  v.BlockHash,
		SendRoot:   extraInfo.SendRoot,
	}, nil
}

func (e *validationEntry) ToInput() (*validator.ValidationInput, error) {
	if e.Stage != Ready {
		return nil, errors.New("cannot create input from non-ready entry")
	}
	startState, err := e.start()
	if err != nil {
		return nil, err
	}
	return &validator.ValidationInput{
		Id:            e.BlockNumber,
		HasDelayedMsg: e.HasDelayedMsg,
		DelayedMsgNr:  e.DelayedMsgNr,
		Preimages:     e.Preimages,
		BatchInfo:     e.BatchInfo,
		DelayedMsg:    e.DelayedMsg,
		StartState:    startState,
	}, nil
}

func usingDelayedMsg(prevHeader *types.Header, header *types.Header) (bool, uint64) {
	if prevHeader == nil {
		return true, 0
	}
	if header.Nonce == prevHeader.Nonce {
		return false, 0
	}
	return true, prevHeader.Nonce.Uint64()
}

func newValidationEntry(
	prevHeader *types.Header,
	header *types.Header,
	msg *arbostypes.MessageWithMetadata,
) (*validationEntry, error) {
	hasDelayedMsg, delayedMsgNr := usingDelayedMsg(prevHeader, header)
	return &validationEntry{
		Stage:           ReadyForRecord,
		BlockNumber:     header.Number.Uint64(),
		PrevBlockHash:   prevHeader.Hash(),
		PrevBlockHeader: prevHeader,
		BlockHash:       header.Hash(),
		BlockHeader:     header,
		HasDelayedMsg:   hasDelayedMsg,
		DelayedMsgNr:    delayedMsgNr,
		msg:             msg,
	}, nil
}

func newRecordedValidationEntry(
	prevHeader *types.Header,
	header *types.Header,
	preimages map[common.Hash][]byte,
	batchInfos []validator.BatchInfo,
	delayedMsg []byte,
) (*validationEntry, error) {
	entry, err := newValidationEntry(prevHeader, header, nil)
	if err != nil {
		return nil, err
	}
	entry.Preimages = preimages
	entry.BatchInfo = batchInfos
	entry.DelayedMsg = delayedMsg
	entry.Stage = Recorded
	return entry, nil
}

func NewStatelessBlockValidator(
	inboxReader InboxReaderInterface,
	inbox InboxTrackerInterface,
	streamer TransactionStreamerInterface,
	blockchain *core.BlockChain,
	blockchainDb ethdb.Database,
	arbdb ethdb.Database,
	das arbstate.DataAvailabilityReader,
	config *BlockValidatorConfig,
) (*StatelessBlockValidator, error) {
	genesisBlockNum, err := streamer.GetGenesisBlockNumber()
	if err != nil {
		return nil, err
	}
	var jwt []byte
	if config.JWTSecret != "" {
		jwtHash, err := signature.LoadSigningKey(config.JWTSecret)
		if err != nil {
			return nil, err
		}
		jwt = jwtHash.Bytes()
	}
	valClient := server_api.NewValidationClient(config.URL, jwt)
	execClient := server_api.NewExecutionClient(config.URL, jwt)
	validator := &StatelessBlockValidator{
		config:             config,
		execSpawner:        execClient,
		validationSpawners: []validator.ValidationSpawner{valClient},
		inboxReader:        inboxReader,
		inboxTracker:       inbox,
		streamer:           streamer,
		blockchain:         blockchain,
		db:                 arbdb,
		daService:          das,
		genesisBlockNum:    genesisBlockNum,
		recordingDatabase:  arbitrum.NewRecordingDatabase(blockchainDb, blockchain),
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

func stateLogFunc(targetHeader, header *types.Header, hasState bool) {
	if targetHeader == nil || header == nil {
		return
	}
	gap := targetHeader.Number.Int64() - header.Number.Int64()
	step := int64(500)
	stage := "computing state"
	if !hasState {
		step = 3000
		stage = "looking for full block"
	}
	if (gap >= step) && (gap%step == 0) {
		log.Info("Setting up validation", "stage", stage, "current", header.Number, "target", targetHeader.Number)
	}
}

// If msg is nil, this will record block creation up to the point where message would be accessed (for a "too far" proof)
// If keepreference == true, reference to state of prevHeader is added (no reference added if an error is returned)
func (v *StatelessBlockValidator) RecordBlockCreation(
	ctx context.Context,
	prevHeader *types.Header,
	msg *arbostypes.MessageWithMetadata,
	keepReference bool,
) (common.Hash, map[common.Hash][]byte, []validator.BatchInfo, error) {

	recordingdb, chaincontext, recordingKV, err := v.recordingDatabase.PrepareRecording(ctx, prevHeader, stateLogFunc)
	if err != nil {
		return common.Hash{}, nil, nil, err
	}
	defer func() { v.recordingDatabase.Dereference(prevHeader) }()

	chainConfig := v.blockchain.Config()

	// Get the chain ID, both to validate and because the replay binary also gets the chain ID,
	// so we need to populate the recordingdb with preimages for retrieving the chain ID.
	if prevHeader != nil {
		initialArbosState, err := arbosState.OpenSystemArbosState(recordingdb, nil, true)
		if err != nil {
			return common.Hash{}, nil, nil, fmt.Errorf("error opening initial ArbOS state: %w", err)
		}
		chainId, err := initialArbosState.ChainId()
		if err != nil {
			return common.Hash{}, nil, nil, fmt.Errorf("error getting chain ID from initial ArbOS state: %w", err)
		}
		if chainId.Cmp(chainConfig.ChainID) != 0 {
			return common.Hash{}, nil, nil, fmt.Errorf("unexpected chain ID %v in ArbOS state, expected %v", chainId, chainConfig.ChainID)
		}
		genesisNum, err := initialArbosState.GenesisBlockNum()
		if err != nil {
			return common.Hash{}, nil, nil, fmt.Errorf("error getting genesis block number from initial ArbOS state: %w", err)
		}
		expectedNum := chainConfig.ArbitrumChainParams.GenesisBlockNum
		if genesisNum != expectedNum {
			return common.Hash{}, nil, nil, fmt.Errorf("unexpected genesis block number %v in ArbOS state, expected %v", genesisNum, expectedNum)
		}
	}

	var blockHash common.Hash
	var readBatchInfo []validator.BatchInfo
	if msg != nil {
		batchFetcher := func(batchNum uint64) ([]byte, error) {
			data, err := v.inboxReader.GetSequencerMessageBytes(ctx, batchNum)
			if err != nil {
				return nil, err
			}
			readBatchInfo = append(readBatchInfo, validator.BatchInfo{
				Number: batchNum,
				Data:   data,
			})
			return data, nil
		}
		// Re-fetch the batch instead of using our cached cost,
		// as the replay binary won't have the cache populated.
		msg.Message.BatchGasCost = nil
		block, _, err := arbos.ProduceBlock(
			msg.Message,
			msg.DelayedMessagesRead,
			prevHeader,
			recordingdb,
			chaincontext,
			chainConfig,
			batchFetcher,
		)
		if err != nil {
			return common.Hash{}, nil, nil, err
		}
		blockHash = block.Hash()
	}

	preimages, err := v.recordingDatabase.PreimagesFromRecording(chaincontext, recordingKV)
	if err != nil {
		return common.Hash{}, nil, nil, err
	}
	if keepReference {
		prevHeader = nil
	}
	return blockHash, preimages, readBatchInfo, err
}

func (v *StatelessBlockValidator) ValidationEntryRecord(ctx context.Context, e *validationEntry, keepReference bool) error {
	if e.Stage != ReadyForRecord {
		return errors.Errorf("validation entry should be ReadyForRecord, is: %v", e.Stage)
	}
	if e.PrevBlockHeader == nil {
		e.Stage = Recorded
		return nil
	}
	blockhash, preimages, readBatchInfo, err := v.RecordBlockCreation(ctx, e.PrevBlockHeader, e.msg, keepReference)
	if err != nil {
		return err
	}
	if blockhash != e.BlockHash {
		return fmt.Errorf("recording failed: blockNum %d, hash expected %v, got %v", e.BlockNumber, e.BlockHash, blockhash)
	}
	if e.HasDelayedMsg {
		delayedMsg, err := v.inboxTracker.GetDelayedMessageBytes(e.DelayedMsgNr)
		if err != nil {
			log.Error(
				"error while trying to read delayed msg for proving",
				"err", err, "seq", e.DelayedMsgNr, "blockNr", e.BlockNumber,
			)
			return fmt.Errorf("error while trying to read delayed msg for proving: %w", err)
		}
		e.DelayedMsg = delayedMsg
	}
	e.Preimages = preimages
	e.BatchInfo = readBatchInfo
	e.msg = nil // no longer needed
	e.Stage = Recorded
	return nil
}

func (v *StatelessBlockValidator) ValidationEntryAddSeqMessage(ctx context.Context, e *validationEntry,
	startPos, endPos GlobalStatePosition, seqMsg []byte) error {
	if e.Stage != Recorded {
		return fmt.Errorf("validation entry stage should be Recorded, is: %v", e.Stage)
	}
	if e.Preimages == nil {
		e.Preimages = make(map[common.Hash][]byte)
	}
	e.StartPosition = startPos
	e.EndPosition = endPos
	seqMsgBatchInfo := validator.BatchInfo{
		Number: startPos.BatchNumber,
		Data:   seqMsg,
	}
	e.BatchInfo = append(e.BatchInfo, seqMsgBatchInfo)

	for _, batch := range e.BatchInfo {
		if len(batch.Data) <= 40 {
			continue
		}
		if !arbstate.IsDASMessageHeaderByte(batch.Data[40]) {
			continue
		}
		if v.daService == nil {
			log.Error("No DAS configured, but sequencer message found with DAS header")
			if v.blockchain.Config().ArbitrumChainParams.DataAvailabilityCommittee {
				return errors.New("processing data availability chain without DAS configured")
			}
		} else {
			_, err := arbstate.RecoverPayloadFromDasBatch(
				ctx, batch.Number, batch.Data, v.daService, e.Preimages, arbstate.KeysetValidate,
			)
			if err != nil {
				return err
			}
		}
	}
	e.Stage = Ready
	return nil
}

func (v *StatelessBlockValidator) CreateReadyValidationEntry(ctx context.Context, header *types.Header) (*validationEntry, error) {
	if header == nil {
		return nil, errors.New("header not found")
	}
	blockNum := header.Number.Uint64()
	msgIndex := arbutil.BlockNumberToMessageCount(blockNum, v.genesisBlockNum) - 1
	prevHeader := v.blockchain.GetHeaderByNumber(blockNum - 1)
	if prevHeader == nil {
		return nil, errors.New("prev header not found")
	}
	if header.ParentHash != prevHeader.Hash() {
		return nil, fmt.Errorf("hashes don't match block %d hash %v parent %v prev-found %v",
			blockNum, header.Hash(), header.ParentHash, prevHeader.Hash())
	}
	msg, err := v.streamer.GetMessage(msgIndex)
	if err != nil {
		return nil, err
	}
	resHash, preimages, readBatchInfo, err := v.RecordBlockCreation(ctx, prevHeader, msg, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get block data to validate: %w", err)
	}
	if resHash != header.Hash() {
		return nil, fmt.Errorf("wrong hash expected %s got %s", header.Hash(), resHash)
	}
	batchCount, err := v.inboxTracker.GetBatchCount()
	if err != nil {
		return nil, err
	}
	batch, err := FindBatchContainingMessageIndex(v.inboxTracker, msgIndex, batchCount)
	if err != nil {
		return nil, err
	}

	startPos, endPos, err := GlobalStatePositionsFor(v.inboxTracker, msgIndex, batch)
	if err != nil {
		return nil, fmt.Errorf("failed calculating position for validation: %w", err)
	}

	usingDelayed, delaydNr := usingDelayedMsg(prevHeader, header)
	var delayed []byte
	if usingDelayed {
		delayed, err = v.inboxTracker.GetDelayedMessageBytes(delaydNr)
		if err != nil {
			return nil, fmt.Errorf("error while trying to read delayed msg for proving: %w", err)
		}
	}
	entry, err := newRecordedValidationEntry(prevHeader, header, preimages, readBatchInfo, delayed)
	if err != nil {
		return nil, fmt.Errorf("failed to create validation entry %w", err)
	}

	seqMsg, err := v.inboxReader.GetSequencerMessageBytes(ctx, startPos.BatchNumber)
	if err != nil {
		return nil, err
	}
	err = v.ValidationEntryAddSeqMessage(ctx, entry, startPos, endPos, seqMsg)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (v *StatelessBlockValidator) ValidateBlock(
	ctx context.Context, header *types.Header, useExec bool, moduleRoot common.Hash,
) (bool, error) {
	entry, err := v.CreateReadyValidationEntry(ctx, header)
	if err != nil {
		return false, err
	}
	expEnd, err := entry.expectedEnd()
	if err != nil {
		return false, err
	}
	input, err := entry.ToInput()
	if err != nil {
		return false, err
	}
	var spawners []validator.ValidationSpawner
	if useExec {
		spawners = append(spawners, v.execSpawner)
	} else {
		spawners = v.validationSpawners
	}
	if len(spawners) == 0 {
		return false, errors.New("no validation defined")
	}
	var runs []validator.ValidationRun
	for _, spawner := range spawners {
		run := spawner.Launch(input, moduleRoot)
		runs = append(runs, run)
	}
	defer func() {
		for _, run := range runs {
			run.Close()
		}
	}()
	for _, run := range runs {
		gsEnd, err := run.Await(ctx)
		if err != nil || gsEnd != expEnd {
			return false, err
		}
	}
	return true, nil
}

func (v *StatelessBlockValidator) RecordDBReferenceCount() int64 {
	return v.recordingDatabase.ReferenceCount()
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
			latest, err := v.execSpawner.LatestWasmModuleRoot()
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
