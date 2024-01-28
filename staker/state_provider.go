// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package staker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/option"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"

	"github.com/offchainlabs/nitro/arbutil"
	challengecache "github.com/offchainlabs/nitro/staker/challenge-cache"
	"github.com/offchainlabs/nitro/validator"
)

var (
	_ l2stateprovider.ProofCollector          = (*StateManager)(nil)
	_ l2stateprovider.L2MessageStateCollector = (*StateManager)(nil)
	_ l2stateprovider.MachineHashCollector    = (*StateManager)(nil)
	_ l2stateprovider.ExecutionProvider       = (*StateManager)(nil)
)

var (
	ErrChainCatchingUp = errors.New("chain catching up")
)

type BoldConfig struct {
	Enable                             bool   `koanf:"enable"`
	Evil                               bool   `koanf:"evil"`
	Mode                               string `koanf:"mode"`
	BlockChallengeLeafHeight           uint64 `koanf:"block-challenge-leaf-height"`
	BigStepLeafHeight                  uint64 `koanf:"big-step-leaf-height"`
	SmallStepLeafHeight                uint64 `koanf:"small-step-leaf-height"`
	NumBigSteps                        uint64 `koanf:"num-big-steps"`
	ValidatorName                      string `koanf:"validator-name"`
	MachineLeavesCachePath             string `koanf:"machine-leaves-cache-path"`
	AssertionPostingIntervalSeconds    uint64 `koanf:"assertion-posting-interval-seconds"`
	AssertionScanningIntervalSeconds   uint64 `koanf:"assertion-scanning-interval-seconds"`
	AssertionConfirmingIntervalSeconds uint64 `koanf:"assertion-confirming-interval-seconds"`
	EdgeTrackerWakeIntervalSeconds     uint64 `koanf:"edge-tracker-wake-interval-seconds"`
	API                                bool   `koanf:"api"`
	APIHost                            string `koanf:"api-host"`
	APIPort                            uint16 `koanf:"api-port"`
	APIDBPath                          string `koanf:"api-db-path"`
}

var DefaultBoldConfig = BoldConfig{
	Enable:                             false,
	Evil:                               false,
	Mode:                               "make-mode",
	BlockChallengeLeafHeight:           1 << 5,
	BigStepLeafHeight:                  1 << 8,
	SmallStepLeafHeight:                1 << 10,
	NumBigSteps:                        3,
	ValidatorName:                      "default-validator",
	MachineLeavesCachePath:             "/tmp/machine-leaves-cache",
	AssertionPostingIntervalSeconds:    30,
	AssertionScanningIntervalSeconds:   30,
	AssertionConfirmingIntervalSeconds: 60,
	EdgeTrackerWakeIntervalSeconds:     1,
}

func (c *BoldConfig) Validate() error {
	return nil
}

type StateManager struct {
	validator            *StatelessBlockValidator
	historyCache         challengecache.HistoryCommitmentCacher
	challengeLeafHeights []l2stateprovider.Height
	validatorName        string
	sync.RWMutex
}

func NewStateManager(
	val *StatelessBlockValidator,
	cacheBaseDir string,
	challengeLeafHeights []l2stateprovider.Height,
	validatorName string,
) (*StateManager, error) {
	historyCache := challengecache.New(cacheBaseDir)
	sm := &StateManager{
		validator:            val,
		historyCache:         historyCache,
		challengeLeafHeights: challengeLeafHeights,
		validatorName:        validatorName,
	}
	return sm, nil
}

// AgreesWithExecutionState If the state manager locally has this validated execution state.
// Returns ErrNoExecutionState if not found, or ErrChainCatchingUp if not yet
// validated / syncing.
func (s *StateManager) AgreesWithExecutionState(ctx context.Context, state *protocol.ExecutionState) error {
	if state.GlobalState.PosInBatch != 0 {
		return fmt.Errorf("position in batch must be zero, but got %d: %+v", state.GlobalState.PosInBatch, state)
	}
	// We always agree with the genesis batch.
	batchIndex := state.GlobalState.Batch
	if batchIndex == 0 && state.GlobalState.PosInBatch == 0 {
		return nil
	}
	// We always agree with the init message.
	if batchIndex == 1 && state.GlobalState.PosInBatch == 0 {
		return nil
	}

	// Because an execution state from the assertion chain fully consumes the preceding batch,
	// we actually want to check if we agree with the last state of the preceding batch, so
	// we decrement the batch index by 1.
	batchIndex -= 1

	totalBatches, err := s.validator.inboxTracker.GetBatchCount()
	if err != nil {
		return err
	}

	// If the batch index is >= the total number of batches we have in our inbox tracker,
	// we are still catching up to the chain.
	if batchIndex >= totalBatches {
		return ErrChainCatchingUp
	}
	messageCount, err := s.validator.inboxTracker.GetBatchMessageCount(batchIndex)
	if err != nil {
		return err
	}
	validatedGlobalState, err := s.findGlobalStateFromMessageCountAndBatch(messageCount, l2stateprovider.Batch(batchIndex))
	if err != nil {
		return err
	}
	// We check if the block hash and send root match at our expected result.
	if state.GlobalState.BlockHash != validatedGlobalState.BlockHash || state.GlobalState.SendRoot != validatedGlobalState.SendRoot {
		return l2stateprovider.ErrNoExecutionState
	}
	return nil
}

