// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validator

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type ExecutionChallengeBackend struct {
	initialMachine    MachineInterface
	lastMachine       MachineInterface
	machineCache      *MachineCache
	machineCacheStart uint64
	machineCacheEnd   uint64
	targetNumMachines int
}

// NewExecutionChallengeBackend creates a backend with the given arguments.
// Note: machineCache may be nil, but if present, it must not have a restricted range.
func NewExecutionChallengeBackend(
	initialMachine MachineInterface,
	targetNumMachines int,
	machineCache *MachineCache,
) (*ExecutionChallengeBackend, error) {
	if initialMachine.GetStepCount() != 0 {
		return nil, errors.New("initialMachine not at step count 0")
	}
	return &ExecutionChallengeBackend{
		initialMachine:    initialMachine,
		targetNumMachines: targetNumMachines,
		machineCache:      machineCache,
	}, nil
}

func (b *ExecutionChallengeBackend) getMachineAt(ctx context.Context, stepCount uint64) (MachineInterface, error) {
	if b.machineCache == nil {
		mach := b.initialMachine
		if b.lastMachine != nil && b.lastMachine.GetStepCount() <= stepCount {
			mach = b.lastMachine
		}
		mach = mach.CloneMachineInterface()
		err := mach.Step(ctx, stepCount-mach.GetStepCount())
		if err != nil {
			return nil, err
		}
		b.lastMachine = mach
		return mach, nil
	} else {
		mach, err := b.machineCache.GetMachineAt(ctx, b.lastMachine, stepCount)
		if err != nil {
			return nil, err
		}
		b.lastMachine = mach
		return mach, nil
	}
}

func (b *ExecutionChallengeBackend) SetRange(ctx context.Context, start uint64, end uint64) error {
	if b.machineCache != nil && b.machineCacheStart == start && b.machineCacheEnd == end {
		return nil
	}
	startMach, err := b.getMachineAt(ctx, start)
	if err != nil {
		return err
	}
	b.machineCache = nil
	b.machineCache, err = NewMachineCacheWithEndSteps(ctx, startMach, b.targetNumMachines, end)
	return err
}

func (b *ExecutionChallengeBackend) GetHashAtStep(ctx context.Context, position uint64) (common.Hash, error) {
	mach, err := b.getMachineAt(ctx, position)
	if err != nil {
		return common.Hash{}, err
	}
	return mach.Hash(), nil
}

func (b *ExecutionChallengeBackend) GetProofAt(
	ctx context.Context,
	step uint64,
) ([]byte, error) {
	mach, err := b.getMachineAt(ctx, step)
	if err != nil {
		return nil, err
	}
	return mach.ProveNextStep(), nil
}

func (b *ExecutionChallengeBackend) GetFinalState(ctx context.Context) (uint64, GoGlobalState, uint8, error) {
	// TODO: we might also use HostIoMachineTo Speed things up
	initialRunMachine := b.initialMachine.CloneMachineInterface()
	var stepCount uint64
	for initialRunMachine.IsRunning() {
		stepsPerLoop := uint64(1_000_000_000)
		if stepCount > 0 {
			log.Debug("step count machine", "steps", stepCount)
		}
		err := initialRunMachine.Step(ctx, stepsPerLoop)
		if err != nil {
			return 0, GoGlobalState{}, 0, err
		}
		stepCount += stepsPerLoop
	}
	stepCount = initialRunMachine.GetStepCount()
	computedStatus := initialRunMachine.GetGlobalState()
	status := initialRunMachine.Status()

	return stepCount, computedStatus, status, nil
}
