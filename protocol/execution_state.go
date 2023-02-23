// From: Nitro validator/execution_state.go
package protocol

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengegen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

type GoGlobalState struct {
	BlockHash  common.Hash
	SendRoot   common.Hash
	Batch      uint64
	PosInBatch uint64
}

func u64ToBe(x uint64) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, x)
	return data
}

func ComputeStateHash(
	execState *ExecutionState,
	inboxMaxCount *big.Int,
) common.Hash {
	data := make([]byte, 0)
	globalHash := execState.GlobalState.Hash()
	data = append(data, globalHash[:]...)
	data = append(data, u64ToBe(inboxMaxCount.Uint64())...)
	data = append(data, u64ToBe(uint64(execState.MachineStatus))...)
	return crypto.Keccak256Hash(data)
}

func (s GoGlobalState) Hash() common.Hash {
	data := []byte("Global state:")
	data = append(data, s.BlockHash.Bytes()...)
	data = append(data, s.SendRoot.Bytes()...)
	data = append(data, u64ToBe(s.Batch)...)
	data = append(data, u64ToBe(s.PosInBatch)...)
	return crypto.Keccak256Hash(data)
}

func (s GoGlobalState) AsSolidityStruct() challengegen.GlobalState {
	return challengegen.GlobalState{
		Bytes32Vals: [2][32]byte{s.BlockHash, s.SendRoot},
		U64Vals:     [2]uint64{s.Batch, s.PosInBatch},
	}
}

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

func NewExecutionStateFromSolidity(eth rollupgen.ExecutionState) *ExecutionState {
	return &ExecutionState{
		GlobalState:   GoGlobalStateFromSolidity(challengegen.GlobalState(eth.GlobalState)),
		MachineStatus: MachineStatus(eth.MachineStatus),
	}
}

func GoGlobalStateFromSolidity(gs challengegen.GlobalState) GoGlobalState {
	return GoGlobalState{
		BlockHash:  gs.Bytes32Vals[0],
		SendRoot:   gs.Bytes32Vals[1],
		Batch:      gs.U64Vals[0],
		PosInBatch: gs.U64Vals[1],
	}
}

func (s *ExecutionState) AsSolidityStruct() rollupgen.ExecutionState {
	return rollupgen.ExecutionState{
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