// ExecutionStateAfterBatchCount Produces the l2 state to assert at the message number specified.
// Makes sure that PosInBatch is always 0
func (s *StateManager) ExecutionStateAfterBatchCount(ctx context.Context, batchCount uint64) (*protocol.ExecutionState, error) {
	if batchCount == 0 {
		return nil, errors.New("batch count cannot be zero")
	}
	batchIndex := batchCount - 1
	messageCount, err := s.validator.inboxTracker.GetBatchMessageCount(batchIndex)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, fmt.Errorf("%w: batch count %d", l2stateprovider.ErrChainCatchingUp, batchCount)
		}
		return nil, err
	}
	globalState, err := s.findGlobalStateFromMessageCountAndBatch(messageCount, l2stateprovider.Batch(batchIndex))
	if err != nil {
		return nil, err
	}
	executionState := &protocol.ExecutionState{
		GlobalState:   protocol.GoGlobalState(globalState),
		MachineStatus: protocol.MachineStatusFinished,
	}
	// If the execution state did not consume all messages in a batch, we then return
	// the next batch's execution state.
	if executionState.GlobalState.PosInBatch != 0 {
		executionState.GlobalState.Batch += 1
		executionState.GlobalState.PosInBatch = 0
	}
	return executionState, nil
}

func (s *StateManager) StatesInBatchRange(
	fromHeight,
	toHeight l2stateprovider.Height,
	fromBatch,
	toBatch l2stateprovider.Batch,
) ([]common.Hash, error) {
	// Check the integrity of the arguments.
	if fromBatch >= toBatch {
		return nil, fmt.Errorf("from batch %v cannot be greater than or equal to batch %v", fromBatch, toBatch)
	}
	if fromHeight > toHeight {
		return nil, fmt.Errorf("from height %v cannot be greater than to height %v", fromHeight, toHeight)
	}
	// Compute the total desired hashes from this request.
	totalDesiredHashes := (toHeight - fromHeight) + 1

	// Get the from batch's message count.
	prevBatchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(uint64(fromBatch) - 1)
	if err != nil {
		return nil, err
	}
	executionResult, err := s.validator.streamer.ResultAtCount(prevBatchMsgCount)
	if err != nil {
		return nil, err
	}
	startState := validator.GoGlobalState{
		BlockHash:  executionResult.BlockHash,
		SendRoot:   executionResult.SendRoot,
		Batch:      uint64(fromBatch),
		PosInBatch: 0,
	}
	machineHashes := []common.Hash{machineHash(startState)}
	states := []validator.GoGlobalState{startState}

	for batch := fromBatch; batch < toBatch; batch++ {
		batchMessageCount, err := s.validator.inboxTracker.GetBatchMessageCount(uint64(batch))
		if err != nil {
			return nil, err
		}
		messagesInBatch := batchMessageCount - prevBatchMsgCount

		// Obtain the states for each message in the batch.
		for i := uint64(0); i < uint64(messagesInBatch); i++ {
			msgIndex := uint64(prevBatchMsgCount) + i
			messageCount := msgIndex + 1
			executionResult, err := s.validator.streamer.ResultAtCount(arbutil.MessageIndex(messageCount))
			if err != nil {
				return nil, err
			}
			// If the position in batch is equal to the number of messages in the batch,
			// we do not include this state. Instead, we break and include the state
			// that fully consumes the batch.
			if i+1 == uint64(messagesInBatch) {
				break
			}
			state := validator.GoGlobalState{
				BlockHash:  executionResult.BlockHash,
				SendRoot:   executionResult.SendRoot,
				Batch:      uint64(batch),
				PosInBatch: i + 1,
			}
			states = append(states, state)
			machineHashes = append(machineHashes, machineHash(state))
		}

		// Fully consume the batch.
		executionResult, err := s.validator.streamer.ResultAtCount(batchMessageCount)
		if err != nil {
			return nil, err
		}
		state := validator.GoGlobalState{
			BlockHash:  executionResult.BlockHash,
			SendRoot:   executionResult.SendRoot,
			Batch:      uint64(batch) + 1,
			PosInBatch: 0,
		}
		states = append(states, state)
		machineHashes = append(machineHashes, machineHash(state))
		prevBatchMsgCount = batchMessageCount
	}
	for uint64(len(machineHashes)) < uint64(totalDesiredHashes) {
		machineHashes = append(machineHashes, machineHashes[len(machineHashes)-1])
	}
	return machineHashes[fromHeight : toHeight+1], nil
}

func machineHash(gs validator.GoGlobalState) common.Hash {
	return crypto.Keccak256Hash([]byte("Machine finished:"), gs.Hash().Bytes())
}

