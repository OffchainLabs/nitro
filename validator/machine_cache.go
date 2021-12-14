//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package validator

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

// Manages a list of machines at various step counts.
// Aims to speed the retrieval of a machine at a given step count.
type MachineCache struct {
	machines            []*ArbitratorMachine
	firstMachineStep    uint64
	machineStepInterval uint64
	targetNumMachines   int
}

func NewMachineCache(ctx context.Context, initialMachine *ArbitratorMachine, targetNumMachines int) *MachineCache {
	cache := &MachineCache{
		machines:          []*ArbitratorMachine{initialMachine},
		targetNumMachines: targetNumMachines,
		firstMachineStep:  initialMachine.GetStepCount(),
	}
	cache.populateInitialCache(ctx)
	return cache
}

// `endSteps` should be the *total* step count at which the cache ends, not the number of steps from `initialMachine` to the end.
func NewMachineCacheWithEndSteps(ctx context.Context, initialMachine *ArbitratorMachine, targetNumMachines int, endSteps uint64) (*MachineCache, error) {
	startSteps := initialMachine.GetStepCount()
	if endSteps < startSteps {
		return nil, errors.Errorf("endSteps %v before initialMachine step count %v", endSteps, startSteps)
	}
	cache := &MachineCache{
		machines:            []*ArbitratorMachine{initialMachine.Clone()},
		targetNumMachines:   targetNumMachines,
		firstMachineStep:    startSteps,
		machineStepInterval: (endSteps - startSteps) / uint64(targetNumMachines+1),
	}
	for i := 1; i < targetNumMachines; i++ {
		if !initialMachine.IsRunning() {
			break
		}
		initialMachine.Step(ctx, cache.machineStepInterval)
		cache.machines = append(cache.machines, initialMachine.Clone())
	}
	return cache, nil
}

func (c *MachineCache) populateInitialCache(ctx context.Context) {
	if c.targetNumMachines <= 1 {
		return
	}
	for {
		nextMachine := c.machines[len(c.machines)-1].Clone()
		if !nextMachine.IsRunning() {
			break
		}
		if len(c.machines) >= c.targetNumMachines {
			// Double the step interval between machines, which halves the number of machines.
			var pruned []*ArbitratorMachine
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
			nextMachine.Step(subCtx, ^uint64(0))
			cancel() // frees resources
			c.machineStepInterval = nextMachine.GetStepCount()
		} else {
			nextMachine.Step(ctx, c.machineStepInterval)
		}
		c.machines = append(c.machines, nextMachine)
	}
}

func (c *MachineCache) getClosestMachine(stepCount uint64) (*ArbitratorMachine, error) {
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

func (c *MachineCache) GetMachineAt(ctx context.Context, haveMachine *ArbitratorMachine, stepCount uint64) (*ArbitratorMachine, error) {
	closestMachine, err := c.getClosestMachine(stepCount)
	if err != nil {
		return nil, err
	}
	if haveMachine != nil && haveMachine.GetStepCount() >= closestMachine.GetStepCount() && haveMachine.GetStepCount() <= stepCount {
		closestMachine = haveMachine
	} else {
		closestMachine = closestMachine.Clone()
	}
	closestMachine.Step(ctx, stepCount-closestMachine.GetStepCount())
	return closestMachine, nil
}
