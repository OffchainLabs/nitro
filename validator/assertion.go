//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package validator

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/solgen/go/challengegen"
	"github.com/offchainlabs/arbstate/solgen/go/rollupgen"
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
	InboxMaxCount *big.Int
}

func newExecutionStateFromSolidity(eth rollupgen.RollupLibExecutionState) *ExecutionState {
	return &ExecutionState{
		GlobalState:   GoGlobalStateFromSolidity(challengegen.GlobalState(eth.GlobalState)),
		MachineStatus: MachineStatus(eth.MachineStatus),
		InboxMaxCount: eth.InboxMaxCount,
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
		InboxMaxCount: s.InboxMaxCount,
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

func (a *Assertion) BeforeExecutionHash() common.Hash {
	return a.BeforeState.BlockStateHash()
}

func (a *Assertion) AfterExecutionHash() common.Hash {
	return a.AfterState.BlockStateHash()
}

func BisectionChunkHash(
	segmentStart uint64,
	segmentLength uint64,
	startHash common.Hash,
	endHash common.Hash,
) common.Hash {
	return crypto.Keccak256Hash(
		math.U256Bytes(new(big.Int).SetUint64(segmentStart)),
		math.U256Bytes(new(big.Int).SetUint64(segmentLength)),
		startHash[:],
		endHash[:],
	)
}

func (a *Assertion) ExecutionHash() common.Hash {
	return BisectionChunkHash(
		0,
		a.NumBlocks,
		a.BeforeExecutionHash(),
		a.AfterExecutionHash(),
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