func (s *StateManager) findGlobalStateFromMessageCountAndBatch(count arbutil.MessageIndex, batchIndex l2stateprovider.Batch) (validator.GoGlobalState, error) {
	var prevBatchMsgCount arbutil.MessageIndex
	var err error
	if batchIndex > 0 {
		prevBatchMsgCount, err = s.validator.inboxTracker.GetBatchMessageCount(uint64(batchIndex) - 1)
		if err != nil {
			return validator.GoGlobalState{}, err
		}
		if prevBatchMsgCount > count {
			return validator.GoGlobalState{}, errors.New("bad batch provided")
		}
	}
	res, err := s.validator.streamer.ResultAtCount(count)
	if err != nil {
		return validator.GoGlobalState{}, fmt.Errorf("%s: could not check if we have result at count %d: %w", s.validatorName, count, err)
	}
	return validator.GoGlobalState{
		BlockHash:  res.BlockHash,
		SendRoot:   res.SendRoot,
		Batch:      uint64(batchIndex),
		PosInBatch: uint64(count - prevBatchMsgCount),
	}, nil
}

// L2MessageStatesUpTo Computes a block history commitment from a start L2 message to an end L2 message index
// and up to a required batch index. The hashes used for this commitment are the machine hashes
// at each message number.
func (s *StateManager) L2MessageStatesUpTo(
	_ context.Context,
	fromHeight l2stateprovider.Height,
	toHeight option.Option[l2stateprovider.Height],
	fromBatch,
	toBatch l2stateprovider.Batch,
) ([]common.Hash, error) {
	var to l2stateprovider.Height
	if !toHeight.IsNone() {
		to = toHeight.Unwrap()
	} else {
		blockChallengeLeafHeight := s.challengeLeafHeights[0]
		to = blockChallengeLeafHeight
	}
	items, err := s.StatesInBatchRange(fromHeight, to, fromBatch, toBatch)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// CollectMachineHashes Collects a list of machine hashes at a message number based on some configuration parameters.
func (s *StateManager) CollectMachineHashes(
	ctx context.Context, cfg *l2stateprovider.HashCollectorConfig,
) ([]common.Hash, error) {
	s.Lock()
	defer s.Unlock()
	prevBatchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(uint64(cfg.FromBatch - 1))
	if err != nil {
		return nil, fmt.Errorf("could not get batch message count at %d: %w", cfg.FromBatch, err)
	}
	messageNum := (prevBatchMsgCount + arbutil.MessageIndex(cfg.BlockChallengeHeight))
	cacheKey := &challengecache.Key{
		WavmModuleRoot: cfg.WasmModuleRoot,
		MessageHeight:  protocol.Height(messageNum),
		StepHeights:    cfg.StepHeights,
	}
	if s.historyCache != nil {
		cachedRoots, err := s.historyCache.Get(cacheKey, cfg.NumDesiredHashes)
		switch {
		case err == nil:
			return cachedRoots, nil
		case !errors.Is(err, challengecache.ErrNotFoundInCache):
			return nil, err
		}
	}
	entry, err := s.validator.CreateReadyValidationEntry(ctx, messageNum)
	if err != nil {
		return nil, err
	}
	input, err := entry.ToInput()
	if err != nil {
		return nil, err
	}
	execRun, err := s.validator.execSpawner.CreateBoldExecutionRun(cfg.WasmModuleRoot, uint64(cfg.StepSize), input).Await(ctx)
	if err != nil {
		return nil, err
	}
	stepLeaves := execRun.GetLeavesWithStepSize(uint64(cfg.MachineStartIndex), uint64(cfg.StepSize), cfg.NumDesiredHashes)
	result, err := stepLeaves.Await(ctx)
	if err != nil {
		return nil, err
	}
	log.Info(fmt.Sprintf("Finished gathering machine hashes for request %+v", cfg))
	// Do not save a history commitment of length 1 to the cache.
	if len(result) > 1 && s.historyCache != nil {
		if err := s.historyCache.Put(cacheKey, result); err != nil {
			if !errors.Is(err, challengecache.ErrFileAlreadyExists) {
				return nil, err
			}
		}
	}
	return result, nil
}

// CollectProof Collects osp of at a message number and OpcodeIndex .
func (s *StateManager) CollectProof(
	ctx context.Context,
	wasmModuleRoot common.Hash,
	fromBatch l2stateprovider.Batch,
	blockChallengeHeight l2stateprovider.Height,
	machineIndex l2stateprovider.OpcodeIndex,
) ([]byte, error) {
	prevBatchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(uint64(fromBatch) - 1)
	if err != nil {
		return nil, err
	}
	messageNum := (prevBatchMsgCount + arbutil.MessageIndex(blockChallengeHeight))
	entry, err := s.validator.CreateReadyValidationEntry(ctx, messageNum)
	if err != nil {
		return nil, err
	}
	input, err := entry.ToInput()
	if err != nil {
		return nil, err
	}
	execRun, err := s.validator.execSpawner.CreateExecutionRun(wasmModuleRoot, input).Await(ctx)
	if err != nil {
		return nil, err
	}
	oneStepProofPromise := execRun.GetProofAt(uint64(machineIndex))
	return oneStepProofPromise.Await(ctx)
}
