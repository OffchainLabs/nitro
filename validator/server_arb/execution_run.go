// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package server_arb

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
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
	position uint64
	status   validator.MachineStatus
	state    validator.GoGlobalState
	hash     common.Hash
	proof    []byte
}

func (s *machineStep) consumeMachine(machine MachineInterface, err error) {
	if err != nil {
		s.SignalReady(err)
		return
	}
	machineStep := machine.GetStepCount()
	if s.position != machine.GetStepCount() {
		machineRunning := machine.IsRunning()
		if (machineRunning && s.position != machineStep) || machineStep > s.position {
			s.SignalReady(fmt.Errorf("machine is in wrong position want:%d, got: %d", s.position, machine.GetStepCount()))
			return
		}
		s.position = machineStep
	}
	s.status = validator.MachineStatus(machine.Status())
	s.state = machine.GetGlobalState()
	s.proof = machine.ProveNextStep()
	s.hash = machine.Hash()
	s.SignalReady(nil)
}

func (s *machineStep) Hash() (common.Hash, error) {
	if err := s.TestReady(); err != nil {
		return common.Hash{}, err
	}
	return s.hash, nil
}

func (s *machineStep) Proof() ([]byte, error) {
	if err := s.TestReady(); err != nil {
		return nil, err
	}
	return s.proof, nil
}

func (s *machineStep) Position() (uint64, error) {
	if err := s.TestReady(); err != nil {
		return 0, err
	}
	return s.position, nil
}

func (s *machineStep) Status() (validator.MachineStatus, error) {
	if err := s.TestReady(); err != nil {
		return 0, err
	}
	return s.status, nil
}

func (s *machineStep) GlobalState() (validator.GoGlobalState, error) {
	if err := s.TestReady(); err != nil {
		return validator.GoGlobalState{}, err
	}
	return s.state, nil
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
		position:    position,
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
