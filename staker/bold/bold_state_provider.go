// Copyright 2023, Offchain Labs, Inc.
// For license information, see
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package bold

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/layer2-state-provider"
	"github.com/offchainlabs/nitro/bold/state-commitments/history"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/challenge-cache"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/proofenhancement"
	"github.com/offchainlabs/nitro/validator/server_arb"
)

var (
	_ l2stateprovider.ProofCollector          = (*BOLDStateProvider)(nil)
	_ l2stateprovider.L2MessageStateCollector = (*BOLDStateProvider)(nil)
	_ l2stateprovider.MachineHashCollector    = (*BOLDStateProvider)(nil)
	_ l2stateprovider.ExecutionProvider       = (*BOLDStateProvider)(nil)
)

type BOLDStateProvider struct {
	validator                *staker.BlockValidator
	statelessValidator       *staker.StatelessBlockValidator
	historyCache             challengecache.HistoryCommitmentCacher
	blockChallengeLeafHeight l2stateprovider.Height
	stateProviderConfig      *StateProviderConfig
	inboxTracker             staker.InboxTrackerInterface
	inboxStreamer            staker.TransactionStreamerInterface
	inboxReader              staker.InboxReaderInterface
	proofEnhancer            proofenhancement.ProofEnhancer
	sync.RWMutex
}

func NewBOLDStateProvider(
	blockValidator *staker.BlockValidator,
	statelessValidator *staker.StatelessBlockValidator,
	blockChallengeLeafHeight l2stateprovider.Height,
	stateProviderConfig *StateProviderConfig,
	machineHashesCachePath string,
	inboxTracker staker.InboxTrackerInterface,
	inboxStreamer staker.TransactionStreamerInterface,
	inboxReader staker.InboxReaderInterface,
	proofEnhancer proofenhancement.ProofEnhancer,
) (*BOLDStateProvider, error) {
	historyCache, err := challengecache.New(machineHashesCachePath)
	if err != nil {
		return nil, err
	}
	sp := &BOLDStateProvider{
		validator:                blockValidator,
		statelessValidator:       statelessValidator,
		historyCache:             historyCache,
		blockChallengeLeafHeight: blockChallengeLeafHeight,
		stateProviderConfig:      stateProviderConfig,
		inboxTracker:             inboxTracker,
		inboxStreamer:            inboxStreamer,
		inboxReader:              inboxReader,
		proofEnhancer:            proofEnhancer,
	}
	return sp, nil
}

// ExecutionStateAfterPreviousState Produces the L2 execution state for the next
// assertion. Returns the state at maxSeqInboxCount or blockChallengeLeafHeight
// after the previous state, whichever is earlier. If previousGlobalState is
// nil, defaults to returning the state at maxSeqInboxCount.
func (s *BOLDStateProvider) ExecutionStateAfterPreviousState(
	ctx context.Context,
	maxSeqInboxCount uint64,
	previousGlobalState protocol.GoGlobalState,
) (*protocol.ExecutionState, error) {
	if maxSeqInboxCount == 0 {
		return nil, errors.New("max inbox count cannot be zero")
	}
	batchIndex := maxSeqInboxCount
	maxNumberOfBlocks := uint64(s.blockChallengeLeafHeight)
	messageCount, err := s.inboxTracker.GetBatchMessageCount(batchIndex - 1)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, fmt.Errorf("%w: batch count %d", l2stateprovider.ErrChainCatchingUp, maxSeqInboxCount)
		}
		return nil, err
	}
	var previousMessageCount arbutil.MessageIndex
	if previousGlobalState.Batch > 0 {
		previousMessageCount, err = s.inboxTracker.GetBatchMessageCount(previousGlobalState.Batch - 1)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return nil, fmt.Errorf("%w: batch count %d", l2stateprovider.ErrChainCatchingUp, maxSeqInboxCount)
			}
			return nil, err
		}
	}
	previousMessageCount += arbutil.MessageIndex(previousGlobalState.PosInBatch)
	messageDiffBetweenBatches := messageCount - previousMessageCount
	maxMessageCount := previousMessageCount + arbutil.MessageIndex(maxNumberOfBlocks)
	if messageDiffBetweenBatches > maxMessageCount {
		messageCount = maxMessageCount
		batchIndex, _, err = s.inboxTracker.FindInboxBatchContainingMessage(messageCount)
		if err != nil {
			return nil, err
		}
	}
	globalState, err := s.findGlobalStateFromMessageCountAndBatch(messageCount, l2stateprovider.Batch(batchIndex))
	if err != nil {
		return nil, err
	}
	// If the state we are requested to produce is neither validated nor past
	// threshold, we return ErrChainCatchingUp as an error.
	stateValidatedAndMessageCountPastThreshold, err := s.isStateValidatedAndMessageCountPastThreshold(ctx, globalState, messageCount)
	if err != nil {
		return nil, err
	}
	if !stateValidatedAndMessageCountPastThreshold {
		return nil, fmt.Errorf("%w: batch count %d", l2stateprovider.ErrChainCatchingUp, maxSeqInboxCount)
	}

	executionState := &protocol.ExecutionState{
		GlobalState:   protocol.GoGlobalState(globalState),
		MachineStatus: protocol.MachineStatusFinished,
	}
	toBatch := executionState.GlobalState.Batch
	historyCommitStates, _, err := s.StatesInBatchRange(
		ctx,
		previousGlobalState,
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
	return executionState, nil
}

