// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

// Package stateprovider defines smarter mocks for testing purposes that can
// simulate a layer 2 state provider and layer 2 state execution.
package stateprovider

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ccoveille/go-safecast"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/api/db"
	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/layer2-state-provider"
	"github.com/offchainlabs/nitro/bold/state-commitments/history"
	"github.com/offchainlabs/nitro/bold/testing"
	"github.com/offchainlabs/nitro/bold/testing/casttest"
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

// L2StateBackend defines a very naive state manager that is initialized from a list of predetermined
// state roots. It can produce state and history commitments from those roots.
type L2StateBackend struct {
	l2stateprovider.HistoryCommitmentProvider
	stateRoots              []common.Hash
	executionStates         []*protocol.ExecutionState
	machineAtBlock          func(context.Context, uint64) (Machine, error)
	maxWavmOpcodes          uint64
	blockDivergenceHeight   uint64
	posInBatchDivergence    int64
	machineDivergenceStep   uint64
	forceMachineBlockCompat bool
	maliciousMachineIndex   uint64
	numBigSteps             uint64
	numBatches              uint64
	challengeLeafHeights    []l2stateprovider.Height
}

// NewWithMockedStateRoots initialize with a list of predefined state roots, useful for tests and simulations.
func NewWithMockedStateRoots(stateRoots []common.Hash, opts ...Opt) (*L2StateBackend, error) {
	if len(stateRoots) == 0 {
		return nil, errors.New("no state roots provided")
	}
	s := &L2StateBackend{
		stateRoots: stateRoots,
		machineAtBlock: func(context.Context, uint64) (Machine, error) {
			return nil, errors.New("state manager created with New() cannot provide machines")
		},
		numBigSteps: 1,
		challengeLeafHeights: []l2stateprovider.Height{
			challenge_testing.LevelZeroBlockEdgeHeight,
			challenge_testing.LevelZeroBigStepEdgeHeight,
			challenge_testing.LevelZeroSmallStepEdgeHeight,
		},
	}
	for _, o := range opts {
		o(s)
	}
	commitmentProvider := l2stateprovider.NewHistoryCommitmentProvider(s, s, s, s.challengeLeafHeights, s, nil)
	s.HistoryCommitmentProvider = *commitmentProvider
	return s, nil
}

type Opt func(*L2StateBackend)

func WithMaxWavmOpcodesPerBlock(maxOpcodes uint64) Opt {
	return func(s *L2StateBackend) {
		s.maxWavmOpcodes = maxOpcodes
	}
}

func WithMachineDivergenceStep(divergenceStep uint64) Opt {
	return func(s *L2StateBackend) {
		s.machineDivergenceStep = divergenceStep
	}
}

func WithBlockDivergenceHeight(divergenceHeight uint64) Opt {
	return func(s *L2StateBackend) {
		s.blockDivergenceHeight = divergenceHeight
	}
}

func WithDivergentBlockHeightOffset(blockHeightOffset int64) Opt {
	return func(s *L2StateBackend) {
		s.posInBatchDivergence = blockHeightOffset * 150
	}
}

func WithMachineAtBlockProvider(machineAtBlock func(ctx context.Context, blockNum uint64) (Machine, error)) Opt {
	return func(s *L2StateBackend) {
		s.machineAtBlock = machineAtBlock
	}
}

// WithForceMachineBlockCompat if enabled, forces the machine hash at block boundaries to be the block hash
func WithForceMachineBlockCompat() Opt {
	return func(s *L2StateBackend) {
		s.forceMachineBlockCompat = true
	}
}

func WithLayerZeroHeights(heights *protocol.LayerZeroHeights, numBigSteps uint8) Opt {
	return func(s *L2StateBackend) {
		challengeLeafHeights := make([]l2stateprovider.Height, 0)
		challengeLeafHeights = append(challengeLeafHeights, l2stateprovider.Height(heights.BlockChallengeHeight))
		for i := uint8(0); i < numBigSteps; i++ {
			challengeLeafHeights = append(challengeLeafHeights, l2stateprovider.Height(heights.BigStepChallengeHeight))
		}
		challengeLeafHeights = append(challengeLeafHeights, l2stateprovider.Height(heights.SmallStepChallengeHeight))
		s.challengeLeafHeights = challengeLeafHeights
	}
}

