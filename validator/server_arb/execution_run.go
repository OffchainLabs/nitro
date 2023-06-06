// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package server_arb

import (
	"context"
	"fmt"
	"sync"

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
)

type executionRun struct {
	stopwaiter.StopWaiter
	cache *MachineCache
	close sync.Once
}

// NewExecutionChallengeBackend creates a backend with the given arguments.
// Note: machineCache may be nil, but if present, it must not have a restricted range.
func NewExecutionRun(
	ctxIn context.Context,
	initialMachineGetter func(context.Context) (MachineInterface, error),
	config *MachineCacheConfig,
) (*executionRun, error) {
	exec := &executionRun{}
	exec.Start(ctxIn, exec)
	exec.cache = NewMachineCache(exec.Context(), initialMachineGetter, config)
	return exec, nil
}

func (e *executionRun) Close() {
	go e.close.Do(func() {
		e.StopAndWait()
		if e.cache != nil {
			e.cache.Destroy(e.ParentContext())
		}
	})
}

func (e *executionRun) PrepareRange(start uint64, end uint64) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](e, func(ctx context.Context) (struct{}, error) {
		err := e.cache.SetRange(ctx, start, end)
		return struct{}{}, err
	})
}

func (e *executionRun) StepAt(position uint64) containers.PromiseInterface[*validator.MachineStepResult] {
	return stopwaiter.LaunchPromiseThread[*validator.MachineStepResult](e, func(ctx context.Context) (*validator.MachineStepResult, error) {
		var machine MachineInterface
		var err error
		if position == ^uint64(0) {
			machine, err = e.cache.FinalMachine(ctx)
		} else {
			// todo cache last machine
			machine, err = e.cache.MachineAt(ctx, position)
		}
		if err != nil {
			return nil, err
		}
		machineStep := machine.StepCount()
		if position != machineStep {
			machineRunning := machine.IsRunning()
			if machineRunning || machineStep > position {
				return nil, fmt.Errorf("machine is in wrong position want: %d, got: %d", position, machine.StepCount())
			}

		}
		result := &validator.MachineStepResult{
			Position:    machineStep,
			Status:      validator.MachineStatus(machine.Status()),
			GlobalState: machine.GlobalState(),
			Hash:        machine.Hash(),
		}
		return result, nil
	})
}

func (e *executionRun) ProofAt(position uint64) containers.PromiseInterface[[]byte] {
	return stopwaiter.LaunchPromiseThread[[]byte](e, func(ctx context.Context) ([]byte, error) {
		machine, err := e.cache.MachineAt(ctx, position)
		if err != nil {
			return nil, err
		}
		return machine.ProveNextStep(), nil
	})
}

func (e *executionRun) LastStep() containers.PromiseInterface[*validator.MachineStepResult] {
	return e.StepAt(^uint64(0))
}
