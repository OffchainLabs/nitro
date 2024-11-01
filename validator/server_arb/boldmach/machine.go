package boldmach

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_arb"
)

// boldMachine wraps a server_arb.MachineInterface.
type boldMachine struct {
	server_arb.MachineInterface
	zeroMachine *server_arb.ArbitratorMachine
	hasStepped  bool
}

// Ensure boldMachine implements server_arb.MachineInterface.
var _ server_arb.MachineInterface = (*boldMachine)(nil)

// MachineWrapper wraps a server_arb.MachineInterface and adds one step to the
// front of the machine's execution.
//
// This zeroth step should be at the same global state as the inner arbitrator
// machine has at step 0, but the machine is in the Finished state rather than
// the Running state.
func MachineWrapper(inner server_arb.MachineInterface) server_arb.MachineInterface {
	z := server_arb.NewFinishedMachine()
	z.SetGlobalState(inner.GetGlobalState())
	return &boldMachine{
		MachineInterface: inner,
		zeroMachine:      z,
		hasStepped:       false,
	}
}

// CloneMachineInterface returns a new boldMachine with the same inner machine.
func (m *boldMachine) CloneMachineInterface() server_arb.MachineInterface {
	c := MachineWrapper(m.MachineInterface.CloneMachineInterface())
	c.(*boldMachine).hasStepped = m.hasStepped
	return c
}

// GetStepCount returns zero if the machine has not stepped, otherwise it
// returns the inner machine's step count plus one.
func (m *boldMachine) GetStepCount() uint64 {
	if !m.hasStepped {
		return 0
	}
	return m.MachineInterface.GetStepCount() + 1
}

// Hash returns the hash of the inner machine if the machine has not stepped,
// otherwise it returns the hash of the zeroth step machine.
func (m *boldMachine) Hash() common.Hash {
	if !m.hasStepped {
		return m.MachineInterface.Hash()
	}
	return m.zeroMachine.Hash()
}

// Destroy destroys the inner machine and the zeroth step machine.
func (m *boldMachine) Destroy() {
	m.MachineInterface.Destroy()
	m.zeroMachine.Destroy()
}

// Freeze freezes the inner machine and the zeroth step machine.
func (m *boldMachine) Freeze() {
	m.MachineInterface.Freeze()
	m.zeroMachine.Freeze()
}

// Status returns the status of the inner machine if the machine has not
// stepped, otherwise it returns the status of the zeroth step machine.
func (m *boldMachine) Status() uint8 {
	if !m.hasStepped {
		return m.MachineInterface.Status()
	}
	return m.zeroMachine.Status()
}

// IsRunning returns the running state of the zeroeth state machine if the
// machine has not stepped, otherwise it returns the running state of the
// inner machine.
func (m *boldMachine) IsRunning() bool {
	if !m.hasStepped {
		return m.zeroMachine.IsRunning()
	}
	return m.MachineInterface.IsRunning()
}

// IsErrored returns the errored state of the inner machine, or false if the
// machine has not stepped.
func (m *boldMachine) IsErrored() bool {
	if !m.hasStepped {
		return false
	}
	return m.MachineInterface.IsErrored()
}

// Step steps the inner machine if the machine has not stepped, otherwise it
// steps the zeroth step machine.
func (m *boldMachine) Step(ctx context.Context, steps uint64) error {
	if !m.hasStepped {
		if steps == 0 {
			// Zero is okay, but doesn't advance the machine.
			return nil
		}
		m.hasStepped = true
		// Only the first step or set of steps needs to be adjusted.
		steps = steps - 1
	}
	return m.MachineInterface.Step(ctx, steps)
}

// ValidForStep returns true for step 0, and the inner machine's ValidForStep
// for the step minus one.
func (m *boldMachine) ValidForStep(step uint64) bool {
	if step == 0 {
		return true
	}
	return m.MachineInterface.ValidForStep(step - 1)
}

// GetGlobalState returns the global state of the inner machine if the machine
// has stepped, otherwise it returns the global state of the zeroth step.
func (m *boldMachine) GetGlobalState() validator.GoGlobalState {
	if !m.hasStepped {
		return m.zeroMachine.GetGlobalState()
	}
	return m.MachineInterface.GetGlobalState()
}

// ProveNextStep returns the proof of the next step of the inner machine if the
// machine has stepped, otherwise it returns the proof that the zeroth step
// results in the inner machine's initial global state.
func (m *boldMachine) ProveNextStep() []byte {
	if !m.hasStepped {
		// NOT AT ALL SURE ABOUT THIS.  I THINK IT'S WRONG.
		return m.zeroMachine.ProveNextStep()
	}
	return m.MachineInterface.ProveNextStep()
}
