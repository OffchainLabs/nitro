// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package staker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/option"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/state-commitments/history"

	"github.com/offchainlabs/nitro/arbutil"
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
	validator            *BlockValidator
	statelessValidator   *StatelessBlockValidator
	historyCache         challengecache.HistoryCommitmentCacher
	challengeLeafHeights []l2stateprovider.Height
	validatorName        string
	checkBatchFinality   bool
	sync.RWMutex
}

type BOLDStateProviderOpt = func(b *BOLDStateProvider)

func WithoutFinalizedBatchChecks() BOLDStateProviderOpt {
	return func(b *BOLDStateProvider) {
		b.checkBatchFinality = false
	}
}

func NewBOLDStateProvider(
	blockValidator *BlockValidator,
	statelessValidator *StatelessBlockValidator,
	cacheBaseDir string,
	challengeLeafHeights []l2stateprovider.Height,
	validatorName string,
	opts ...BOLDStateProviderOpt,
) (*BOLDStateProvider, error) {
	historyCache, err := challengecache.New(cacheBaseDir)
	if err != nil {
		return nil, err
	}
	sp := &BOLDStateProvider{
		validator:            blockValidator,
		statelessValidator:   statelessValidator,
		historyCache:         historyCache,
		challengeLeafHeights: challengeLeafHeights,
		validatorName:        validatorName,
		checkBatchFinality:   true,
	}
	for _, o := range opts {
		o(sp)
	}
	return sp, nil
}

// Produces the L2 execution state to assert to after the previous assertion state.
// Returns either the state at the batch count maxInboxCount or the state maxNumberOfBlocks after previousBlockHash,
// whichever is an earlier state. If previousBlockHash is zero, this function simply returns the state at maxInboxCount.
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
	messageCount, err := s.validator.inboxTracker.GetBatchMessageCount(batchIndex)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, fmt.Errorf("%w: batch count %d", l2stateprovider.ErrChainCatchingUp, maxInboxCount)
		}
		return nil, err
	}
	if previousGlobalState != nil {
		// TODO: Use safer sub here.
		previousMessageCount, err := s.validator.inboxTracker.GetBatchMessageCount(previousGlobalState.Batch - 1)
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
			batchIndex, _, err = s.validator.inboxTracker.FindInboxBatchContainingMessage(messageCount)
			if err != nil {
				return nil, err
			}
		}
	}
	globalState, err := s.findGlobalStateFromMessageCountAndBatch(messageCount, l2stateprovider.Batch(batchIndex))
	if err != nil {
		return nil, err
	}
	// If the state we are requested to produce is neither validated nor finalized, we return ErrChainCatchingUp as an error.
	stateValidatedAndFinal, err := s.isStateValidatedAndFinal(ctx, globalState, messageCount)
	if err != nil {
		return nil, err
	}
	if !stateValidatedAndFinal {
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
		0,
		l2stateprovider.Height(maxNumberOfBlocks)+1,
		l2stateprovider.Batch(fromBatch),
		l2stateprovider.Batch(toBatch),
	)
	if err != nil {
		return nil, err
	}
	historyCommit, err := history.New(historyCommitStates)
	if err != nil {
		return nil, err
	}
	executionState.EndHistoryRoot = historyCommit.Merkle
	return executionState, nil
}

func (s *BOLDStateProvider) isStateValidatedAndFinal(
	ctx context.Context, gs validator.GoGlobalState, messageCount arbutil.MessageIndex,
) (bool, error) {
	lastValidatedGs, err := s.validator.ReadLastValidatedInfo()
	if err != nil {
		return false, err
	}
	if lastValidatedGs == nil {
		return false, ErrChainCatchingUp
	}
	stateValidated := gs.Batch <= lastValidatedGs.GlobalState.Batch
	if !s.checkBatchFinality {
		return stateValidated, nil
	}
	finalizedMessageCount, err := s.validator.inboxReader.GetFinalizedMsgCount(ctx)
	if err != nil {
		return false, err
	}
	messageCountFinalized := messageCount <= finalizedMessageCount
	return messageCountFinalized && stateValidated, nil
}

// messageCountFromGlobalState returns the corresponding message count of a global state, assuming that gs is a valid global state.
func (s *BOLDStateProvider) messageCountFromGlobalState(_ context.Context, gs protocol.GoGlobalState) (arbutil.MessageIndex, error) {
	// Start by getting the message count at the start of the batch
	var batchMessageCount arbutil.MessageIndex
	if batchMessageCount != 0 {
		var err error
		batchMessageCount, err = s.validator.inboxTracker.GetBatchMessageCount(gs.Batch - 1)
		if err != nil {
			return 0, err
		}
	}
	// Add on the PosInBatch
	return batchMessageCount + arbutil.MessageIndex(gs.PosInBatch), nil
}

