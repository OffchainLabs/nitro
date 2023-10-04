// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package staker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

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

// Defines the ABI encoding structure for submission of prefix proofs to the protocol contracts
var (
	b32Arr, _ = abi.NewType("bytes32[]", "", nil)
	// ProofArgs for submission to the protocol.
	ProofArgs = abi.Arguments{
		{Type: b32Arr, Name: "prefixExpansion"},
		{Type: b32Arr, Name: "prefixProof"},
	}
)

var (
	ErrChainCatchingUp = errors.New("chain catching up")
)

type StateManager struct {
	validator            *StatelessBlockValidator
	historyCache         challengecache.HistoryCommitmentCacher
	challengeLeafHeights []l2stateprovider.Height
	sync.RWMutex
}

func NewStateManager(val *StatelessBlockValidator, cacheBaseDir string, challengeLeafHeights []l2stateprovider.Height) (*StateManager, error) {
	historyCache := challengecache.New(cacheBaseDir)
	return &StateManager{
		validator:            val,
		historyCache:         historyCache,
		challengeLeafHeights: challengeLeafHeights,
	}, nil
}

// ExecutionStateMsgCount If the state manager locally has this validated execution state.
// Returns ErrNoExecutionState if not found, or ErrChainCatchingUp if not yet
// validated / syncing.
func (s *StateManager) ExecutionStateMsgCount(ctx context.Context, state *protocol.ExecutionState) (uint64, error) {
	if state.GlobalState.PosInBatch != 0 {
		return 0, fmt.Errorf("position in batch must be zero, but got %d", state.GlobalState.PosInBatch)
	}
	if state.GlobalState.Batch == 1 && state.GlobalState.PosInBatch == 0 {
		return 1, nil
	}
	batch := state.GlobalState.Batch - 1
	messageCount, err := s.validator.inboxTracker.GetBatchMessageCount(batch)
	if err != nil {
		return 0, err
	}
	validatedExecutionState, err := s.executionStateAtMessageNumberImpl(ctx, uint64(messageCount)-1)
	if err != nil {
		return 0, err
	}
	if validatedExecutionState.GlobalState.Batch < batch {
		return 0, ErrChainCatchingUp
	}
	res, err := s.validator.streamer.ResultAtCount(messageCount)
	if err != nil {
		return 0, err
	}
	if res.BlockHash != state.GlobalState.BlockHash || res.SendRoot != state.GlobalState.SendRoot {
		return 0, l2stateprovider.ErrNoExecutionState
	}
	return uint64(messageCount), nil
}

// ExecutionStateAtMessageNumber Produces the l2 state to assert at the message number specified.
// Makes sure that PosInBatch is always 0
func (s *StateManager) ExecutionStateAtMessageNumber(ctx context.Context, messageNumber uint64) (*protocol.ExecutionState, error) {
	executionState, err := s.executionStateAtMessageNumberImpl(ctx, messageNumber)
	if err != nil {
		return nil, err
	}
	if executionState.GlobalState.PosInBatch != 0 {
		executionState.GlobalState.Batch++
		executionState.GlobalState.PosInBatch = 0
	}
	return executionState, nil
}

func (s *StateManager) executionStateAtMessageNumberImpl(_ context.Context, messageNumber uint64) (*protocol.ExecutionState, error) {
	batch, err := s.findBatchAfterMessageCount(arbutil.MessageIndex(messageNumber))
	if err != nil {
		return &protocol.ExecutionState{}, err
	}
	batchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(batch)
	if err != nil {
		return &protocol.ExecutionState{}, err
	}
	if batchMsgCount <= arbutil.MessageIndex(messageNumber) {
		batch++
	}
	globalState, err := s.getInfoAtMessageCountAndBatch(arbutil.MessageIndex(messageNumber), batch)
	if err != nil {
		return &protocol.ExecutionState{}, err
	}
	if globalState.PosInBatch != 0 {
		return &protocol.ExecutionState{}, fmt.Errorf("position in batch must be zero, but got %d", globalState.PosInBatch)
	}
	return &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState(globalState),
		// Batches with position 0 consume all the messages from the previous batch, so their machine status is finished.
		MachineStatus: protocol.MachineStatusFinished,
	}, nil
}

func (s *StateManager) statesUpTo(blockStart uint64, blockEnd uint64, nextBatchCount uint64) ([]common.Hash, error) {
	if blockEnd < blockStart {
		return nil, fmt.Errorf("end block %v is less than start block %v", blockEnd, blockStart)
	}
	batch, err := s.findBatchAfterMessageCount(arbutil.MessageIndex(blockStart))
	if err != nil {
		return nil, err
	}
	if batch == 0 {
		// Genesis cannot be validated. If genesis is passed in, we start from batch index 1.
		batch += 1
	}
	// The size is the number of elements being committed to. For example, if the height is 7, there will
	// be 8 elements being committed to from [0, 7] inclusive.
	desiredStatesLen := int(blockEnd - blockStart + 1)
	var stateRoots []common.Hash
	var lastStateRoot common.Hash

	// Genesis cannot be validated. If genesis is passed in, we start from block number 1.
	if blockStart == 0 {
		blockStart += 1
	}
	for i := blockStart; i <= blockEnd; i++ {
		batchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(batch)
		if err != nil {
			return nil, err
		}
		if batchMsgCount <= arbutil.MessageIndex(i) {
			batch++
		}
		gs, err := s.getInfoAtMessageCountAndBatch(arbutil.MessageIndex(i), batch)
		if err != nil {
			return nil, err
		}
		if gs.Batch >= nextBatchCount {
			if gs.Batch > nextBatchCount || gs.PosInBatch > 0 {
				return nil, fmt.Errorf("overran next batch count %v with global state batch %v position %v", nextBatchCount, gs.Batch, gs.PosInBatch)
			}
			break
		}
		stateRoot := crypto.Keccak256Hash([]byte("Machine finished:"), gs.Hash().Bytes())
		stateRoots = append(stateRoots, stateRoot)
		lastStateRoot = stateRoot
	}
	for len(stateRoots) < desiredStatesLen {
		stateRoots = append(stateRoots, lastStateRoot)
	}
	return stateRoots, nil
}

