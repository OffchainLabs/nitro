// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validator

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/solgen/go/challengegen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

type MachineStatus uint8

const (
	MachineStatusRunning  MachineStatus = 0
	MachineStatusFinished MachineStatus = 1
	MachineStatusErrored  MachineStatus = 2
	MachineStatusTooFar   MachineStatus = 3
)

type ExecutionState struct {
	GlobalState   GoGlobalState
	MachineStatus MachineStatus
}

func newExecutionStateFromSolidity(eth rollupgen.RollupLibExecutionState) *ExecutionState {
	return &ExecutionState{
		GlobalState:   GoGlobalStateFromSolidity(challengegen.GlobalState(eth.GlobalState)),
		MachineStatus: MachineStatus(eth.MachineStatus),
	}
}

func NewAssertionFromSolidity(assertion rollupgen.RollupLibAssertion) *Assertion {
	return &Assertion{
		BeforeState: newExecutionStateFromSolidity(assertion.BeforeState),
		AfterState:  newExecutionStateFromSolidity(assertion.AfterState),
		NumBlocks:   assertion.NumBlocks,
	}
}

func (s *ExecutionState) AsSolidityStruct() rollupgen.RollupLibExecutionState {
	return rollupgen.RollupLibExecutionState{
		GlobalState:   rollupgen.GlobalState(s.GlobalState.AsSolidityStruct()),
		MachineStatus: uint8(s.MachineStatus),
	}
}

func (s *ExecutionState) BlockStateHash() common.Hash {
	if s.MachineStatus == MachineStatusFinished {
		return crypto.Keccak256Hash([]byte("Block state:"), s.GlobalState.Hash().Bytes())
	} else if s.MachineStatus == MachineStatusErrored {
		return crypto.Keccak256Hash([]byte("Block state, errored:"), s.GlobalState.Hash().Bytes())
	} else if s.MachineStatus == MachineStatusTooFar {
		return crypto.Keccak256Hash([]byte("Block state, too far:"))
	} else {
		panic(fmt.Sprintf("invalid machine status %v", s.MachineStatus))
	}
}

// Determine the batch count required to reach the execution state.
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

func (a *Assertion) AsSolidityStruct() rollupgen.RollupLibAssertion {
	return rollupgen.RollupLibAssertion{
		BeforeState: a.BeforeState.AsSolidityStruct(),
		AfterState:  a.AfterState.AsSolidityStruct(),
		NumBlocks:   a.NumBlocks,
	}
}

func HashChallengeState(
	segmentStart uint64,
	segmentLength uint64,
	hashes []common.Hash,
) common.Hash {
	var hashesBytes []byte
	for _, h := range hashes {
		hashesBytes = append(hashesBytes, h[:]...)
	}
	return crypto.Keccak256Hash(
		math.U256Bytes(new(big.Int).SetUint64(segmentStart)),
		math.U256Bytes(new(big.Int).SetUint64(segmentLength)),
		hashesBytes,
	)
}

func (a *Assertion) ExecutionHash() common.Hash {
	return HashChallengeState(
		0,
		a.NumBlocks,
		[]common.Hash{
			a.BeforeState.BlockStateHash(),
			a.AfterState.BlockStateHash(),
		},
	)
}

type Assertion struct {
	BeforeState *ExecutionState
	AfterState  *ExecutionState
	NumBlocks   uint64
}

type NodeInfo struct {
	NodeNum            uint64
	BlockProposed      uint64
	Assertion          *Assertion
	InboxMaxCount      *big.Int
	AfterInboxBatchAcc common.Hash
	NodeHash           common.Hash
	WasmModuleRoot     common.Hash
}

func (n *NodeInfo) AfterState() *ExecutionState {
	return n.Assertion.AfterState
}

func (n *NodeInfo) MachineStatuses() [2]uint8 {
	return [2]uint8{
		uint8(n.Assertion.BeforeState.MachineStatus),
		uint8(n.Assertion.AfterState.MachineStatus),
	}
}

func (n *NodeInfo) GlobalStates() [2]rollupgen.GlobalState {
	return [2]rollupgen.GlobalState{
		rollupgen.GlobalState(n.Assertion.BeforeState.GlobalState.AsSolidityStruct()),
		rollupgen.GlobalState(n.Assertion.AfterState.GlobalState.AsSolidityStruct()),
	}
}
