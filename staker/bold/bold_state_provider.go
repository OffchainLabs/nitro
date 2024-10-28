// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package bold

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
	batchIndex := maxInboxCount
	messageCount, err := s.statelessValidator.InboxTracker().GetBatchMessageCount(batchIndex - 1)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, fmt.Errorf("%w: batch count %d", l2stateprovider.ErrChainCatchingUp, maxInboxCount)
		}
		return nil, err
	}
	if previousGlobalState != nil {
		var previousMessageCount arbutil.MessageIndex
		if previousGlobalState.Batch > 0 {
			previousMessageCount, err = s.statelessValidator.InboxTracker().GetBatchMessageCount(previousGlobalState.Batch - 1)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					return nil, fmt.Errorf("%w: batch count %d", l2stateprovider.ErrChainCatchingUp, maxInboxCount)
				}
				return nil, err
			}
		}
		previousMessageCount += arbutil.MessageIndex(previousGlobalState.PosInBatch)
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

	var previousGlobalStateOrDefault protocol.GoGlobalState
	if previousGlobalState != nil {
		previousGlobalStateOrDefault = *previousGlobalState
	}
	toBatch := executionState.GlobalState.Batch
	historyCommitStates, _, err := s.StatesInBatchRange(
		ctx,
		previousGlobalStateOrDefault,
		toBatch,
		l2stateprovider.Height(maxNumberOfBlocks),
	)
	if err != nil {
		return nil, err
	}
	historyCommit, err := history.NewCommitment(historyCommitStates, maxNumberOfBlocks+1)
	if err != nil {
		return nil, err
	}
	executionState.EndHistoryRoot = historyCommit.Merkle
	fmt.Printf("ExecutionStateAfterPreviousState for previous state batch %v pos %v got end batch %v pos %v last leaf %v hash %v\n", previousGlobalStateOrDefault.Batch, previousGlobalStateOrDefault.PosInBatch, executionState.GlobalState.Batch, executionState.GlobalState.PosInBatch, historyCommitStates[len(historyCommitStates)-1], executionState.EndHistoryRoot)
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
	fromState protocol.GoGlobalState,
	batchLimit uint64,
	toHeight l2stateprovider.Height,
) ([]common.Hash, []validator.GoGlobalState, error) {
	// Check the integrity of the arguments.
	if batchLimit < fromState.Batch || (batchLimit == fromState.Batch && fromState.PosInBatch > 0) {
		return nil, nil, fmt.Errorf("batch limit %v cannot be less than from batch %v", batchLimit, fromState.Batch)
	}
	// Compute the total desired hashes from this request.
	totalDesiredHashes := uint64(toHeight + 1)
	machineHashes := make([]common.Hash, 0)
	states := make([]validator.GoGlobalState, 0)

	var prevBatchMsgCount arbutil.MessageIndex
	var err error
	if fromState.Batch > 0 {
		prevBatchMsgCount, err = s.statelessValidator.InboxTracker().GetBatchMessageCount(uint64(fromState.Batch) - 1)
		if err != nil {
			return nil, nil, err
		}
	}

	batchNum := fromState.Batch
	currBatchMsgCount, err := s.statelessValidator.InboxTracker().GetBatchMessageCount(batchNum)
	if err != nil {
		return nil, nil, err
	}
	posInBatch := fromState.PosInBatch
	initialPos := prevBatchMsgCount + arbutil.MessageIndex(posInBatch)
	if initialPos >= currBatchMsgCount {
		return nil, nil, fmt.Errorf("initial position %v is past end of from batch %v message count %v", initialPos, batchNum, currBatchMsgCount)
	}
	for pos := initialPos; uint64(len(states)) < totalDesiredHashes; pos++ {
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
		if batchNum >= batchLimit {
			break
		}
		// Check if the next message is in the next batch.
		if uint64(pos+1) == uint64(currBatchMsgCount) {
			posInBatch = 0
			batchNum++
			// Only get the next batch metadata if it'll be needed.
			// Otherwise, we might try to read too many batches, and hit an error that the next batch isn't found.
			if uint64(len(states)) < totalDesiredHashes && batchNum < batchLimit {
				currBatchMsgCount, err = s.statelessValidator.InboxTracker().GetBatchMessageCount(batchNum)
				if err != nil {
					return nil, nil, err
				}
			}
		} else {
			posInBatch++
		}
	}
	fmt.Printf("got states from batch %v pos %v up to batch %v height %v\n", fromState.Batch, fromState.PosInBatch, batchLimit, toHeight)
	println("----- states -----")
	for i, state := range states {
		fmt.Printf("batch %v pos %v hash %v\n", state.Batch, state.PosInBatch, machineHashes[i])
	}
	println("------------------")
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
			return validator.GoGlobalState{}, fmt.Errorf("bad batch %v provided for message count %v as previous batch ended at message count %v", batchIndex, count, prevBatchMsgCount)
		}
	}
	if count != prevBatchMsgCount {
		batchMsgCount, err := s.statelessValidator.InboxTracker().GetBatchMessageCount(uint64(batchIndex))
		if err != nil {
			return validator.GoGlobalState{}, err
		}
		if count > batchMsgCount {
			return validator.GoGlobalState{}, fmt.Errorf("message count %v is past end of batch %v message count %v", count, batchIndex, batchMsgCount)
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
	fromState protocol.GoGlobalState,
	batchLimit l2stateprovider.Batch,
	toHeight option.Option[l2stateprovider.Height],
) ([]common.Hash, error) {
	var to l2stateprovider.Height
	if !toHeight.IsNone() {
		to = toHeight.Unwrap()
	} else {
		to = s.blockChallengeLeafHeight
	}
	items, _, err := s.StatesInBatchRange(ctx, fromState, uint64(batchLimit), to)
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
	var prevBatchMsgCount arbutil.MessageIndex
	if cfg.FromState.Batch > 0 {
		var err error
		prevBatchMsgCount, err = s.statelessValidator.InboxTracker().GetBatchMessageCount(uint64(cfg.FromState.Batch - 1))
		if err != nil {
			return nil, fmt.Errorf("could not get batch message count at %d: %w", cfg.FromState.Batch-1, err)
		}
	}
	// cfg.BlockChallengeHeight is the index of the last correct block, before the block we're challenging.
	messageNum := prevBatchMsgCount + arbutil.MessageIndex(cfg.FromState.PosInBatch) + arbutil.MessageIndex(cfg.BlockChallengeHeight)
	stepHeights := make([]uint64, len(cfg.StepHeights))
	for i, h := range cfg.StepHeights {
		stepHeights[i] = uint64(h)
	}
	messageResult, err := s.statelessValidator.InboxStreamer().ResultAtCount(arbutil.MessageIndex(messageNum + 1))
	if err != nil {
		return nil, err
	}
	cacheKey := &challengecache.Key{
		RollupBlockHash: messageResult.BlockHash,
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
	// TODO: Enable Redis streams.
	execRun, err := s.statelessValidator.ExecutionSpawners()[0].CreateExecutionRun(cfg.WasmModuleRoot, input).Await(ctx)
	if err != nil {
		return nil, err
	}
	defer execRun.Close()
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
	fromState protocol.GoGlobalState,
	wasmModuleRoot common.Hash,
	blockChallengeHeight l2stateprovider.Height,
	machineIndex l2stateprovider.OpcodeIndex,
) ([]byte, error) {
	prevBatchMsgCount, err := s.statelessValidator.InboxTracker().GetBatchMessageCount(uint64(fromState.Batch) - 1)
	if err != nil {
		return nil, err
	}
	// blockChallengeHeight is the index of the last correct block, before the block we're challenging.
	messageNum := prevBatchMsgCount + arbutil.MessageIndex(fromState.PosInBatch) + arbutil.MessageIndex(blockChallengeHeight)
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
		"fromBatch", fromState.Batch,
		"fromPosInBatch", fromState.PosInBatch,
		"prevBatchMsgCount", prevBatchMsgCount,
		"blockChallengeHeight", blockChallengeHeight,
		"messageNum", messageNum,
		"startState", fmt.Sprintf("%+v", input.StartState),
	)
	execRun, err := s.statelessValidator.ExecutionSpawners()[0].CreateExecutionRun(wasmModuleRoot, input).Await(ctx)
	if err != nil {
		return nil, err
	}
	defer execRun.Close()
	ctxCheckAlive, cancelCheckAlive := ctxWithCheckAlive(ctx, execRun)
	defer cancelCheckAlive()
	oneStepProofPromise := execRun.GetProofAt(uint64(machineIndex))
	return oneStepProofPromise.Await(ctxCheckAlive)
}
