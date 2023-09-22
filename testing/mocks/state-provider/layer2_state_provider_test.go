// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package stateprovider

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"testing"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func mockMachineAtBlock(_ context.Context, block uint64) (Machine, error) {
	blockBytes := make([]uint8, 8)
	binary.BigEndian.PutUint64(blockBytes, block)
	startState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash: crypto.Keccak256Hash(blockBytes),
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	return NewSimpleMachine(startState, nil), nil
}
func setupStates(t *testing.T, numStates, divergenceHeight uint64) ([]*protocol.ExecutionState, []common.Hash) {
	t.Helper()
	states := make([]*protocol.ExecutionState, numStates)
	roots := make([]common.Hash, numStates)
	for i := uint64(0); i < numStates; i++ {
		var blockHash common.Hash
		if divergenceHeight == 0 || i < divergenceHeight {
			blockHash = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
		} else {
			junkRoot := make([]byte, 32)
			_, err := rand.Read(junkRoot)
			require.NoError(t, err)
			blockHash = crypto.Keccak256Hash(junkRoot)
		}
		state := &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash:  blockHash,
				Batch:      0,
				PosInBatch: i,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}
		if i+1 == numStates {
			state.GlobalState.Batch = 1
			state.GlobalState.PosInBatch = 0
		}
		states[i] = state
		roots[i] = protocol.ComputeSimpleMachineChallengeHash(state)
	}
	return states, roots
}

func newTestingMachine(
	assertionChainExecutionStates []*protocol.ExecutionState,
	opts ...Opt,
) (*L2StateBackend, error) {
	if len(assertionChainExecutionStates) == 0 {
		return nil, errors.New("must have execution states")
	}
	stateRoots := make([]common.Hash, len(assertionChainExecutionStates))
	var lastBatch uint64 = math.MaxUint64
	var lastPosInBatch uint64 = math.MaxUint64
	for i := 0; i < len(stateRoots); i++ {
		state := assertionChainExecutionStates[i]
		if state.GlobalState.Batch == lastBatch && state.GlobalState.PosInBatch == lastPosInBatch {
			return nil, fmt.Errorf("execution states %v and %v have the same batch %v and position in batch %v", i-1, i, lastBatch, lastPosInBatch)
		}
		lastBatch = state.GlobalState.Batch
		lastPosInBatch = state.GlobalState.PosInBatch
		stateRoots[i] = protocol.ComputeSimpleMachineChallengeHash(state)
	}
	s := &L2StateBackend{
		stateRoots:      stateRoots,
		executionStates: assertionChainExecutionStates,
		machineAtBlock: func(context.Context, uint64) (Machine, error) {
			return nil, errors.New("state manager created with NewWithAssertionStates() cannot provide machines")
		},
		levelZeroBlockEdgeHeight:     challenge_testing.LevelZeroBlockEdgeHeight,
		levelZeroBigStepEdgeHeight:   challenge_testing.LevelZeroBigStepEdgeHeight,
		levelZeroSmallStepEdgeHeight: challenge_testing.LevelZeroSmallStepEdgeHeight,
	}
	for _, o := range opts {
		o(s)
	}
	return s, nil
}