func (s *BOLDStateProvider) isStateValidatedAndMessageCountPastThreshold(
	ctx context.Context, gs validator.GoGlobalState, messageCount arbutil.MessageIndex,
) (bool, error) {
	if s.stateProviderConfig.CheckBatchFinality {
		finalizedMessageCount, err := s.inboxReader.GetFinalizedMsgCount(ctx)
		if err != nil {
			return false, err
		}
		if messageCount > finalizedMessageCount {
			return false, nil
		}
	}
	if s.validator == nil {
		// If we do not have a validator, we cannot check if the state is validated.
		// So we assume it is validated and return true.
		return true, nil
	}
	lastValidatedGs, err := s.validator.ReadLastValidatedInfo()
	if err != nil {
		return false, err
	}
	if lastValidatedGs == nil {
		return false, l2stateprovider.ErrChainCatchingUp
	}
	stateValidated := gs.Batch < lastValidatedGs.GlobalState.Batch || (gs.Batch == lastValidatedGs.GlobalState.Batch && gs.PosInBatch <= lastValidatedGs.GlobalState.PosInBatch)
	return stateValidated, nil
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
		prevBatchMsgCount, err = s.inboxTracker.GetBatchMessageCount(uint64(fromState.Batch) - 1)
		if err != nil {
			return nil, nil, err
		}
	}

	batchNum := fromState.Batch
	currBatchMsgCount, err := s.inboxTracker.GetBatchMessageCount(batchNum)
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
		executionResult := &execution.MessageResult{}
		if pos > 0 {
			executionResult, err = s.inboxStreamer.ResultAtMessageIndex(pos - 1)
			if err != nil {
				return nil, nil, err
			}
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
			// Otherwise, we might try to read too many batches, and hit an error that
			// the next batch isn't found.
			if uint64(len(states)) < totalDesiredHashes && batchNum < batchLimit {
				currBatchMsgCount, err = s.inboxTracker.GetBatchMessageCount(batchNum)
				if err != nil {
					return nil, nil, err
				}
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
		prevBatchMsgCount, err = s.inboxTracker.GetBatchMessageCount(uint64(batchIndex) - 1)
		if err != nil {
			return validator.GoGlobalState{}, err
		}
		if prevBatchMsgCount > count {
			return validator.GoGlobalState{}, fmt.Errorf("bad batch %v provided for message count %v as previous batch ended at message count %v", batchIndex, count, prevBatchMsgCount)
		}
	}
	if count != prevBatchMsgCount {
		batchMsgCount, err := s.inboxTracker.GetBatchMessageCount(uint64(batchIndex))
		if err != nil {
			return validator.GoGlobalState{}, err
		}
		if count > batchMsgCount {
			return validator.GoGlobalState{}, fmt.Errorf("message count %v is past end of batch %v message count %v", count, batchIndex, batchMsgCount)
		}
	}
	res := &execution.MessageResult{}
	if count > 0 {
		res, err = s.inboxStreamer.ResultAtMessageIndex(count - 1)
		if err != nil {
			return validator.GoGlobalState{}, fmt.Errorf("%s: could not check if we have result at count %d: %w", s.stateProviderConfig.ValidatorName, count, err)
		}
	}
	return validator.GoGlobalState{
		BlockHash:  res.BlockHash,
		SendRoot:   res.SendRoot,
		Batch:      uint64(batchIndex),
		PosInBatch: uint64(count - prevBatchMsgCount),
	}, nil
}

