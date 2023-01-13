// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package server_arb

import (
	"context"
	"fmt"

	"github.com/offchainlabs/nitro/util/readymarker"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
)

type executionRun struct {
	stopwaiter.StopWaiter
	cache *MachineCache
}

type machineStep struct {
	readymarker.ReadyMarker
	result validator.MachineStepResult
}

func (s *machineStep) consumeMachine(machine MachineInterface, err error) {
	if err != nil {
		s.SignalReady(err)
		return
	}
	machineStep := machine.GetStepCount()
	if s.result.Position != machine.GetStepCount() {
		machineRunning := machine.IsRunning()
		if (machineRunning && s.result.Position != machineStep) || machineStep > s.result.Position {
			s.SignalReady(fmt.Errorf("machine is in wrong position want:%d, got: %d", s.result.Position, machine.GetStepCount()))
			return
		}
		s.result.Position = machineStep
	}
	s.result.Status = validator.MachineStatus(machine.Status())
	s.result.GlobalState = machine.GetGlobalState()
	s.result.Proof = machine.ProveNextStep()
	s.result.Hash = machine.Hash()
	s.SignalReady(nil)
}

func (s *machineStep) Get() (*validator.MachineStepResult, error) {
	if err := s.TestReady(); err != nil {
		return nil, err
	}
	return &s.result, nil
}

func (s *machineStep) Close() {}

// NewExecutionChallengeBackend creates a backend with the given arguments.
// Note: machineCache may be nil, but if present, it must not have a restricted range.
func NewExecutionRun(
	ctxIn context.Context,
	initialMachineGetter func(context.Context) (MachineInterface, error),
	config *MachineCacheConfig,
) (*executionRun, error) {
	exec := &executionRun{}
	exec.Start(ctxIn, exec)
	exec.cache = NewMachineCache(exec.GetContext(), initialMachineGetter, config)
	return exec, nil
}

func (e *executionRun) Close() {
	e.StopOnly()
}

func (e *executionRun) PrepareRange(start uint64, end uint64) {
	newCache := e.cache.SpawnCacheWithLimits(e.GetContext(), start, end)
	e.cache = newCache
}

func (e *executionRun) GetStepAt(position uint64) validator.MachineStep {
	mstep := &machineStep{
		ReadyMarker: readymarker.NewReadyMarker(),
		result: validator.MachineStepResult{
			Position: position,
		},
	}
	e.LaunchThread(func(ctx context.Context) {
		if position == ^uint64(0) {
			mstep.consumeMachine(e.cache.GetFinalMachine(ctx))
		} else {
			// todo cache last machine
			mstep.consumeMachine(e.cache.GetMachineAt(ctx, nil, position))
		}
	})
	return mstep
}

func (e *executionRun) GetLastStep() validator.MachineStep {
	return e.GetStepAt(^uint64(0))
}
