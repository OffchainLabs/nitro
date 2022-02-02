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
}

func newExecutionStateFromFields(a [2][32]byte, b [3]uint64) *ExecutionState {
	if b[2] >= (1 << 8) {
		panic(fmt.Sprintf("invalid machine status %v", b[2]))
	}
	return &ExecutionState{
		GlobalState: GoGlobalState{
			BlockHash:  a[0],
			SendRoot:   a[1],
			Batch:      b[0],
			PosInBatch: b[1],
		},
		MachineStatus: MachineStatus(uint8(b[2])),
	}
}

func NewAssertionFromFields(a [2][2][32]byte, b [2][3]uint64) *Assertion {
	return &Assertion{
		BeforeState: newExecutionStateFromFields(a[0], b[0]),
		AfterState:  newExecutionStateFromFields(a[1], b[1]),
	}
}

func (s *ExecutionState) ByteFields() [2][32]byte {
	return [2][32]byte{
		s.GlobalState.BlockHash,
		s.GlobalState.SendRoot,
	}
}

func (s *ExecutionState) IntFields() [3]uint64 {
	return [3]uint64{
		s.GlobalState.Batch,
		s.GlobalState.PosInBatch,
		uint64(s.MachineStatus),
	}
}

func (a *ExecutionState) BlockStateHash() common.Hash {
	if a.MachineStatus == MachineStatusFinished {
		return crypto.Keccak256Hash([]byte("Block state:"), a.GlobalState.Hash().Bytes())
	} else if a.MachineStatus == MachineStatusErrored {
		return crypto.Keccak256Hash([]byte("Block state, errored:"), a.GlobalState.Hash().Bytes())
	} else if a.MachineStatus == MachineStatusTooFar {
		return crypto.Keccak256Hash([]byte("Block state, too far:"))
	} else {
		panic(fmt.Sprintf("invalid machine status %v", a.MachineStatus))
	}
}

func (a *Assertion) BytesFields() [2][2][32]byte {
	return [2][2][32]byte{
		a.BeforeState.ByteFields(),
		a.AfterState.ByteFields(),
	}
}

func (a *Assertion) IntFields() [2][3]uint64 {
	return [2][3]uint64{
		a.BeforeState.IntFields(),
		a.AfterState.IntFields(),
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

type NodeState struct {
	InboxMaxCount *big.Int
	*ExecutionState
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

func (n *NodeInfo) AfterState() *NodeState {
	return &NodeState{
		InboxMaxCount:  n.InboxMaxCount,
		ExecutionState: n.Assertion.AfterState,
	}
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
