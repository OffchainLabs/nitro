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
	flag "github.com/spf13/pflag"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/option"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/state-commitments/history"

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

var executionNodeOfflineGauge = metrics.NewRegisteredGauge("arb/state_provider/execution_node_offline", nil)

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
	API:                                false,
	APIHost:                            "127.0.0.1",
	APIPort:                            9393,
	APIDBPath:                          "/tmp/bold-api-db",
}

func BoldConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBoldConfig.Enable, "enable bold challenge protocol")
	f.Bool(prefix+".evil", DefaultBoldConfig.Evil, "enable evil bold validator")
	f.String(prefix+".mode", DefaultBoldConfig.Mode, "define the bold validator staker strategy")
	f.Uint64(prefix+".block-challenge-leaf-height", DefaultBoldConfig.BlockChallengeLeafHeight, "block challenge leaf height")
	f.Uint64(prefix+".big-step-leaf-height", DefaultBoldConfig.BigStepLeafHeight, "big challenge leaf height")
	f.Uint64(prefix+".small-step-leaf-height", DefaultBoldConfig.SmallStepLeafHeight, "small challenge leaf height")
	f.Uint64(prefix+".num-big-steps", DefaultBoldConfig.NumBigSteps, "num big steps")
	f.String(prefix+".validator-name", DefaultBoldConfig.ValidatorName, "name identifier for cosmetic purposes")
	f.String(prefix+".machine-leaves-cache-path", DefaultBoldConfig.MachineLeavesCachePath, "path to machine cache")
	f.Uint64(prefix+".assertion-posting-interval-seconds", DefaultBoldConfig.AssertionPostingIntervalSeconds, "assertion posting interval")
	f.Uint64(prefix+".assertion-scanning-interval-seconds", DefaultBoldConfig.AssertionScanningIntervalSeconds, "scan assertion interval")
	f.Uint64(prefix+".assertion-confirming-interval-seconds", DefaultBoldConfig.AssertionConfirmingIntervalSeconds, "confirm assertion interval")
	f.Uint64(prefix+".edge-tracker-wake-interval-seconds", DefaultBoldConfig.EdgeTrackerWakeIntervalSeconds, "edge act interval")
	f.Bool(prefix+".api", DefaultBoldConfig.API, "enable api")
	f.String(prefix+".api-host", DefaultBoldConfig.APIHost, "bold api host")
	f.Uint16(prefix+".api-port", DefaultBoldConfig.APIPort, "bold api port")
	f.String(prefix+".api-db-path", DefaultBoldConfig.APIDBPath, "bold api db path")
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

// Produces the L2 execution state to assert to after the previous assertion state.
// Returns either the state at the batch count maxInboxCount or the state maxNumberOfBlocks after previousBlockHash,
// whichever is an earlier state. If previousBlockHash is zero, this function simply returns the state at maxInboxCount.
func (s *StateManager) ExecutionStateAfterPreviousState(
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
		previousMessageCount, err := s.messageCountFromGlobalState(ctx, *previousGlobalState)
		if err != nil {
			return nil, err
		}
		maxMessageCount := previousMessageCount + arbutil.MessageIndex(maxNumberOfBlocks)
		if messageCount > maxMessageCount {
			messageCount = maxMessageCount
			batchIndex, err = FindBatchContainingMessageIndex(s.validator.inboxTracker, messageCount, maxInboxCount)
			if err != nil {
				return nil, err
			}
		}
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

// messageCountFromGlobalState returns the corresponding message count of a global state, assuming that gs is a valid global state.
func (s *StateManager) messageCountFromGlobalState(ctx context.Context, gs protocol.GoGlobalState) (arbutil.MessageIndex, error) {
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

func (s *StateManager) StatesInBatchRange(
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
	machineHashes := []common.Hash{machineHash(startState)}
	states := []validator.GoGlobalState{startState}

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
	items, _, err := s.StatesInBatchRange(fromHeight, to, fromBatch, toBatch)
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
	defer execRun.Close()
	ctxCheckAlive, cancelCheckAlive := ctxWithCheckAlive(ctx, execRun)
	defer cancelCheckAlive()
	stepLeaves := execRun.GetLeavesWithStepSize(uint64(cfg.FromBatch), uint64(cfg.MachineStartIndex), uint64(cfg.StepSize), cfg.NumDesiredHashes)
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
	defer execRun.Close()
	ctxCheckAlive, cancelCheckAlive := ctxWithCheckAlive(ctx, execRun)
	defer cancelCheckAlive()
	oneStepProofPromise := execRun.GetProofAt(uint64(machineIndex))
	return oneStepProofPromise.Await(ctxCheckAlive)
}
