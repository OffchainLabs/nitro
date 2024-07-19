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

// NewExecutionRun creates a backend with the given arguments.
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

func (e *executionRun) GetMachineHashesWithStepSize(machineStartIndex, stepSize, maxIterations uint64) containers.PromiseInterface[[]common.Hash] {
	return stopwaiter.LaunchPromiseThread(e, func(ctx context.Context) ([]common.Hash, error) {
		return e.machineHashesWithStepSize(ctx, machineStartIndex, stepSize, maxIterations)
	})
}

func (e *executionRun) machineHashesWithStepSize(
	ctx context.Context,
	machineStartIndex,
	stepSize,
	maxIterations uint64,
) ([]common.Hash, error) {
	if stepSize == 0 {
		return nil, fmt.Errorf("step size cannot be 0")
	}
	if maxIterations == 0 {
		return nil, fmt.Errorf("max number of iterations cannot be 0")
	}
	machine, err := e.cache.GetMachineAt(ctx, machineStartIndex)
	if err != nil {
		return nil, err
	}
	log.Debug(fmt.Sprintf("Advanced machine to index %d, beginning hash computation", machineStartIndex))

	// In BOLD, the hash of a machine at index 0 is a special hash that is computed as the
	// `machineFinishedHash(gs)` where `gs` is the global state of the machine at index 0.
	// This is so that the hash aligns with the start state of the claimed challenge edge
	// at the level above, as required by the BOLD protocol.
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
	if maxIterations == 1 {
		return machineHashes, nil
	}

	logInterval := maxIterations / 20 // Log every 5% progress
	if logInterval == 0 {
		logInterval = 1
	}

	start := time.Now()
	for i := uint64(0); i < maxIterations; i++ {
		// The absolute program counter the machine should be in after stepping.
		absoluteMachineIndex := machineStartIndex + stepSize*(i+1)

		// Advance the machine in step size increments.
		if err := machine.Step(ctx, stepSize); err != nil {
			return nil, fmt.Errorf("failed to step machine to position %d: %w", absoluteMachineIndex, err)
		}
		if i%logInterval == 0 || i == maxIterations-1 {
			progressPercent := (float64(i+1) / float64(maxIterations)) * 100
			log.Info(
				fmt.Sprintf(
					"Computing BOLD subchallenge progress: %.2f%% - %d of %d hashes",
					progressPercent,
					i+1,
					maxIterations,
				),
				"machinePosition", i*stepSize+machineStartIndex,
				"timeSinceStart", time.Since(start),
				"stepSize", stepSize,
				"startHash", startHash,
				"machineStartIndex", machineStartIndex,
				"maxIterations", maxIterations,
			)
		}
		machineHashes = append(machineHashes, machine.Hash())
		if uint64(len(machineHashes)) == maxIterations {
			log.Info("Reached the max number of iterations for the hashes needed to open a subchallenge")
			break
		}
		if !machine.IsRunning() {
			log.Info("Machine no longer running, exiting early from hash computation loop")
			break
		}
	}
	log.Info(
		"Successfully finished computing the data needed for opening a subchallenge",
		"stepSize", stepSize,
		"startHash", startHash,
		"machineStartIndex", machineStartIndex,
		"numberOfHashesComputed", len(machineHashes),
		"maxIterations", maxIterations,
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
		return machine.ProveNextStep(), nil
	})
}

func (e *executionRun) GetLastStep() containers.PromiseInterface[*validator.MachineStepResult] {
	return e.GetStepAt(^uint64(0))
}

func machineFinishedHash(gs validator.GoGlobalState) common.Hash {
	return crypto.Keccak256Hash([]byte("Machine finished:"), gs.Hash().Bytes())
}
