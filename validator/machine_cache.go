// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validator

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

// Manages a list of machines at various step counts.
// Aims to speed the retrieval of a machine at a given step count.
type MachineCache struct {
	machines            []MachineInterface
	firstMachineStep    uint64
	machineStepInterval uint64
	targetNumMachines   int
}

// `initialMachine` won't be mutated by this function.
func NewMachineCache(ctx context.Context, initialMachine MachineInterface, targetNumMachines int) (*MachineCache, error) {
	cache := &MachineCache{
		machines:          []MachineInterface{initialMachine},
		targetNumMachines: targetNumMachines,
		firstMachineStep:  initialMachine.GetStepCount(),
	}
	err := cache.populateInitialCache(ctx)
	if err != nil {
		return nil, err
	}
	return cache, nil
}

// `endSteps` should be the *total* step count at which the cache ends, not the number of steps from `initialMachine` to the end.
// `initialMachine` may be mutated by this function.
func NewMachineCacheWithEndSteps(ctx context.Context, initialMachine MachineInterface, targetNumMachines int, endSteps uint64) (*MachineCache, error) {
	startSteps := initialMachine.GetStepCount()
	if endSteps < startSteps {
		return nil, errors.Errorf("endSteps %v before initialMachine step count %v", endSteps, startSteps)
	}
	cache := &MachineCache{
		machines:            []MachineInterface{initialMachine.CloneMachineInterface()},
		targetNumMachines:   targetNumMachines,
		firstMachineStep:    startSteps,
		machineStepInterval: (endSteps - startSteps) / uint64(targetNumMachines+1),
	}
	for i := 1; i < targetNumMachines; i++ {
		if !initialMachine.IsRunning() {
			break
		}
		err := initialMachine.Step(ctx, cache.machineStepInterval)
		if err != nil {
			return nil, err
		}
		cache.machines = append(cache.machines, initialMachine.CloneMachineInterface())
	}
	return cache, nil
}

func (c *MachineCache) populateInitialCache(ctx context.Context) error {
	if c.targetNumMachines <= 1 {
		return nil
	}
	for {
		nextMachine := c.machines[len(c.machines)-1].CloneMachineInterface()
		if !nextMachine.IsRunning() {
			break
		}
		if len(c.machines) >= c.targetNumMachines {
			// Double the step interval between machines, which halves the number of machines.
			var pruned []MachineInterface
			for i, mach := range c.machines {
				// If i%2 == 0, this machine is no longer on the step interval.
				if i%2 == 1 {
					pruned = append(pruned, mach)
				}
			}
			c.machines = pruned
			c.machineStepInterval *= 2
		}
		if c.machineStepInterval == 0 {
			subCtx, cancel := context.WithTimeout(ctx, time.Minute)
			err := nextMachine.Step(subCtx, ^uint64(0))
			if err != nil {
				cancel()
				return err
			}
			cancel() // frees resources
			c.machineStepInterval = nextMachine.GetStepCount()
		} else {
			err := nextMachine.Step(ctx, c.machineStepInterval)
			if err != nil {
				return err
			}
		}
		c.machines = append(c.machines, nextMachine)
	}
	return nil
}

// Warning: don't mutate the result of this!
func (c *MachineCache) getClosestMachine(stepCount uint64) (MachineInterface, error) {
	if stepCount < c.firstMachineStep {
		return nil, errors.Errorf("requested step count %v but cache starts at %v", stepCount, c.firstMachineStep)
	}
	stepsFromStart := stepCount - c.firstMachineStep
	if c.machineStepInterval == 0 || stepsFromStart > c.machineStepInterval*uint64(len(c.machines)-1) {
		return c.machines[len(c.machines)-1], nil
	} else {
		return c.machines[stepsFromStart/c.machineStepInterval], nil
	}
}

// Gets a machine at a given step count, optionally using a passed in machine if that's the best option.
func (c *MachineCache) GetMachineAt(ctx context.Context, haveMachine MachineInterface, stepCount uint64) (MachineInterface, error) {
	closestMachine, err := c.getClosestMachine(stepCount)
	if err != nil {
		return nil, err
	}
	if haveMachine != nil && haveMachine.GetStepCount() >= closestMachine.GetStepCount() && haveMachine.GetStepCount() <= stepCount {
		closestMachine = haveMachine
	} else {
		closestMachine = closestMachine.CloneMachineInterface()
	}
	err = closestMachine.Step(ctx, stepCount-closestMachine.GetStepCount())
	if err != nil {
		return nil, err
	}
	if !closestMachine.ValidForStep(stepCount) {
		return nil, errors.Errorf("internal error: got machine with wrong step count %v looking for step count %v", closestMachine.GetStepCount(), stepCount)
	}
	return closestMachine, nil
}
