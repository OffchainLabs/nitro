// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package staker

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/validator"
)

func NewAssertionFromSolidity(assertion rollupgen.RollupLibAssertion) *Assertion {
	return &Assertion{
		BeforeState: validator.NewExecutionStateFromSolidity(assertion.BeforeState),
		AfterState:  validator.NewExecutionStateFromSolidity(assertion.AfterState),
		NumBlocks:   assertion.NumBlocks,
	}
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
	BeforeState *validator.ExecutionState
	AfterState  *validator.ExecutionState
	NumBlocks   uint64
}

type NodeInfo struct {
	NodeNum                  uint64
	L1BlockProposed          uint64
	ParentChainBlockProposed uint64
	Assertion                *Assertion
	InboxMaxCount            *big.Int
	AfterInboxBatchAcc       common.Hash
	NodeHash                 common.Hash
	WasmModuleRoot           common.Hash
}

func (n *NodeInfo) AfterState() *validator.ExecutionState {
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