func WithMaliciousMachineIndex(index uint64) Opt {
	return func(s *L2StateBackend) {
		s.maliciousMachineIndex = index
	}
}

func WithNumBatchesRead(n uint64) Opt {
	return func(s *L2StateBackend) {
		s.numBatches = n
	}
}

func NewForSimpleMachine(
	t testing.TB,
	opts ...Opt,
) (*L2StateBackend, error) {
	s := &L2StateBackend{
		maliciousMachineIndex: 0,
		challengeLeafHeights: []l2stateprovider.Height{
			challenge_testing.LevelZeroBlockEdgeHeight,
			challenge_testing.LevelZeroBigStepEdgeHeight,
			challenge_testing.LevelZeroSmallStepEdgeHeight,
		},
		numBatches: 1,
	}
	for _, o := range opts {
		o(s)
	}
	commitmentProvider := l2stateprovider.NewHistoryCommitmentProvider(s, s, s, s.challengeLeafHeights, s, nil)
	s.HistoryCommitmentProvider = *commitmentProvider
	totalWavmOpcodes := uint64(1)
	for _, h := range s.challengeLeafHeights[1:] {
		totalWavmOpcodes *= uint64(h)
	}
	s.maxWavmOpcodes = totalWavmOpcodes
	if s.maxWavmOpcodes == 0 {
		return nil, errors.New("maxWavmOpcodes cannot be zero")
	}
	if s.blockDivergenceHeight > 0 && s.machineDivergenceStep == 0 {
		return nil, errors.New("machineDivergenceStep cannot be zero if blockDivergenceHeight is non-zero")
	}
	nextMachineState := &protocol.ExecutionState{
		GlobalState:   protocol.GoGlobalState{},
		MachineStatus: protocol.MachineStatusFinished,
	}
	maxBatchesRead := big.NewInt(casttest.ToInt64(t, s.numBatches))
	for block := uint64(0); ; block++ {
		machine := NewSimpleMachine(nextMachineState, maxBatchesRead)
		state := machine.GetExecutionState()
		machHash := machine.Hash()
		if machHash != state.GlobalState.Hash() {
			return nil, fmt.Errorf("machine at block %v has hash %v but we expected hash %v", block, machine.Hash(), state.GlobalState.Hash())
		}
		if s.blockDivergenceHeight > 0 {
			if block == s.blockDivergenceHeight {
				// Note: blockHeightOffset might be negative, but two's complement subtraction works regardless
				state.GlobalState.PosInBatch -= casttest.ToUint64(t, s.posInBatchDivergence)
			}
			if block >= s.blockDivergenceHeight {
				state.GlobalState.BlockHash[s.maliciousMachineIndex] = 1
			}
			machHash = protocol.ComputeSimpleMachineChallengeHash(state)
		}
		s.executionStates = append(s.executionStates, state)
		s.stateRoots = append(s.stateRoots, machHash)

		if machine.IsStopped() || state.GlobalState.Batch >= s.numBatches {
			break
		}
		err := machine.Step(s.maxWavmOpcodes)
		if err != nil {
			return nil, err
		}
		nextMachineState = machine.GetExecutionState()
	}
	s.machineAtBlock = func(_ context.Context, block uint64) (Machine, error) {
		if block >= uint64(len(s.executionStates)) {
			block = casttest.ToUint64(t, len(s.executionStates)-1)
		}
		return NewSimpleMachine(s.executionStates[block], maxBatchesRead), nil
	}
	return s, nil
}

func (s *L2StateBackend) UpdateAPIDatabase(database db.Database) {
	commitmentProvider := l2stateprovider.NewHistoryCommitmentProvider(s, s, s, s.challengeLeafHeights, s, database)
	s.HistoryCommitmentProvider = *commitmentProvider
}

