// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/validator/server_arb"
)

// IncorrectIntermediateMachine will report an incorrect hash while running from incorrectStep onwards.
// However, it'll reach the correct final hash and global state once finished.
type IncorrectIntermediateMachine struct {
	server_arb.MachineInterface
	incorrectStep uint64
}

var _ server_arb.MachineInterface = (*IncorrectIntermediateMachine)(nil)

func NewIncorrectIntermediateMachine(inner server_arb.MachineInterface, incorrectStep uint64) *IncorrectIntermediateMachine {
	return &IncorrectIntermediateMachine{
		MachineInterface: inner,
		incorrectStep:    incorrectStep,
	}
}

func (m *IncorrectIntermediateMachine) CloneMachineInterface() server_arb.MachineInterface {
	return &IncorrectIntermediateMachine{
		MachineInterface: m.MachineInterface.CloneMachineInterface(),
		incorrectStep:    m.incorrectStep,
	}
}

func (m *IncorrectIntermediateMachine) Hash() common.Hash {
	h := m.MachineInterface.Hash()
	if m.GetStepCount() >= m.incorrectStep && m.IsRunning() {
		h[0] ^= 0xFF
	}
	return h
}
