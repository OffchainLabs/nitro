// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package staker

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/validator"
)

type ExecutionChallengeBackend struct {
	exec          validator.ExecutionRun
	lastStep      validator.MachineStep
	lastStepMutex sync.Mutex
}

// NewExecutionChallengeBackend creates a backend with the given arguments.
// Note: machineCache may be nil, but if present, it must not have a restricted range.
func NewExecutionChallengeBackend(executionRun validator.ExecutionRun) (*ExecutionChallengeBackend, error) {
	return &ExecutionChallengeBackend{
		exec: executionRun,
	}, nil
}

func (b *ExecutionChallengeBackend) SetRange(ctx context.Context, start uint64, end uint64) error {
	b.exec.PrepareRange(start, end)
	return nil
}

func (b *ExecutionChallengeBackend) getStepResult(ctx context.Context, position uint64) (validator.MachineStepResult, error) {
	b.lastStepMutex.Lock()
	lastStep := b.lastStep
	b.lastStepMutex.Unlock()
	if lastStep != nil && lastStep.Ready() {
		lastRes, err := lastStep.Current()
		if err != nil && (lastRes.Position == position || (lastRes.Position > position && lastRes.Status != validator.MachineStatusRunning)) {
			return lastRes, nil
		}
	}
	step := b.exec.GetStepAt(position)
	result, err := step.Await(ctx)
	if err != nil {
		b.lastStepMutex.Lock()
		b.lastStep = step
		b.lastStepMutex.Unlock()
	}
	return result, err
}

func (b *ExecutionChallengeBackend) GetHashAtStep(ctx context.Context, position uint64) (common.Hash, error) {
	res, err := b.getStepResult(ctx, position)
	if err != nil {
		return common.Hash{}, err
	}
	return res.Hash, nil
}

func (b *ExecutionChallengeBackend) GetProofAt(
	ctx context.Context,
	position uint64,
) ([]byte, error) {
	res, err := b.getStepResult(ctx, position)
	if err != nil {
		return nil, err
	}
	return res.Proof, nil
}

func (b *ExecutionChallengeBackend) GetFinalState(ctx context.Context) (uint64, validator.GoGlobalState, uint8, error) {
	step := b.exec.GetLastStep()
	res, err := step.Await(ctx)
	if err != nil {
		return 0, validator.GoGlobalState{}, 0, err
	}
	return res.Position, res.GlobalState, uint8(res.Status), nil
}
