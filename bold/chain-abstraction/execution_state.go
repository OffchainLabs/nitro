// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

// Package protocol a series of interfaces for interacting with Arbitrum chains' rollup
// and challenge contracts via a developer-friendly, high-level API.
package protocol

import (
	"encoding/binary"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/solgen/go/challengeV2gen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

type GoGlobalState struct {
	BlockHash  common.Hash `json:"blockHash"`
	SendRoot   common.Hash `json:"sendRoot"`
	Batch      uint64      `json:"batch"`
	PosInBatch uint64      `json:"positionInBatch"`
}

func GoGlobalStateFromSolidity(globalState rollupgen.GlobalState) GoGlobalState {
	return GoGlobalState{
		BlockHash:  globalState.Bytes32Vals[0],
		SendRoot:   globalState.Bytes32Vals[1],
		Batch:      globalState.U64Vals[0],
		PosInBatch: globalState.U64Vals[1],
	}
}

func u64ToBe(x uint64) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, x)
	return data
}

func ComputeSimpleMachineChallengeHash(
	execState *ExecutionState,
) common.Hash {
	return execState.GlobalState.Hash()
}

func (s GoGlobalState) Hash() common.Hash {
	data := []byte("Global state:")
	data = append(data, s.BlockHash.Bytes()...)
	data = append(data, s.SendRoot.Bytes()...)
	data = append(data, u64ToBe(s.Batch)...)
	data = append(data, u64ToBe(s.PosInBatch)...)
	return crypto.Keccak256Hash(data)
}

func (s GoGlobalState) AsSolidityStruct() challengeV2gen.GlobalState {
	return challengeV2gen.GlobalState{
		Bytes32Vals: [2][32]byte{s.BlockHash, s.SendRoot},
		U64Vals:     [2]uint64{s.Batch, s.PosInBatch},
	}
}

func (s GoGlobalState) Equals(other GoGlobalState) bool {
	// This is correct because we don't have any pointers or slices
	return s == other
}

type MachineStatus uint8

const (
	MachineStatusRunning  MachineStatus = 0
	MachineStatusFinished MachineStatus = 1
	MachineStatusErrored  MachineStatus = 2
)

type ExecutionState struct {
	GlobalState    GoGlobalState
	MachineStatus  MachineStatus
	EndHistoryRoot common.Hash
}

func GoExecutionStateFromSolidity(executionState rollupgen.AssertionState) *ExecutionState {
	return &ExecutionState{
		GlobalState:    GoGlobalStateFromSolidity(executionState.GlobalState),
		MachineStatus:  MachineStatus(executionState.MachineStatus),
		EndHistoryRoot: executionState.EndHistoryRoot,
	}
}

func (s *ExecutionState) AsSolidityStruct() rollupgen.AssertionState {
	return rollupgen.AssertionState{
		GlobalState:    rollupgen.GlobalState(s.GlobalState.AsSolidityStruct()),
		MachineStatus:  uint8(s.MachineStatus),
		EndHistoryRoot: s.EndHistoryRoot,
	}
}

func (s *ExecutionState) Equals(other *ExecutionState) bool {
	return s.MachineStatus == other.MachineStatus && s.GlobalState.Equals(other.GlobalState) && s.EndHistoryRoot == other.EndHistoryRoot
}

// RequiredBatches determines the batch count required to reach the execution state.
// If the machine errored or the state is after the beginning of the batch,
// the current batch is required to reach the state.
// That's because if the machine errored, it might've read the current batch before erroring,
// and if it's in the middle of a batch, it had to read prior parts of the batch to get there.
// However, if the machine finished successfully and the new state is the start of the batch,
// it hasn't read the batch yet, as it just finished the last batch.
//
// This logic is replicated in Solidity in a few places; search for RequiredBatches to find them.
func (s *ExecutionState) RequiredBatches() uint64 {
	count := s.GlobalState.Batch
	if (s.MachineStatus == MachineStatusErrored || s.GlobalState.PosInBatch > 0) && count < math.MaxUint64 {
		// The current batch was read
		count++
	}
	return count
}
