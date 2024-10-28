// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package boldstaker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	protocol "github.com/offchainlabs/bold/chain-abstraction"
	"github.com/offchainlabs/bold/containers/option"
	l2stateprovider "github.com/offchainlabs/bold/layer2-state-provider"
	"github.com/offchainlabs/bold/state-commitments/history"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/staker"
	challengecache "github.com/offchainlabs/nitro/staker/challenge-cache"
	"github.com/offchainlabs/nitro/validator"
)

var (
	_ l2stateprovider.ProofCollector          = (*BOLDStateProvider)(nil)
	_ l2stateprovider.L2MessageStateCollector = (*BOLDStateProvider)(nil)
	_ l2stateprovider.MachineHashCollector    = (*BOLDStateProvider)(nil)
	_ l2stateprovider.ExecutionProvider       = (*BOLDStateProvider)(nil)
)

var executionNodeOfflineGauge = metrics.NewRegisteredGauge("arb/state_provider/execution_node_offline", nil)

var (
	ErrChainCatchingUp = errors.New("chain catching up")
)

type BOLDStateProvider struct {
	validator                *staker.BlockValidator
	statelessValidator       *staker.StatelessBlockValidator
	historyCache             challengecache.HistoryCommitmentCacher
	blockChallengeLeafHeight l2stateprovider.Height
	stateProviderConfig      *StateProviderConfig
	sync.RWMutex
}

func NewBOLDStateProvider(
	blockValidator *staker.BlockValidator,
	statelessValidator *staker.StatelessBlockValidator,
	blockChallengeLeafHeight l2stateprovider.Height,
	stateProviderConfig *StateProviderConfig,
) (*BOLDStateProvider, error) {
	historyCache, err := challengecache.New(stateProviderConfig.MachineLeavesCachePath)
	if err != nil {
		return nil, err
	}
	sp := &BOLDStateProvider{
		validator:                blockValidator,
		statelessValidator:       statelessValidator,
		historyCache:             historyCache,
		blockChallengeLeafHeight: blockChallengeLeafHeight,
		stateProviderConfig:      stateProviderConfig,
	}
	return sp, nil
}

// ExecutionStateAfterPreviousState Produces the L2 execution state for the next assertion.
// Returns the state at maxInboxCount or maxNumberOfBlocks after the previous state, whichever is earlier.
// If previousGlobalState is nil, defaults to returning the state at maxInboxCount.
// TODO: Check the block validator has validated the execution state we are proposing.
func (s *BOLDStateProvider) ExecutionStateAfterPreviousState(
	ctx context.Context,
	maxInboxCount uint64,
	previousGlobalState *protocol.GoGlobalState,
	maxNumberOfBlocks uint64,
) (*protocol.ExecutionState, error) {
	if maxInboxCount == 0 {
		return nil, errors.New("max inbox count cannot be zero")
	}
	batchIndex := maxInboxCount - 1
	messageCount, err := s.statelessValidator.InboxTracker().GetBatchMessageCount(batchIndex)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, fmt.Errorf("%w: batch count %d", l2stateprovider.ErrChainCatchingUp, maxInboxCount)
		}
		return nil, err
	}
	if previousGlobalState != nil {
		// TODO: Use safer sub here.
		previousMessageCount, err := s.statelessValidator.InboxTracker().GetBatchMessageCount(previousGlobalState.Batch - 1)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return nil, fmt.Errorf("%w: batch count %d", l2stateprovider.ErrChainCatchingUp, maxInboxCount)
			}
			return nil, err
		}
		messageDiffBetweenBatches := messageCount - previousMessageCount
		maxMessageCount := previousMessageCount + arbutil.MessageIndex(maxNumberOfBlocks)
		if messageDiffBetweenBatches > maxMessageCount {
			messageCount = maxMessageCount
			batchIndex, _, err = s.statelessValidator.InboxTracker().FindInboxBatchContainingMessage(messageCount)
			if err != nil {
				return nil, err
			}
		}
	}
	globalState, err := s.findGlobalStateFromMessageCountAndBatch(messageCount, l2stateprovider.Batch(batchIndex))
	if err != nil {
		return nil, err
	}
	// If the state we are requested to produce is neither validated nor past threshold, we return ErrChainCatchingUp as an error.
	stateValidatedAndMessageCountPastThreshold, err := s.isStateValidatedAndMessageCountPastThreshold(ctx, globalState, messageCount)
	if err != nil {
		return nil, err
	}
	if !stateValidatedAndMessageCountPastThreshold {
		return nil, fmt.Errorf("%w: batch count %d", l2stateprovider.ErrChainCatchingUp, maxInboxCount)
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

	fromBatch := uint64(0)
	if previousGlobalState != nil {
		fromBatch = previousGlobalState.Batch
	}
	toBatch := executionState.GlobalState.Batch
	historyCommitStates, _, err := s.StatesInBatchRange(
		ctx,
		0,
		l2stateprovider.Height(maxNumberOfBlocks)+1,
		l2stateprovider.Batch(fromBatch),
		l2stateprovider.Batch(toBatch),
	)
	if err != nil {
		return nil, err
	}
	historyCommit, err := history.NewCommitment(historyCommitStates, maxNumberOfBlocks+1)
	if err != nil {
		return nil, err
	}
	executionState.EndHistoryRoot = historyCommit.Merkle
	return executionState, nil
}

