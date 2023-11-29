// Package stateprovider defines smarter mocks for testing purposes that can simulate a layer 2
// state provider and and layer 2 state execution.
//
// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package stateprovider

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
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
	commitmentProvider := l2stateprovider.NewHistoryCommitmentProvider(s, s, s, s.challengeLeafHeights, s)
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
	commitmentProvider := l2stateprovider.NewHistoryCommitmentProvider(s, s, s, s.challengeLeafHeights, s)
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
	maxBatchesRead := big.NewInt(int64(s.numBatches))
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
				state.GlobalState.PosInBatch -= uint64(s.posInBatchDivergence)
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
			block = uint64(len(s.executionStates) - 1)
		}
		return NewSimpleMachine(s.executionStates[block], maxBatchesRead), nil
	}
	return s, nil
}

// ExecutionStateAfterBatchCount produces the l2 state to assert at the message number specified.
func (s *L2StateBackend) ExecutionStateAfterBatchCount(ctx context.Context, batchCount uint64) (*protocol.ExecutionState, error) {
	if len(s.executionStates) == 0 {
		return nil, errors.New("no execution states")
	}
	if batchCount >= uint64(len(s.executionStates)) {
		return nil, fmt.Errorf("message number %v is greater than number of execution states %v", batchCount, len(s.executionStates))
	}
	for _, st := range s.executionStates {
		if st.GlobalState.Batch == batchCount {
			return st, nil
		}
	}
	return nil, fmt.Errorf("no execution state at message number %d found", batchCount)
}

// AgreesWithExecutionState returns whether or not we agree with a state.
func (s *L2StateBackend) AgreesWithExecutionState(ctx context.Context, state *protocol.ExecutionState) error {
	for _, r := range s.executionStates {
		if r.Equals(state) {
			return nil
		}
	}
	return l2stateprovider.ErrNoExecutionState
}

func (s *L2StateBackend) statesUpTo(blockStart, blockEnd, fromBatch, toBatch uint64) ([]common.Hash, error) {
	if blockEnd < blockStart {
		return nil, fmt.Errorf("end block %v is less than start block %v", blockEnd, blockStart)
	}
	var startIndex uint64
	for i, st := range s.executionStates {
		if st.GlobalState.Batch == fromBatch {
			startIndex = uint64(i)
			break
		}
	}
	start := startIndex + blockStart
	end := start + blockEnd

	// The size is the number of elements being committed to. For example, if the height is 7, there will
	// be 8 elements being committed to from [0, 7] inclusive.
	desiredStatesLen := int(blockEnd - blockStart + 1)
	var states []common.Hash
	var lastState common.Hash
	for i := start; i <= end; i++ {
		if i >= uint64(len(s.stateRoots)) {
			break
		}
		state := s.stateRoots[i]
		states = append(states, state)
		lastState = state
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
	for len(states) < desiredStatesLen {
		states = append(states, lastState)
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
