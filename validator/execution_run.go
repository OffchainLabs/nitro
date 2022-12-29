// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validator

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/pkg/errors"
)

type ExecutionRun interface {
	GetStepAt(uint64) MachineStep
	GetLastStep() MachineStep
	PrepareRange(uint64, uint64)
	Close()
}

type MachineStep interface {
	ReadyMarker
	Hash() (common.Hash, error)
	Proof() ([]byte, error)
	Position() (uint64, error)
	Status() (MachineStatus, error)
	GlobalState() (GoGlobalState, error)
	Close()
}

type executionRun struct {
	stopwaiter.StopWaiter
	cache *MachineCache
}

type machineStep struct {
	readyMarker
	position uint64
	status   MachineStatus
	state    GoGlobalState
	hash     common.Hash
	proof    []byte
}

func (s *machineStep) consumeMachine(machine MachineInterface, err error) {
	if err != nil {
		s.signalReady(err)
		return
	}
	if s.position == ^uint64(0) {
		s.position = machine.GetStepCount()
	} else if s.position != machine.GetStepCount() {
		s.signalReady(errors.New("machine is in wrong position"))
	}
	s.status = MachineStatus(machine.Status())
	s.state = machine.GetGlobalState()
	s.proof = machine.ProveNextStep()
	s.hash = machine.Hash()
	s.signalReady(nil)
}

func (s *machineStep) Hash() (common.Hash, error) {
	if !s.Ready() {
		return common.Hash{}, ErrNotReady
	}
	return s.hash, nil
}

func (s *machineStep) Proof() ([]byte, error) {
	if !s.Ready() {
		return nil, ErrNotReady
	}
	return s.proof, nil
}

func (s *machineStep) Position() (uint64, error) {
	if !s.Ready() {
		return 0, ErrNotReady
	}
	return s.position, nil
}

func (s *machineStep) Status() (MachineStatus, error) {
	if !s.Ready() {
		return 0, ErrNotReady
	}
	return s.status, nil
}

func (s *machineStep) GlobalState() (GoGlobalState, error) {
	if !s.Ready() {
		return GoGlobalState{}, ErrNotReady
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

func (e *executionRun) GetStepAt(position uint64) MachineStep {
	mstep := &machineStep{
		readyMarker: newReadyMarker(),
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

func (e *executionRun) GetLastStep() MachineStep {
	return e.GetStepAt(^uint64(0))
}
