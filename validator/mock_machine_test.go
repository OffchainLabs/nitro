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
}

var _ MachineInterface = &IncorrectMachine{}

func NewIncorrectMachine(inner MachineInterface, incorrectStep uint64) *IncorrectMachine {
	return &IncorrectMachine{
		inner:         inner,
		incorrectStep: incorrectStep,
	}
}

func (m *IncorrectMachine) CloneMachineInterface() MachineInterface {
	return &IncorrectMachine{
		inner:         m.inner.CloneMachineInterface(),
		incorrectStep: m.incorrectStep,
	}
}

func (m *IncorrectMachine) GetStepCount() uint64 {
	return m.inner.GetStepCount()
}

func (m *IncorrectMachine) IsRunning() bool {
	return m.inner.IsRunning()
}

func (m *IncorrectMachine) ValidForStep(step uint64) bool {
	return m.inner.ValidForStep(step)
}

func (m *IncorrectMachine) Step(ctx context.Context, count uint64) error {
	return m.inner.Step(ctx, count)
}

func (m *IncorrectMachine) Hash() common.Hash {
	h := m.inner.Hash()
	if m.GetStepCount() >= m.incorrectStep {
		h[0] ^= 0xF0
		h[31] ^= 0x0F
	}
	return h
}

func (m *IncorrectMachine) ProveNextStep() []byte {
	return m.inner.ProveNextStep()
}
