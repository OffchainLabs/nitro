// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package server_arb

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

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
	exec.cache = NewMachineCache(exec.GetContext(), initialMachineGetter, config)
	return exec, nil
}

func (e *executionRun) Close() {
	go e.close.Do(func() {
		e.StopAndWait()
		if e.cache != nil {
			e.cache.Destroy(e.GetParentContext())
		}
	})
}

func (e *executionRun) PrepareRange(start uint64, end uint64) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](e, func(ctx context.Context) (struct{}, error) {
		err := e.cache.SetRange(ctx, start, end)
		return struct{}{}, err
	})
}

func (e *executionRun) GetStepAt(position uint64) containers.PromiseInterface[*validator.MachineStepResult] {
	return stopwaiter.LaunchPromiseThread[*validator.MachineStepResult](e, func(ctx context.Context) (*validator.MachineStepResult, error) {
		return e.intermediateGetStepAt(ctx, position)
	})
}

func (e *executionRun) GetLeavesWithStepSize(machineStartIndex, stepSize, numDesiredLeaves uint64) containers.PromiseInterface[[]common.Hash] {
	return stopwaiter.LaunchPromiseThread[[]common.Hash](e, func(ctx context.Context) ([]common.Hash, error) {
		machine, err := e.cache.GetMachineAt(ctx, machineStartIndex)
		if err != nil {
			return nil, err
		}
		// If the machine is starting at index 0, we always want to start at the "Machine finished" global state status
		// to align with the state roots that the inbox machine will produce.
		var stateRoots []common.Hash
		startGlobalState := machine.GetGlobalState()
		if machineStartIndex == 0 {
			hash := crypto.Keccak256Hash([]byte("Machine finished:"), startGlobalState.Hash().Bytes())
			stateRoots = append(stateRoots, hash)
		} else {
			// Otherwise, we simply append the machine hash at the specified start index.
			stateRoots = append(stateRoots, machine.Hash())
		}

		// If we only want 1 state root, we can return early.
		if numDesiredLeaves == 1 {
			return stateRoots, nil
		}
		for numIterations := uint64(0); numIterations < numDesiredLeaves; numIterations++ {
			// The absolute opcode position the machine should be in after stepping.
			position := machineStartIndex + stepSize*(numIterations+1)
			// Advance the machine in step size increments.
			if err := machine.Step(ctx, stepSize); err != nil {
				return nil, fmt.Errorf("failed to step machine to position %d: %w", position, err)
			}
			// If the machine reached the finished state, we can break out of the loop and append to
			// our state roots slice a finished machine hash.
			machineStep := machine.GetStepCount()
			if validator.MachineStatus(machine.Status()) == validator.MachineStatusFinished {
				gs := machine.GetGlobalState()
				hash := crypto.Keccak256Hash([]byte("Machine finished:"), gs.Hash().Bytes())
				stateRoots = append(stateRoots, hash)
				break
			}
			// Otherwise, if the position and machine step mismatch and the machine is running, something went wrong.
			if position != machineStep {
				machineRunning := machine.IsRunning()
				if machineRunning || machineStep > position {
					return nil, fmt.Errorf("machine is in wrong position want: %d, got: %d", position, machineStep)
				}
			}
			hash := machine.Hash()
			stateRoots = append(stateRoots, hash)
		}
		// If the machine finished in less than the number of hashes we anticipate, we pad
		// to the expected value by repeating the last machine hash until the state roots are the correct
		// length.
		for uint64(len(stateRoots)) < numDesiredLeaves {
			stateRoots = append(stateRoots, stateRoots[len(stateRoots)-1])
		}
		return stateRoots, nil
	})
}

func (e *executionRun) intermediateGetStepAt(ctx context.Context, position uint64) (*validator.MachineStepResult, error) {
	var machine MachineInterface
	var err error
	if position == ^uint64(0) {
		machine, err = e.cache.GetFinalMachine(ctx)
	} else {
		// todo cache last machina
		machine, err = e.cache.GetMachineAt(ctx, position)
	}
	if err != nil {
		return nil, err
	}
	machineStep := machine.GetStepCount()
	if position != machineStep {
		machineRunning := machine.IsRunning()
		if machineRunning || machineStep > position {
			return nil, fmt.Errorf("machine is in wrong position want: %d, got: %d", position, machine.GetStepCount())
		}

	}
	result := &validator.MachineStepResult{
		Position:    machineStep,
		Status:      validator.MachineStatus(machine.Status()),
		GlobalState: machine.GetGlobalState(),
		Hash:        machine.Hash(),
	}
	return result, nil
}

func (e *executionRun) GetProofAt(position uint64) containers.PromiseInterface[[]byte] {
	return stopwaiter.LaunchPromiseThread[[]byte](e, func(ctx context.Context) ([]byte, error) {
		machine, err := e.cache.GetMachineAt(ctx, position)
		if err != nil {
			return nil, err
		}
		return machine.ProveNextStep(), nil
	})
}

func (e *executionRun) GetLastStep() containers.PromiseInterface[*validator.MachineStepResult] {
	return e.GetStepAt(^uint64(0))
}
