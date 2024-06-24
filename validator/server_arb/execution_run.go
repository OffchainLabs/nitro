// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package server_arb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/log"
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
		var machine MachineInterface
		var err error
		if position == ^uint64(0) {
			machine, err = e.cache.GetFinalMachine(ctx)
		} else {
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
	})
}

func (e *executionRun) GetMachineHashesWithStepSize(machineStartIndex, stepSize, requiredNumHashes uint64) containers.PromiseInterface[[]common.Hash] {
	return stopwaiter.LaunchPromiseThread(e, func(ctx context.Context) ([]common.Hash, error) {
		return e.machineHashesWithStepSize(ctx, machineStartIndex, stepSize, requiredNumHashes)
	})
}

func (e *executionRun) machineHashesWithStepSize(
	ctx context.Context,
	machineStartIndex,
	stepSize,
	requiredNumHashes uint64,
) ([]common.Hash, error) {
	if stepSize == 0 {
		return nil, fmt.Errorf("step size cannot be 0")
	}
	if requiredNumHashes == 0 {
		return nil, fmt.Errorf("required number of hashes cannot be 0")
	}
	machine, err := e.cache.GetMachineAt(ctx, machineStartIndex)
	if err != nil {
		return nil, err
	}
	log.Debug(fmt.Sprintf("Advanced machine to index %d, beginning hash computation", machineStartIndex))

	// If the machine is starting at index 0, we always want to start at the "Machine finished" global state status
	// to align with the machine hashes that the inbox machine will produce.
	var machineHashes []common.Hash
	if machineStartIndex == 0 {
		gs := machine.GetGlobalState()
		log.Debug(fmt.Sprintf("Start global state for machine index 0: %+v", gs))
		machineHashes = append(machineHashes, machineFinishedHash(gs))
	} else {
		// Otherwise, we simply append the machine hash at the specified start index.
		machineHashes = append(machineHashes, machine.Hash())
	}
	startHash := machineHashes[0]

	// If we only want 1 hash, we can return early.
	if requiredNumHashes == 1 {
		return machineHashes, nil
	}

	logInterval := requiredNumHashes / 20 // Log every 5% progress
	if logInterval == 0 {
		logInterval = 1
	}

	start := time.Now()
	for numIterations := uint64(0); numIterations < requiredNumHashes; numIterations++ {
		// The absolute program counter the machine should be in after stepping.
		absoluteMachineIndex := machineStartIndex + stepSize*(numIterations+1)

		// Advance the machine in step size increments.
		if err := machine.Step(ctx, stepSize); err != nil {
			return nil, fmt.Errorf("failed to step machine to position %d: %w", absoluteMachineIndex, err)
		}
		if numIterations%logInterval == 0 || numIterations == requiredNumHashes-1 {
			progressPercent := (float64(numIterations+1) / float64(requiredNumHashes)) * 100
			log.Info(
				fmt.Sprintf(
					"Computing BOLD subchallenge progress: %.2f%% - %d of %d hashes needed",
					progressPercent,
					numIterations+1,
					requiredNumHashes,
				),
				"machinePosition", numIterations*stepSize+machineStartIndex,
				"timeSinceStart", time.Since(start),
				"stepSize", stepSize,
				"startHash", startHash,
				"machineStartIndex", machineStartIndex,
				"numDesiredLeaves", requiredNumHashes,
			)
		}
		machineHashes = append(machineHashes, machine.Hash())
		if uint64(len(machineHashes)) == requiredNumHashes {
			break
		}
	}
	log.Info(
		"Successfully finished computing the data needed for opening a subchallenge",
		"stepSize", stepSize,
		"startHash", startHash,
		"machineStartIndex", machineStartIndex,
		"numDesiredLeaves", requiredNumHashes,
		"finishedHash", machineHashes[len(machineHashes)-1],
		"finishedGlobalState", fmt.Sprintf("%+v", machine.GetGlobalState()),
	)
	return machineHashes, nil
}

func (e *executionRun) GetProofAt(position uint64) containers.PromiseInterface[[]byte] {
	return stopwaiter.LaunchPromiseThread[[]byte](e, func(ctx context.Context) ([]byte, error) {
		machine, err := e.cache.GetMachineAt(ctx, position)
		if err != nil {
			return nil, err
		}
		log.Info("Getting machine proof at position", "position", position)
		log.Info(
			"Machine start global state at OSP is",
			"globalState",
			fmt.Sprintf("%+v", machine.GetGlobalState()),
			"machineHash",
			fmt.Sprintf("%#x", machine.Hash()),
		)
		return machine.ProveNextStep(), nil
	})
}

func (e *executionRun) GetLastStep() containers.PromiseInterface[*validator.MachineStepResult] {
	return e.GetStepAt(^uint64(0))
}

func (e *executionRun) CheckAlive(ctx context.Context) error {
	return nil
}

func machineFinishedHash(gs validator.GoGlobalState) common.Hash {
	return crypto.Keccak256Hash([]byte("Machine finished:"), gs.Hash().Bytes())
}
