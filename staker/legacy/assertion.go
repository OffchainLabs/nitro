// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package legacystaker

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/solgen/go/rollup_legacy_gen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/validator"
)

func NewAssertionFromLegacySolidity(assertion rollup_legacy_gen.Assertion) *Assertion {
	return &Assertion{
		BeforeState: validator.NewExecutionStateFromLegacySolidity(assertion.BeforeState),
		AfterState:  validator.NewExecutionStateFromLegacySolidity(assertion.AfterState),
		NumBlocks:   assertion.NumBlocks,
	}
}

func (a *Assertion) AsLegacySolidityStruct() rollup_legacy_gen.Assertion {
	return rollup_legacy_gen.Assertion{
		BeforeState: a.BeforeState.AsLegacySolidityStruct(),
		AfterState:  a.AfterState.AsLegacySolidityStruct(),
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
		arbmath.Uint64ToU256Bytes(segmentStart),
		arbmath.Uint64ToU256Bytes(segmentLength),
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

func (n *NodeInfo) GlobalStates() [2]rollup_legacy_gen.GlobalState {
	return [2]rollup_legacy_gen.GlobalState{
		n.Assertion.BeforeState.GlobalState.AsLegacySolidityStruct(),
		n.Assertion.AfterState.GlobalState.AsLegacySolidityStruct(),
	}
}