// L2MessageStatesUpTo Computes a block history commitment from a start L2
// message to an end L2 message index and up to a required batch index. The
// hashes used for this commitment are the machine hashes at each message
// number.
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

// CollectMachineHashes Collects a list of machine hashes at a message number
// based on some configuration parameters.
func (s *BOLDStateProvider) CollectMachineHashes(
	ctx context.Context, cfg *l2stateprovider.HashCollectorConfig,
) ([]common.Hash, error) {
	s.RLock()
	defer s.RUnlock()
	batchLimit := cfg.AssertionMetadata.BatchLimit
	messageNum, err := s.messageNum(cfg.AssertionMetadata, cfg.BlockChallengeHeight)
	if err != nil {
		return nil, err
	}
	// Check if we have a virtual global state.
	vs, err := s.virtualState(messageNum, batchLimit)
	if err != nil {
		return nil, err
	}
	if vs.IsSome() {
		m := server_arb.NewFinishedMachine(vs.Unwrap())
		defer m.Destroy()
		return []common.Hash{m.Hash()}, nil
	}
	stepHeights := make([]uint64, len(cfg.StepHeights))
	for i, h := range cfg.StepHeights {
		stepHeights[i] = uint64(h)
	}
	messageResult, err := s.inboxStreamer.ResultAtMessageIndex(messageNum)
	if err != nil {
		return nil, err
	}
	cacheKey := &challengecache.Key{
		RollupBlockHash: messageResult.BlockHash,
		WavmModuleRoot:  cfg.AssertionMetadata.WasmModuleRoot,
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
	input, err := entry.ToInput([]rawdb.WasmTarget{rawdb.TargetWavm})
	if err != nil {
		return nil, err
	}
	// TODO: Enable Redis streams.
	result, err := s.statelessValidator.BOLDExecutionSpawners()[0].GetMachineHashesWithStepSize(
		ctx,
		cfg.AssertionMetadata.WasmModuleRoot,
		input,
		uint64(cfg.MachineStartIndex),
		uint64(cfg.StepSize),
		cfg.NumDesiredHashes,
	)
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

// messageNum returns the message number at which the BoLD protocol should
// process machine hashes based on the AssociatedAssertionMetadata and
// chalHeight.
func (s *BOLDStateProvider) messageNum(md *l2stateprovider.AssociatedAssertionMetadata, chalHeight l2stateprovider.Height) (arbutil.MessageIndex, error) {
	var prevBatchMsgCount arbutil.MessageIndex
	bNum := md.FromState.Batch
	posInBatch := md.FromState.PosInBatch
	if bNum > 0 {
		var err error
		prevBatchMsgCount, err = s.inboxTracker.GetBatchMessageCount(uint64(bNum - 1))
		if err != nil {
			return 0, fmt.Errorf("could not get prevBatchMsgCount at %d: %w", bNum-1, err)
		}
	}
	return prevBatchMsgCount + arbutil.MessageIndex(posInBatch) + arbutil.MessageIndex(chalHeight), nil
}

// virtualState returns an optional global state.
//
// If messageNum is a virtual block or the last real block to which this
// validator's assertion committed, then this function returns a global state
// representing that virtual block's finished machine. Otherwise, it returns
// an Option.None.
//
// This can happen in the BoLD protocol when the rival block-level challenge
// edge has committed to more blocks that this validator expected for the
// current batch. In that case, the chalHeight will be a block in the virtual
// padding of the history commitment of this validator.
//
// If there is an Option.Some() retrun value, it means that callers don't need
// to actually step through a machine to produce a series of hashes, because all
// of the hashes can just be "virtual" copies of a single machine in the
// FINISHED state's hash.
func (s *BOLDStateProvider) virtualState(msgNum arbutil.MessageIndex, limit l2stateprovider.Batch) (option.Option[validator.GoGlobalState], error) {
	gs := option.None[validator.GoGlobalState]()
	limitMsgCount, err := s.inboxTracker.GetBatchMessageCount(uint64(limit) - 1)
	if err != nil {
		return gs, fmt.Errorf("could not get limitMsgCount at %d: %w", limit, err)
	}
	if msgNum >= limitMsgCount {
		result := &execution.MessageResult{}
		if limitMsgCount > 0 {
			result, err = s.inboxStreamer.ResultAtMessageIndex(limitMsgCount - 1)
			if err != nil {
				return gs, fmt.Errorf("could not get global state at limitMsgCount %d: %w", limitMsgCount, err)
			}
		}
		gs = option.Some(validator.GoGlobalState{
			BlockHash:  result.BlockHash,
			SendRoot:   result.SendRoot,
			Batch:      uint64(limit),
			PosInBatch: 0,
		})
	}
	return gs, nil
}

// CollectProof collects a one-step proof at a message number and OpcodeIndex.
func (s *BOLDStateProvider) CollectProof(
	ctx context.Context,
	assertionMetadata *l2stateprovider.AssociatedAssertionMetadata,
	blockChallengeHeight l2stateprovider.Height,
	machineIndex l2stateprovider.OpcodeIndex,
) ([]byte, error) {
	messageNum, err := s.messageNum(assertionMetadata, blockChallengeHeight)
	if err != nil {
		return nil, err
	}
	// Check if we have a virtual global state.
	vs, err := s.virtualState(messageNum, assertionMetadata.BatchLimit)
	if err != nil {
		return nil, err
	}
	if vs.IsSome() {
		m := server_arb.NewFinishedMachine(vs.Unwrap())
		defer m.Destroy()
		log.Info(
			"Getting machine OSP from virtual state",
			"fromBatch", assertionMetadata.FromState.Batch,
			"fromPosInBatch", assertionMetadata.FromState.PosInBatch,
			"blockChallengeHeight", blockChallengeHeight,
			"messageNum", messageNum,
			"machineIndex", machineIndex,
		)
		return m.ProveNextStep(), nil
	}
	entry, err := s.statelessValidator.CreateReadyValidationEntry(ctx, messageNum)
	if err != nil {
		return nil, err
	}
	input, err := entry.ToInput([]rawdb.WasmTarget{rawdb.TargetWavm})
	if err != nil {
		return nil, err
	}
	log.Info(
		"Getting machine OSP",
		"fromBatch", assertionMetadata.FromState.Batch,
		"fromPosInBatch", assertionMetadata.FromState.PosInBatch,
		"blockChallengeHeight", blockChallengeHeight,
		"messageNum", messageNum,
		"machineIndex", machineIndex,
		"startState", fmt.Sprintf("%+v", input.StartState),
	)
	baseProof, err := s.statelessValidator.BOLDExecutionSpawners()[0].GetProofAt(
		ctx,
		assertionMetadata.WasmModuleRoot,
		input,
		uint64(machineIndex),
	)
	if err != nil {
		return nil, err
	}

	// Apply proof enhancement if configured
	if s.proofEnhancer != nil {
		return s.proofEnhancer.EnhanceProof(ctx, messageNum, baseProof)
	}

	return baseProof, nil
}
