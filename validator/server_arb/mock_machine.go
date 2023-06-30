// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package server_arb

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/validator"
)

type IncorrectMachine struct {
	inner         *ArbitratorMachine
	incorrectStep uint64
	stepCount     uint64
}

var badGlobalState = validator.GoGlobalState{Batch: 0xbadbadbadbad, PosInBatch: 0xbadbadbadbad}

var _ MachineInterface = (*IncorrectMachine)(nil)

func NewIncorrectMachine(inner *ArbitratorMachine, incorrectStep uint64) *IncorrectMachine {
	return &IncorrectMachine{
		inner:         inner.Clone(),
		incorrectStep: incorrectStep,
	}
}

func (m *IncorrectMachine) CloneMachineInterface() MachineInterface {
	return &IncorrectMachine{
		inner:         m.inner.Clone(),
		incorrectStep: m.incorrectStep,
		stepCount:     m.stepCount,
	}
}

func (m *IncorrectMachine) GetGlobalState() validator.GoGlobalState {
	if m.GetStepCount() >= m.incorrectStep {
		return badGlobalState
	}
	return m.inner.GetGlobalState()
}

func (m *IncorrectMachine) GetStepCount() uint64 {
	if !m.IsRunning() {
		endStep := m.incorrectStep
		if endStep < m.inner.GetStepCount() {
			endStep = m.inner.GetStepCount()
		}
		return endStep
	}
	return m.stepCount
}

func (m *IncorrectMachine) IsRunning() bool {
	return m.inner.IsRunning() || m.stepCount < m.incorrectStep
}

func (m *IncorrectMachine) ValidForStep(step uint64) bool {
	return m.inner.ValidForStep(step)
}

func (m *IncorrectMachine) Step(ctx context.Context, count uint64) error {
	err := m.inner.Step(ctx, count)
	if err != nil {
		return err
	}
	prevStepCount := m.stepCount
	m.stepCount += count
	if m.stepCount < prevStepCount {
		// saturate on overflow instead of wrapping
		m.stepCount = ^uint64(0)
	}
	return nil
}

func (m *IncorrectMachine) Status() uint8 {
	return m.inner.Status()
}

func (m *IncorrectMachine) Hash() common.Hash {
	if m.GetStepCount() >= m.incorrectStep {
		if m.inner.IsErrored() {
			return common.HexToHash("0xbad00000bad00000bad00000bad00000")
		}
		beforeGs := m.inner.GetGlobalState()
		if beforeGs != badGlobalState {
			if err := m.inner.SetGlobalState(badGlobalState); err != nil {
				panic(err)
			}
		}
	}
	return m.inner.Hash()
}

func (m *IncorrectMachine) ProveNextStep() []byte {
	return m.inner.ProveNextStep()
}

func (m *IncorrectMachine) Freeze() {
	m.inner.Freeze()
}

func (m *IncorrectMachine) Destroy() {
	m.inner.Destroy()
}
