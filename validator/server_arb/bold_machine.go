package server_arb

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/validator"
)

// boldMachine wraps a server_arb.MachineInterface.
type BoldMachine struct {
	inner       MachineInterface
	zeroMachine *ArbitratorMachine
	hasStepped  bool
}

// Ensure boldMachine implements server_arb.MachineInterface.
var _ MachineInterface = (*BoldMachine)(nil)

func newBoldMachine(inner MachineInterface) *BoldMachine {
	z := NewFinishedMachine(inner.GetGlobalState())
	return &BoldMachine{
		inner:       inner,
		zeroMachine: z,
		hasStepped:  false,
	}
}

// Wraps a server_arb.MachineInterface and adds one step to the
// front of the machine's execution.
//
// This zeroth step should be at the same global state as the inner arbitrator
// machine has at step 0, but the machine is in the Finished state rather than
// the Running state.
func BoldMachineWrapper(inner MachineInterface) MachineInterface {
	return newBoldMachine(inner)
}

// CloneMachineInterface returns a new boldMachine with the same inner machine.
func (m *BoldMachine) CloneMachineInterface() MachineInterface {
	bMach := newBoldMachine(m.inner.CloneMachineInterface())
	bMach.hasStepped = m.hasStepped
	return bMach
}

// GetStepCount returns zero if the machine has not stepped, otherwise it
// returns the inner machine's step count plus one.
func (m *BoldMachine) GetStepCount() uint64 {
	if !m.hasStepped {
		return 0
	}
	return m.inner.GetStepCount() + 1
}

// Hash returns the hash of the inner machine if the machine has not stepped,
// otherwise it returns the hash of the zeroth step machine.
func (m *BoldMachine) Hash() common.Hash {
	if !m.hasStepped {
		return m.zeroMachine.Hash()
	}
	return m.inner.Hash()
}

// Destroy destroys the inner machine and the zeroth step machine.
func (m *BoldMachine) Destroy() {
	m.inner.Destroy()
	m.zeroMachine.Destroy()
}

// Freeze freezes the inner machine and the zeroth step machine.
func (m *BoldMachine) Freeze() {
	m.inner.Freeze()
	m.zeroMachine.Freeze()
}

// Status returns the status of the inner machine if the machine has not
// stepped, otherwise it returns the status of the zeroth step machine.
func (m *BoldMachine) Status() uint8 {
	if !m.hasStepped {
		return m.zeroMachine.Status()
	}
	return m.inner.Status()
}

// IsRunning returns true if the machine has not stepped, otherwise it
// returns the running state of the inner machine.
func (m *BoldMachine) IsRunning() bool {
	if !m.hasStepped {
		return true
	}
	return m.inner.IsRunning()
}

// IsErrored returns the errored state of the inner machine, or false if the
// machine has not stepped.
func (m *BoldMachine) IsErrored() bool {
	if !m.hasStepped {
		return false
	}
	return m.inner.IsErrored()
}

// Step steps the inner machine if the machine has not stepped, otherwise it
// steps the zeroth step machine.
func (m *BoldMachine) Step(ctx context.Context, steps uint64) error {
	if !m.hasStepped {
		if steps == 0 {
			// Zero is okay, but doesn't advance the machine.
			return nil
		}
		m.hasStepped = true
		// Only the first step or set of steps needs to be adjusted.
		steps = steps - 1
	}
	return m.inner.Step(ctx, steps)
}

// ValidForStep returns true for step 0 if and only if the machine has not stepped yet,
// and the inner machine's ValidForStep for the step minus one otherwise.
func (m *BoldMachine) ValidForStep(step uint64) bool {
	if step == 0 {
		return !m.hasStepped
	}
	return m.inner.ValidForStep(step - 1)
}

// GetGlobalState returns the global state of the inner machine if the machine
// has stepped, otherwise it returns the global state of the zeroth step.
func (m *BoldMachine) GetGlobalState() validator.GoGlobalState {
	if !m.hasStepped {
		return m.zeroMachine.GetGlobalState()
	}
	return m.inner.GetGlobalState()
}

// ProveNextStep returns the proof of the next step of the inner machine if the
// machine has stepped, otherwise it returns the proof that the zeroth step
// results in the inner machine's initial global state.
func (m *BoldMachine) ProveNextStep() []byte {
	if !m.hasStepped {
		return m.zeroMachine.ProveNextStep()
	}
	return m.inner.ProveNextStep()
}

func (m *BoldMachine) GetNextOpcode() uint16 {
	if !m.hasStepped {
		return m.zeroMachine.GetNextOpcode()
	}
	return m.inner.GetNextOpcode()
}