func (s *BOLDStateProvider) isStateValidatedAndMessageCountPastThreshold(
	ctx context.Context, gs validator.GoGlobalState, messageCount arbutil.MessageIndex,
) (bool, error) {
	if s.validator == nil {
		// If we do not have a validator, we cannot check if the state is validated.
		// So we assume it is validated and return true.
		// This is a dangerous option, only users
		return true, nil
	}
	lastValidatedGs, err := s.validator.ReadLastValidatedInfo()
	if err != nil {
		return false, err
	}
	if lastValidatedGs == nil {
		return false, ErrChainCatchingUp
	}
	stateValidated := gs.Batch <= lastValidatedGs.GlobalState.Batch
	if !s.stateProviderConfig.CheckBatchFinality {
		return stateValidated, nil
	}
	finalizedMessageCount, err := s.statelessValidator.InboxReader().GetFinalizedMsgCount(ctx)
	if err != nil {
		return false, err
	}
	messageCountFinalized := messageCount <= finalizedMessageCount
	return messageCountFinalized && stateValidated, nil
}

func (s *BOLDStateProvider) StatesInBatchRange(
	ctx context.Context,
	fromHeight l2stateprovider.Height,
	toHeight l2stateprovider.Height,
	fromBatch l2stateprovider.Batch,
	toBatch l2stateprovider.Batch,
) ([]common.Hash, []validator.GoGlobalState, error) {
	// Check the integrity of the arguments.
	if fromBatch >= toBatch {
		return nil, nil, fmt.Errorf("from batch %v cannot be greater than or equal to batch %v", fromBatch, toBatch)
	}
	if fromHeight > toHeight {
		return nil, nil, fmt.Errorf("from height %v cannot be greater than to height %v", fromHeight, toHeight)
	}
	// Compute the total desired hashes from this request.
	totalDesiredHashes := (toHeight - fromHeight) + 1
	machineHashes := make([]common.Hash, 0, totalDesiredHashes)
	states := make([]validator.GoGlobalState, 0, totalDesiredHashes)

	var prevBatchMsgCount arbutil.MessageIndex
	var err error
	batchNum, found, err := s.statelessValidator.InboxTracker().FindInboxBatchContainingMessage(arbutil.MessageIndex(fromHeight))
	if err != nil {
		return nil, nil, err
	}
	if !found {
		return nil, nil, fmt.Errorf("could not find batch containing message %d", fromHeight)
	}
	if batchNum == 0 {
		prevBatchMsgCount = 0
	} else {
		prevBatchMsgCount, err = s.statelessValidator.InboxTracker().GetBatchMessageCount(batchNum - 1)
	}
	if err != nil {
		return nil, nil, err
	}
	currBatchMsgCount, err := s.statelessValidator.InboxTracker().GetBatchMessageCount(batchNum)
	if err != nil {
		return nil, nil, err
	}
	posInBatch := uint64(fromHeight) - uint64(prevBatchMsgCount)
	for pos := fromHeight; pos <= toHeight; pos++ {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		executionResult, err := s.statelessValidator.InboxStreamer().ResultAtCount(arbutil.MessageIndex(pos))
		if err != nil {
			return nil, nil, err
		}
		state := validator.GoGlobalState{
			BlockHash:  executionResult.BlockHash,
			SendRoot:   executionResult.SendRoot,
			Batch:      batchNum,
			PosInBatch: posInBatch,
		}
		states = append(states, state)
		machineHashes = append(machineHashes, machineHash(state))
		if uint64(pos) == uint64(currBatchMsgCount) {
			posInBatch = 0
			batchNum++
			currBatchMsgCount, err = s.statelessValidator.InboxTracker().GetBatchMessageCount(batchNum)
			if err != nil {
				return nil, nil, err
			}
		} else {
			posInBatch++
		}
	}
	return machineHashes, states, nil
}