func (s *BOLDStateProvider) StatesInBatchRange(
	fromHeight,
	toHeight l2stateprovider.Height,
	fromBatch,
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

	var prevBatchMsgCount arbutil.MessageIndex
	var err error
	if fromBatch == 0 {
		prevBatchMsgCount, err = s.validator.inboxTracker.GetBatchMessageCount(0)
	} else {
		prevBatchMsgCount, err = s.validator.inboxTracker.GetBatchMessageCount(uint64(fromBatch) - 1)
	}
	if err != nil {
		return nil, nil, err
	}
	executionResult, err := s.validator.streamer.ResultAtCount(prevBatchMsgCount)
	if err != nil {
		return nil, nil, err
	}
	startState := validator.GoGlobalState{
		BlockHash:  executionResult.BlockHash,
		SendRoot:   executionResult.SendRoot,
		Batch:      uint64(fromBatch),
		PosInBatch: 0,
	}
	machineHashes := make([]common.Hash, 0, totalDesiredHashes)
	states := make([]validator.GoGlobalState, 0, totalDesiredHashes)
	machineHashes = append(machineHashes, machineHash(startState))
	states = append(states, startState)

	for batch := fromBatch; batch < toBatch; batch++ {
		batchMessageCount, err := s.validator.inboxTracker.GetBatchMessageCount(uint64(batch))
		if err != nil {
			return nil, nil, err
		}
		messagesInBatch := batchMessageCount - prevBatchMsgCount

		// Obtain the states for each message in the batch.
		for i := uint64(0); i < uint64(messagesInBatch); i++ {
			msgIndex := uint64(prevBatchMsgCount) + i
			messageCount := msgIndex + 1
			executionResult, err := s.validator.streamer.ResultAtCount(arbutil.MessageIndex(messageCount))
			if err != nil {
				return nil, nil, err
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
			return nil, nil, err
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
		states = append(states, states[len(states)-1])
	}
	return machineHashes[fromHeight : toHeight+1], states[fromHeight : toHeight+1], nil
}

func machineHash(gs validator.GoGlobalState) common.Hash {
	return crypto.Keccak256Hash([]byte("Machine finished:"), gs.Hash().Bytes())
}

func (s *BOLDStateProvider) findGlobalStateFromMessageCountAndBatch(count arbutil.MessageIndex, batchIndex l2stateprovider.Batch) (validator.GoGlobalState, error) {
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
func (s *BOLDStateProvider) L2MessageStatesUpTo(
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
	items, _, err := s.StatesInBatchRange(fromHeight, to, fromBatch, toBatch)
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
	prevBatchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(uint64(cfg.FromBatch - 1))
	if err != nil {
		return nil, fmt.Errorf("could not get batch message count at %d: %w", cfg.FromBatch, err)
	}
	messageNum := (prevBatchMsgCount + arbutil.MessageIndex(cfg.BlockChallengeHeight))
	stepHeights := make([]uint64, len(cfg.StepHeights))
	for i, h := range cfg.StepHeights {
		stepHeights[i] = uint64(h)
	}
	globalState, err := s.findGlobalStateFromMessageCountAndBatch(prevBatchMsgCount, l2stateprovider.Batch((cfg.FromBatch - 1)))
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
	input, err := entry.ToInput()
	if err != nil {
		return nil, err
	}
	// TODO: Enable Redis streams.
	execRun, err := s.statelessValidator.execSpawners[0].CreateExecutionRun(cfg.WasmModuleRoot, input).Await(ctx)
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
	prevBatchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(uint64(fromBatch) - 1)
	if err != nil {
		return nil, err
	}
	messageNum := (prevBatchMsgCount + arbutil.MessageIndex(blockChallengeHeight))
	entry, err := s.statelessValidator.CreateReadyValidationEntry(ctx, messageNum)
	if err != nil {
		return nil, err
	}
	input, err := entry.ToInput()
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
	execRun, err := s.statelessValidator.execSpawners[0].CreateExecutionRun(wasmModuleRoot, input).Await(ctx)
	if err != nil {
		return nil, err
	}
	ctxCheckAlive, cancelCheckAlive := ctxWithCheckAlive(ctx, execRun)
	defer cancelCheckAlive()
	oneStepProofPromise := execRun.GetProofAt(uint64(machineIndex))
	return oneStepProofPromise.Await(ctxCheckAlive)
}