// ExecutionStateAfterPreviousState produces the l2 state to assert at the message number specified.
func (s *L2StateBackend) ExecutionStateAfterPreviousState(ctx context.Context, maxInboxCount uint64, previousGlobalState protocol.GoGlobalState) (*protocol.ExecutionState, error) {
	if len(s.executionStates) == 0 {
		return nil, errors.New("no execution states")
	}
	if maxInboxCount >= uint64(len(s.executionStates)) {
		return nil, fmt.Errorf("message number %v is greater than number of execution states %v", maxInboxCount, len(s.executionStates))
	}
	blocksSincePrevious := -1
	for _, st := range s.executionStates {
		if st.GlobalState.Equals(previousGlobalState) {
			blocksSincePrevious = 0
		}
		bsp64, err := safecast.ToUint64(blocksSincePrevious + 1)
		if err != nil {
			return nil, fmt.Errorf("could not convert blocksSincePrevious to uint64: %w", err)
		}
		if st.GlobalState.Batch == maxInboxCount || (blocksSincePrevious >= 0 && bsp64 >= uint64(s.challengeLeafHeights[0])) {
			if blocksSincePrevious < 0 {
				return nil, fmt.Errorf("missing previous global state %+v", previousGlobalState)
			}
			// Compute the history commitment for the assertion state.
			fromBatch := previousGlobalState.Batch
			historyCommit, err := s.statesUpTo(0, uint64(s.challengeLeafHeights[0]), fromBatch, st.GlobalState.Batch)
			if err != nil {
				return nil, err
			}
			commit, err := history.NewCommitment(historyCommit, uint64(s.challengeLeafHeights[0])+1)
			if err != nil {
				return nil, err
			}
			st.EndHistoryRoot = commit.Merkle
			return st, nil
		}
		if blocksSincePrevious >= 0 {
			blocksSincePrevious++
		}
	}
	return nil, fmt.Errorf("no execution state at message number %d found", maxInboxCount)
}

func (s *L2StateBackend) statesUpTo(blockStart, blockEnd, fromBatch, toBatch uint64) ([]common.Hash, error) {
	if blockEnd < blockStart {
		return nil, fmt.Errorf("end block %v is less than start block %v", blockEnd, blockStart)
	}
	var err error
	var startIndex uint64
	for i, st := range s.executionStates {
		if st.GlobalState.Batch == fromBatch {
			startIndex, err = safecast.ToUint64(i)
			if err != nil {
				return nil, fmt.Errorf("could not convert start index to uint64: %w", err)
			}
			break
		}
	}
	start := startIndex + blockStart
	end := start + blockEnd

	var states []common.Hash
	for i := start; i <= end; i++ {
		if i >= uint64(len(s.stateRoots)) {
			break
		}
		state := s.stateRoots[i]
		states = append(states, state)
		if len(s.executionStates) == 0 {
			// should only happen in tests
			continue
		}
		gs := s.executionStates[i].GlobalState
		if gs.Batch >= toBatch {
			if gs.Batch > toBatch || gs.PosInBatch > 0 {
				return nil, fmt.Errorf("overran next batch count %v with global state batch %v position %v", toBatch, gs.Batch, gs.PosInBatch)
			}
			break
		}
	}
	return states, nil
}

func (s *L2StateBackend) maybeDivergeState(state *protocol.ExecutionState, block uint64, step uint64) {
	if block+1 == s.blockDivergenceHeight && step == s.maxWavmOpcodes {
		*state = *s.executionStates[block+1]
	}
	if block+1 > s.blockDivergenceHeight || step >= s.machineDivergenceStep {
		state.GlobalState.BlockHash[s.maliciousMachineIndex] = 1
	}
}

// May modify the machine hash if divergence is enabled
func (s *L2StateBackend) getMachineHash(machine Machine, block uint64) common.Hash {
	if s.forceMachineBlockCompat {
		step := machine.CurrentStepNum()
		if step == 0 {
			return s.stateRoots[block]
		}
		if step == s.maxWavmOpcodes {
			return s.stateRoots[block+1]
		}
	}
	if s.blockDivergenceHeight == 0 || block+1 < s.blockDivergenceHeight {
		return machine.Hash()
	}
	state := machine.GetExecutionState()
	s.maybeDivergeState(state, block, machine.CurrentStepNum())
	return protocol.ComputeSimpleMachineChallengeHash(state)
}