func (s *StateManager) findBatchAfterMessageCount(msgCount arbutil.MessageIndex) (uint64, error) {
	if msgCount == 0 {
		return 0, nil
	}
	low := uint64(0)
	batchCount, err := s.validator.inboxTracker.GetBatchCount()
	if err != nil {
		return 0, err
	}
	high := batchCount
	for {
		// Binary search invariants:
		//   - messageCount(high) >= msgCount
		//   - messageCount(low-1) < msgCount
		//   - high >= low
		if high < low {
			return 0, fmt.Errorf("when attempting to find batch for message count %v high %v < low %v", msgCount, high, low)
		}
		mid := (low + high) / 2
		batchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(mid)
		if err != nil {
			// TODO: There is a circular dep with the error in inbox_tracker.go, we
			// should move it somewhere else and use errors.Is.
			if strings.Contains(err.Error(), "accumulator not found") {
				high = mid
			} else {
				return 0, fmt.Errorf("failed to get batch metadata while binary searching: %w", err)
			}
		}
		if batchMsgCount < msgCount {
			low = mid + 1
		} else if batchMsgCount == msgCount {
			return mid + 1, nil
		} else if mid == low { // batchMsgCount > msgCount
			return mid, nil
		} else { // batchMsgCount > msgCount
			high = mid
		}
	}
}

func (s *StateManager) getInfoAtMessageCountAndBatch(messageCount arbutil.MessageIndex, batch uint64) (validator.GoGlobalState, error) {
	globalState, err := s.findGlobalStateFromMessageCountAndBatch(messageCount, batch)
	if err != nil {
		return validator.GoGlobalState{}, err
	}
	return globalState, nil
}

func (s *StateManager) findGlobalStateFromMessageCountAndBatch(count arbutil.MessageIndex, batch uint64) (validator.GoGlobalState, error) {
	var prevBatchMsgCount arbutil.MessageIndex
	var err error
	if batch > 0 {
		prevBatchMsgCount, err = s.validator.inboxTracker.GetBatchMessageCount(batch - 1)
		if err != nil {
			return validator.GoGlobalState{}, err
		}
		if prevBatchMsgCount > count {
			return validator.GoGlobalState{}, errors.New("bad batch provided")
		}
	}
	res, err := s.validator.streamer.ResultAtCount(count)
	if err != nil {
		return validator.GoGlobalState{}, err
	}
	return validator.GoGlobalState{
		BlockHash:  res.BlockHash,
		SendRoot:   res.SendRoot,
		Batch:      batch,
		PosInBatch: uint64(count - prevBatchMsgCount),
	}, nil
}

// L2MessageStatesUpTo Computes a block history commitment from a start L2 message to an end L2 message index
// and up to a required batch index. The hashes used for this commitment are the machine hashes
// at each message number.
func (s *StateManager) L2MessageStatesUpTo(
	_ context.Context,
	from l2stateprovider.Height,
	upTo option.Option[l2stateprovider.Height],
	batch l2stateprovider.Batch,
) ([]common.Hash, error) {
	var to l2stateprovider.Height
	if !upTo.IsNone() {
		to = upTo.Unwrap()
	} else {
		blockChallengeLeafHeight := s.challengeLeafHeights[0]
		to = blockChallengeLeafHeight
	}
	items, err := s.statesUpTo(uint64(from), uint64(to), uint64(batch))
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
	cacheKey := &challengecache.Key{
		WavmModuleRoot: cfg.WasmModuleRoot,
		MessageHeight:  protocol.Height(cfg.MessageNumber),
		StepHeights:    cfg.StepHeights,
	}
	cachedRoots, err := s.historyCache.Get(cacheKey, cfg.NumDesiredHashes)
	switch {
	case err == nil:
		return cachedRoots, nil
	case !errors.Is(err, challengecache.ErrNotFoundInCache):
		return nil, err
	}
	entry, err := s.validator.CreateReadyValidationEntry(ctx, arbutil.MessageIndex(cfg.MessageNumber))
	if err != nil {
		return nil, err
	}
	input, err := entry.ToInput()
	if err != nil {
		return nil, err
	}
	execRun, err := s.validator.execSpawner.CreateExecutionRun(cfg.WasmModuleRoot, input).Await(ctx)
	if err != nil {
		return nil, err
	}
	stepLeaves := execRun.GetLeavesWithStepSize(uint64(cfg.MachineStartIndex), uint64(cfg.StepSize), cfg.NumDesiredHashes)
	result, err := stepLeaves.Await(ctx)
	if err != nil {
		return nil, err
	}
	// Do not save a history commitment of length 1 to the cache.
	if len(result) > 1 {
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
	messageNumber l2stateprovider.Height,
	machineIndex l2stateprovider.OpcodeIndex,
) ([]byte, error) {
	entry, err := s.validator.CreateReadyValidationEntry(ctx, arbutil.MessageIndex(messageNumber))
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