func machineHash(gs validator.GoGlobalState) common.Hash {
	return crypto.Keccak256Hash([]byte("Machine finished:"), gs.Hash().Bytes())
}

func (s *BOLDStateProvider) findGlobalStateFromMessageCountAndBatch(count arbutil.MessageIndex, batchIndex l2stateprovider.Batch) (validator.GoGlobalState, error) {
	var prevBatchMsgCount arbutil.MessageIndex
	var err error
	if batchIndex > 0 {
		prevBatchMsgCount, err = s.statelessValidator.InboxTracker().GetBatchMessageCount(uint64(batchIndex) - 1)
		if err != nil {
			return validator.GoGlobalState{}, err
		}
		if prevBatchMsgCount > count {
			return validator.GoGlobalState{}, errors.New("bad batch provided")
		}
	}
	res, err := s.statelessValidator.InboxStreamer().ResultAtCount(count)
	if err != nil {
		return validator.GoGlobalState{}, fmt.Errorf("%s: could not check if we have result at count %d: %w", s.stateProviderConfig.ValidatorName, count, err)
	}
	return validator.GoGlobalState{
		BlockHash:  res.BlockHash,
		SendRoot:   res.SendRoot,
		Batch:      uint64(batchIndex),
		PosInBatch: uint64(count - prevBatchMsgCount),
	}, nil
}

