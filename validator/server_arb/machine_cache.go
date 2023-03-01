// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package server_arb

import (
	"context"
	"fmt"
	"sync"

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

// MachineCache manages a list of machines at various step counts.
// Aims to speed the retrieval of a machine at a given step count.
type MachineCache struct {
	containers.Promise[struct{}]
	zeroStepMachine     MachineInterface
	finalMachine        MachineInterface
	machines            []MachineInterface
	firstMachineStep    uint64
	machineStepInterval uint64
	config              *MachineCacheConfig

	lastMachine     MachineInterface
	lastMachineLock sync.Mutex
}

type MachineCacheConfig struct {
	CachedChallengeMachines int    `koanf:"cached-challenge-machines"`
	InitialSteps            uint64 `koanf:"initial-steps"`
}

var DefaultMachineCacheConfig = MachineCacheConfig{
	CachedChallengeMachines: 4,
	InitialSteps:            100000,
}

func MachineCacheConfigConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".initial-steps", DefaultMachineCacheConfig.InitialSteps, "initial steps between machines")
	f.Int(prefix+".cached-challenge-machines", DefaultMachineCacheConfig.CachedChallengeMachines, "how many machines to store in cache while working on a challenge")
}

// `initialMachine` won't be mutated by this function.
func NewMachineCache(ctx context.Context, initialMachineGetter func(context.Context) (MachineInterface, error), config *MachineCacheConfig) *MachineCache {
	cache := &MachineCache{
		Promise: containers.NewPromise[struct{}](),
		config:  config,
	}
	go func() {
		zeroStepMachine, err := initialMachineGetter(ctx)
		if err == nil && zeroStepMachine.GetStepCount() != 0 {
			zeroStepMachine.Destroy()
			err = errors.New("initialMachine not at step count 0")
		}
		if err != nil {
			cache.ProduceError(err)
			return
		}
		zeroStepMachine.Freeze()
		cache.zeroStepMachine = zeroStepMachine
		cache.machines = []MachineInterface{zeroStepMachine}
		cache.firstMachineStep = 0
		cache.machineStepInterval = config.InitialSteps
		err = cache.populateInitialCache(ctx, ^uint64(0))
		if err != nil {
			cache.ProduceError(err)
			return
		}
		cache.finalMachine = cache.machines[len(cache.machines)-1]
		cache.finalMachine.Freeze()
		cache.Produce(struct{}{})
	}()
	return cache
}

func (c *MachineCache) SpawnCacheWithLimits(ctx context.Context, start uint64, end uint64) *MachineCache {
	newInterval := (start - end) / uint64(c.config.CachedChallengeMachines)
	if start == c.firstMachineStep && newInterval == c.machineStepInterval {
		return c
	}
	newCache := &MachineCache{
		Promise: containers.NewPromise[struct{}](),
		config:  c.config,
	}
	go func() {
		_, err := c.Await(ctx)
		if err != nil {
			newCache.ProduceError(err)
			return
		}
		newCache.zeroStepMachine = c.zeroStepMachine
		newCache.finalMachine = c.finalMachine
		closest, err := c.getClosestMachine(start)
		if err != nil {
			newCache.ProduceError(err)
			return
		}
		closestStep := closest.GetStepCount()
		var initial MachineInterface
		if closestStep > start {
			newCache.ProduceError(fmt.Errorf("initial machine step too large %d > %d", closestStep, start))
			return
		}
		if closestStep < start {
			initial = closest.CloneMachineInterface()
			err := initial.Step(ctx, start-closestStep)
			if err != nil {
				newCache.ProduceError(err)
				return
			}
			initial.Freeze()
		} else {
			initial = closest
		}
		newCache.machines = []MachineInterface{initial}
		newCache.firstMachineStep = start
		newCache.machineStepInterval = newInterval
		err = newCache.populateInitialCache(ctx, newInterval*uint64(c.config.CachedChallengeMachines))
		if err != nil {
			newCache.ProduceError(err)
		} else {
			newCache.Produce(struct{}{})
		}
	}()
	return newCache
}

func (c *MachineCache) populateInitialCache(ctx context.Context, target_step uint64) error {
	for {
		nextMachine := c.machines[len(c.machines)-1].CloneMachineInterface()
		if !nextMachine.IsRunning() {
			break
		}
		if nextMachine.GetStepCount() >= target_step {
			break
		}
		err := nextMachine.Step(ctx, c.machineStepInterval)
		if err != nil {
			return err
		}
		nextMachine.Freeze()
		if len(c.machines) >= c.config.CachedChallengeMachines {
			// Double the step interval between machines, which halves the number of machines.
			var pruned []MachineInterface
			for i, mach := range c.machines {
				// If i%2 == 1, this machine is no longer on the step interval.
				if i%2 == 0 {
					pruned = append(pruned, mach)
				} else {
					mach.Destroy()
				}
			}
			c.machines = pruned
			c.machineStepInterval *= 2
		}
		c.machines = append(c.machines, nextMachine)
	}
	return nil
}

// Warning: don't mutate the result of this!
func (c *MachineCache) getClosestMachine(stepCount uint64) (MachineInterface, error) {
	if stepCount < c.firstMachineStep {
		return c.zeroStepMachine, nil
	}
	stepsFromStart := stepCount - c.firstMachineStep
	if c.machineStepInterval == 0 || stepsFromStart > c.machineStepInterval*uint64(len(c.machines)-1) {
		return c.machines[len(c.machines)-1], nil
	} else {
		return c.machines[stepsFromStart/c.machineStepInterval], nil
	}
}

func (c *MachineCache) getLastMachine() MachineInterface {
	c.lastMachineLock.Lock()
	defer c.lastMachineLock.Unlock()
	return c.lastMachine
}

func (c *MachineCache) setLastMachine(machine MachineInterface) {
	c.lastMachineLock.Lock()
	prevLast := c.lastMachine
	c.lastMachine = machine
	c.lastMachineLock.Unlock()
	if prevLast != nil && prevLast != machine {
		prevLast.Destroy()
	}
}

// GetMachineAt a given step count, optionally using a passed in machine if that's the best option.
func (c *MachineCache) GetMachineAt(ctx context.Context, stepCount uint64) (MachineInterface, error) {
	_, err := c.Await(ctx)
	if err != nil {
		return nil, err
	}
	closestMachine, err := c.getClosestMachine(stepCount)
	if err != nil {
		return nil, err
	}
	lastMachine := c.getLastMachine()
	if lastMachine != nil && lastMachine.GetStepCount() >= closestMachine.GetStepCount() && lastMachine.GetStepCount() <= stepCount {
		closestMachine = lastMachine
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
	c.setLastMachine(closestMachine)
	return closestMachine, nil
}

func (c *MachineCache) GetFinalMachine(ctx context.Context) (MachineInterface, error) {
	_, err := c.Await(ctx)
	if err != nil {
		return nil, err
	}
	return c.finalMachine, nil
}
