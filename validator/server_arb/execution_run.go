// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package server_arb

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"

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

func (e *executionRun) GetBigStepLeavesUpTo(toBigStep uint64, numOpcodesPerBigStep uint64) containers.PromiseInterface[[]common.Hash] {
	return stopwaiter.LaunchPromiseThread[[]common.Hash](e, func(ctx context.Context) ([]common.Hash, error) {
		var stateRoots []common.Hash
		for i := uint64(0); i <= toBigStep; i++ {
			position := i * numOpcodesPerBigStep
			machineStep, err := e.intermediateGetStepAt(ctx, position)
			if err != nil {
				return nil, err
			}
			stateRoots = append(stateRoots, machineStep.Hash)
		}
		return stateRoots, nil
	})
}

func (e *executionRun) GetSmallStepLeavesUpTo(bigStep uint64, toSmallStep uint64, numOpcodesPerBigStep uint64) containers.PromiseInterface[[]common.Hash] {
	return stopwaiter.LaunchPromiseThread[[]common.Hash](e, func(ctx context.Context) ([]common.Hash, error) {
		var stateRoots []common.Hash
		fromSmall := bigStep * numOpcodesPerBigStep
		toSmall := fromSmall + toSmallStep
		for i := fromSmall; i <= toSmall; i++ {
			machineStep, err := e.intermediateGetStepAt(ctx, i)
			if err != nil {
				return nil, err
			}
			stateRoots = append(stateRoots, machineStep.Hash)
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
		// todo cache last machine
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
