// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package staker

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/validator"
)

type ExecutionChallengeBackend struct {
	exec validator.ExecutionRun
}

// NewExecutionChallengeBackend creates a backend with the given arguments.
// Note: machineCache may be nil, but if present, it must not have a restricted range.
func NewExecutionChallengeBackend(executionRun validator.ExecutionRun) (*ExecutionChallengeBackend, error) {
	return &ExecutionChallengeBackend{
		exec: executionRun,
	}, nil
}

func (b *ExecutionChallengeBackend) SetRange(ctx context.Context, start uint64, end uint64) error {
	_, err := b.exec.PrepareRange(start, end).Await(ctx)
	return err
}

func (b *ExecutionChallengeBackend) GetHashAtStep(ctx context.Context, position uint64) (common.Hash, error) {
	step := b.exec.GetStepAt(position)
	result, err := step.Await(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return result.Hash, nil
}

func (b *ExecutionChallengeBackend) GetProofAt(
	ctx context.Context,
	position uint64,
) ([]byte, error) {
	return b.exec.GetProofAt(position).Await(ctx)
}

func (b *ExecutionChallengeBackend) GetFinalState(ctx context.Context) (uint64, validator.GoGlobalState, uint8, error) {
	step := b.exec.GetLastStep()
	res, err := step.Await(ctx)
	if err != nil {
		return 0, validator.GoGlobalState{}, 0, err
	}
	return res.Position, res.GlobalState, uint8(res.Status), nil
}
