// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/challenge-protocol-v2/blob/main/LICENSE
package toys

import (
	"encoding/binary"
	"errors"
	"math/big"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrOutOfBounds = errors.New("instruction number out of bounds")
)

type Machine interface {
	CurrentStepNum() uint64
	GetExecutionState() *protocol.ExecutionState
	Hash() common.Hash
	IsStopped() bool
	Clone() Machine
	Step(steps uint64) error
	OneStepProof() ([]byte, error)
}

type SimpleMachine struct {
	step           uint64
	state          *protocol.ExecutionState
	maxBatchesRead *big.Int
}

func NewSimpleMachine(startingState *protocol.ExecutionState, maxBatchesRead *big.Int) *SimpleMachine {
	stateCopy := *startingState
	if maxBatchesRead != nil {
		maxBatchesRead = new(big.Int).Set(maxBatchesRead)
	}
	return &SimpleMachine{
		step:           0,
		state:          &stateCopy,
		maxBatchesRead: maxBatchesRead,
	}
}

func (m *SimpleMachine) CurrentStepNum() uint64 {
	return m.step
}

func (m *SimpleMachine) GetExecutionState() *protocol.ExecutionState {
	stateCopy := *m.state
	return &stateCopy
}

func (m *SimpleMachine) Hash() common.Hash {
	return m.GetExecutionState().GlobalState.Hash()
}

func (m *SimpleMachine) IsStopped() bool {
	if m.step == 0 && m.state.MachineStatus == protocol.MachineStatusFinished {
		if m.maxBatchesRead == nil || new(big.Int).SetUint64(m.state.GlobalState.Batch).Cmp(m.maxBatchesRead) < 0 {
			// Kickstart the machine at step 0
			return false
		}
	}
	return m.state.MachineStatus != protocol.MachineStatusRunning
}

func (m *SimpleMachine) Clone() Machine {
	newMachine := *m
	stateCopy := *m.state
	newMachine.state = &stateCopy
	return &newMachine
}

// End the batch after 2000 steps. This results in 11 blocks for an honest validator.
// This constant must be synchronized with the one in execution/engine.go
const stepsPerBatch = 2000

func (m *SimpleMachine) Step(steps uint64) error {
	for ; steps > 0; steps-- {
		if m.IsStopped() {
			m.step += steps
			return nil
		}
		m.state.MachineStatus = protocol.MachineStatusRunning
		m.step++
		m.state.GlobalState.PosInBatch++
		if m.state.GlobalState.PosInBatch%stepsPerBatch == 0 {
			m.state.GlobalState.Batch++
			m.state.GlobalState.PosInBatch = 0
			m.state.MachineStatus = protocol.MachineStatusFinished
		}
		if m.Hash()[0] == 0 {
			m.state.MachineStatus = protocol.MachineStatusFinished
		}
	}
	return nil
}

func (m *SimpleMachine) OneStepProof() ([]byte, error) {
	proof := make([]byte, 16)
	binary.BigEndian.PutUint64(proof[:8], m.state.GlobalState.Batch)
	binary.BigEndian.PutUint64(proof[8:], m.state.GlobalState.PosInBatch)
	return proof, nil
}

// VerifySimpleMachineOneStepProof checks the claimed post-state root results from executing
// a specified pre-state hash.
func VerifySimpleMachineOneStepProof(beforeStateRoot common.Hash, claimedAfterStateRoot common.Hash, step uint64, maxBatchesRead *big.Int, proof []byte) bool {
	if len(proof) != 16 {
		return false
	}
	batch := binary.BigEndian.Uint64(proof[:8])
	posInBatch := binary.BigEndian.Uint64(proof[8:])
	state := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			Batch:      batch,
			PosInBatch: posInBatch,
		},
		MachineStatus: protocol.MachineStatusRunning,
	}
	mach := NewSimpleMachine(state, maxBatchesRead)
	mach.step = step
	if step == 0 || mach.Hash()[0] == 0 {
		mach.state.MachineStatus = protocol.MachineStatusFinished
	}
	if mach.Hash() != beforeStateRoot {
		return false
	}
	err := mach.Step(1)
	if err != nil {
		return false
	}
	return mach.Hash() == claimedAfterStateRoot
}