// L2MessageStatesUpTo Computes a block history commitment from a
// start L2 message to an end L2 message index and up to a required
// batch index. The hashes used for this commitment are the machine
// hashes at each message number.
func (s *BOLDStateProvider) L2MessageStatesUpTo(
	ctx context.Context,
	fromHeight l2stateprovider.Height,
	toHeight option.Option[l2stateprovider.Height],
	fromBatch,
	toBatch l2stateprovider.Batch,
) ([]common.Hash, error) {
	var to l2stateprovider.Height
	if !toHeight.IsNone() {
		to = toHeight.Unwrap()
	} else {
		to = s.blockChallengeLeafHeight
	}
	items, _, err := s.StatesInBatchRange(ctx, fromHeight, to, fromBatch, toBatch)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// CollectMachineHashes Collects a list of machine hashes at a message number based on some configuration parameters.
func (s *BOLDStateProvider) CollectMachineHashes(
	ctx context.Context, cfg *l2stateprovider.HashCollectorConfig,
) ([]common.Hash, error) {
	s.Lock()
	defer s.Unlock()
	prevBatchMsgCount, err := s.statelessValidator.InboxTracker().GetBatchMessageCount(uint64(cfg.FromBatch - 1))
	if err != nil {
		return nil, fmt.Errorf("could not get batch message count at %d: %w", cfg.FromBatch, err)
	}
	messageNum := prevBatchMsgCount + arbutil.MessageIndex(cfg.BlockChallengeHeight)
	stepHeights := make([]uint64, len(cfg.StepHeights))
	for i, h := range cfg.StepHeights {
		stepHeights[i] = uint64(h)
	}
	globalState, err := s.findGlobalStateFromMessageCountAndBatch(prevBatchMsgCount, cfg.FromBatch-1)
	if err != nil {
		return nil, err
	}
	cacheKey := &challengecache.Key{
		RollupBlockHash: globalState.BlockHash,
		WavmModuleRoot:  cfg.WasmModuleRoot,
		MessageHeight:   uint64(messageNum),
		StepHeights:     stepHeights,
	}
	if s.historyCache != nil {
		cachedRoots, err := s.historyCache.Get(cacheKey, cfg.NumDesiredHashes)
		switch {
		case err == nil:
			log.Info(
				"In collect machine hashes",
				"cfg", fmt.Sprintf("%+v", cfg),
				"firstHash", fmt.Sprintf("%#x", cachedRoots[0]),
				"lastHash", fmt.Sprintf("%#x", cachedRoots[len(cachedRoots)-1]),
			)
			return cachedRoots, nil
		case !errors.Is(err, challengecache.ErrNotFoundInCache):
			return nil, err
		}
	}
	entry, err := s.statelessValidator.CreateReadyValidationEntry(ctx, messageNum)
	if err != nil {
		return nil, err
	}
	input, err := entry.ToInput([]ethdb.WasmTarget{rawdb.TargetWavm})
	if err != nil {
		return nil, err
	}
	execRun, err := s.statelessValidator.ExecutionSpawners()[0].CreateExecutionRun(cfg.WasmModuleRoot, input).Await(ctx)
	if err != nil {
		return nil, err
	}
	ctxCheckAlive, cancelCheckAlive := ctxWithCheckAlive(ctx, execRun)
	defer cancelCheckAlive()
	stepLeaves := execRun.GetMachineHashesWithStepSize(uint64(cfg.MachineStartIndex), uint64(cfg.StepSize), cfg.NumDesiredHashes)
	result, err := stepLeaves.Await(ctxCheckAlive)
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

// CtxWithCheckAlive Creates a context with a check alive routine
// that will cancel the context if the check alive routine fails.
func ctxWithCheckAlive(ctxIn context.Context, execRun validator.ExecutionRun) (context.Context, context.CancelFunc) {
	// Create a context that will cancel if the check alive routine fails.
	// This is to ensure that we do not have the validator froze indefinitely if the execution run
	// is no longer alive.
	ctx, cancel := context.WithCancel(ctxIn)
	// Create a context with cancel, so that we can cancel the check alive routine
	// once the calling function returns.
	ctxCheckAlive, cancelCheckAlive := context.WithCancel(ctxIn)
	go func() {
		// Call cancel so that the calling function is canceled if the check alive routine fails/returns.
		defer cancel()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctxCheckAlive.Done():
				return
			case <-ticker.C:
				// Create a context with a timeout, so that the check alive routine does not run indefinitely.
				ctxCheckAliveWithTimeout, cancelCheckAliveWithTimeout := context.WithTimeout(ctxCheckAlive, 5*time.Second)
				err := execRun.CheckAlive(ctxCheckAliveWithTimeout)
				if err != nil {
					executionNodeOfflineGauge.Inc(1)
					cancelCheckAliveWithTimeout()
					return
				}
				cancelCheckAliveWithTimeout()
			}
		}
	}()
	return ctx, cancelCheckAlive
}

// CollectProof Collects osp of at a message number and OpcodeIndex .
func (s *BOLDStateProvider) CollectProof(
	ctx context.Context,
	wasmModuleRoot common.Hash,
	fromBatch l2stateprovider.Batch,
	blockChallengeHeight l2stateprovider.Height,
	machineIndex l2stateprovider.OpcodeIndex,
) ([]byte, error) {
	prevBatchMsgCount, err := s.statelessValidator.InboxTracker().GetBatchMessageCount(uint64(fromBatch) - 1)
	if err != nil {
		return nil, err
	}
	messageNum := prevBatchMsgCount + arbutil.MessageIndex(blockChallengeHeight)
	entry, err := s.statelessValidator.CreateReadyValidationEntry(ctx, messageNum)
	if err != nil {
		return nil, err
	}
	input, err := entry.ToInput([]ethdb.WasmTarget{rawdb.TargetWavm})
	if err != nil {
		return nil, err
	}
	log.Info(
		"Getting machine OSP",
		"fromBatch", fromBatch,
		"prevBatchMsgCount", prevBatchMsgCount,
		"blockChallengeHeight", blockChallengeHeight,
		"messageNum", messageNum,
		"startState", fmt.Sprintf("%+v", input.StartState),
	)
	execRun, err := s.statelessValidator.ExecutionSpawners()[0].CreateExecutionRun(wasmModuleRoot, input).Await(ctx)
	if err != nil {
		return nil, err
	}
	ctxCheckAlive, cancelCheckAlive := ctxWithCheckAlive(ctx, execRun)
	defer cancelCheckAlive()
	oneStepProofPromise := execRun.GetProofAt(uint64(machineIndex))
	return oneStepProofPromise.Await(ctxCheckAlive)
}
