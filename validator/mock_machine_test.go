//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package validator

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

type IncorrectMachine struct {
	inner         MachineInterface
	incorrectStep uint64
	stepCount     uint64
}

var _ MachineInterface = &IncorrectMachine{}

func NewIncorrectMachine(inner MachineInterface, incorrectStep uint64) *IncorrectMachine {
	return &IncorrectMachine{
		inner:         inner,
		incorrectStep: incorrectStep,
	}
}

func IncorrectMachineHash(correctHash common.Hash) common.Hash {
	correctHash[0] ^= 0xF0
	correctHash[31] ^= 0x0F
	return correctHash
}

func (m *IncorrectMachine) CloneMachineInterface() MachineInterface {
	return &IncorrectMachine{
		inner:         m.inner.CloneMachineInterface(),
		incorrectStep: m.incorrectStep,
		stepCount:     m.stepCount,
	}
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

func (m *IncorrectMachine) Hash() common.Hash {
	h := m.inner.Hash()
	if m.GetStepCount() >= m.incorrectStep {
		h = IncorrectMachineHash(h)
	}
	return h
}

func (m *IncorrectMachine) ProveNextStep() []byte {
	return m.inner.ProveNextStep()
}
