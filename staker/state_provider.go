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

type BoldConfig struct {
	Enable              bool   `koanf:"enable"`
	Evil                bool   `koanf:"evil"`
	Mode                string `koanf:"mode"`
	ValidatorPrivateKey string `koanf:"validator-private-key"`
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

// ExecutionStateMsgCount If the state manager locally has this validated execution state.
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
	fmt.Printf("Global state %+v, batch index %d, count %d\n", globalState, batchIndex, messageCount)
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
	// Check integrity of the arguments.
	if fromBatch > toBatch {
		return nil, fmt.Errorf("from batch %v is greater than to batch %v", fromBatch, toBatch)
	}
	if fromHeight > toHeight {
		return nil, fmt.Errorf("from height %v is greater than to height %v", fromHeight, toHeight)
	}

	// The last message's batch count.
	prevBatchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(uint64(fromBatch) - 1)
	if err != nil {
		return nil, err
	}
	gs, err := s.findGlobalStateFromMessageCountAndBatch(prevBatchMsgCount, fromBatch-1)
	if err != nil {
		return nil, err
	}
	if gs.PosInBatch == 0 {
		return nil, errors.New("final state of batch cannot be at position zero")
	}
	// The start state root of our history commitment starts at `batch: fromBatch, pos: 0` using the state
	// from the last batch.
	gs.Batch += 1
	gs.PosInBatch = 0
	stateRoots := []common.Hash{
		crypto.Keccak256Hash([]byte("Machine finished:"), gs.Hash().Bytes()),
	}
	globalStates := []validator.GoGlobalState{gs}

	// Check if there are enough messages in the range to satisfy our request.
	totalDesiredHashes := (toHeight - fromHeight) + 1

	// We can return early if all we want is one hash.
	if totalDesiredHashes == 1 && fromHeight == 0 && toHeight == 0 {
		return stateRoots, nil
	}

	for batch := fromBatch; batch < toBatch; batch++ {
		msgCount, err := s.validator.inboxTracker.GetBatchMessageCount(uint64(batch))
		if err != nil {
			return nil, err
		}
		var lastGlobalState validator.GoGlobalState

		msgsInBatch := msgCount - prevBatchMsgCount

		if msgsInBatch > 1 {
			for i := uint64(1); i < uint64(msgsInBatch); i++ {
				msgIndex := uint64(prevBatchMsgCount) + i
				gs, err := s.findGlobalStateFromMessageCountAndBatch(arbutil.MessageIndex(msgIndex), batch)
				if err != nil {
					return nil, err
				}
				if gs.BlockHash == (common.Hash{}) {
					continue
				}
				machHash := crypto.Keccak256Hash([]byte("Machine finished:"), gs.Hash().Bytes())
				globalStates = append(globalStates, gs)
				stateRoots = append(stateRoots,
					machHash,
				)
				fmt.Printf("Gs at message index %d and batch %d was %+v and mach hash %#x\n", msgIndex, batch, gs, machHash)
				lastGlobalState = gs
			}
			prevBatchMsgCount = msgCount
			lastGlobalState.Batch += 1
			lastGlobalState.PosInBatch = 0
			machHash := crypto.Keccak256Hash([]byte("Machine finished:"), lastGlobalState.Hash().Bytes())
			stateRoots = append(stateRoots,
				machHash,
			)
			globalStates = append(globalStates, lastGlobalState)
		} else {
			result, err := s.validator.streamer.ResultAtCount(msgCount)
			if err != nil {
				return nil, err
			}
			lastGlobalState.Batch = uint64(batch + 1)
			lastGlobalState.PosInBatch = 0
			lastGlobalState.BlockHash = result.BlockHash
			lastGlobalState.SendRoot = result.SendRoot
			hash := crypto.Keccak256Hash([]byte("Machine finished:"), lastGlobalState.Hash().Bytes())
			stateRoots = append(stateRoots, hash)
			globalStates = append(globalStates, lastGlobalState)
		}
	}

	for _, gs := range globalStates {
		hash := crypto.Keccak256Hash([]byte("Machine finished:"), gs.Hash().Bytes())
		fmt.Printf("Global state %+v and mach hash %#x\n", gs, hash)
	}

	duplicates := make(map[common.Hash]bool)
	finalRoots := make([]common.Hash, 0)
	for _, hash := range stateRoots {
		if ok := duplicates[hash]; ok {
			continue
		}
		finalRoots = append(finalRoots, hash)
		duplicates[hash] = true
	}

	for uint64(len(finalRoots)) < uint64(totalDesiredHashes) {
		finalRoots = append(finalRoots, finalRoots[len(finalRoots)-1])
	}
	return stateRoots[fromHeight : toHeight+1], nil
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
	fmt.Printf("Collecting machine hashes at from batch %d, total %+v, message %d\n", cfg.FromBatch, cfg, messageNum)
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
