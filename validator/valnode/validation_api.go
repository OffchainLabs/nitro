// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package valnode

import (
	"context"
	"encoding/base64"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/server_arb"
)

type ValidationServerAPI struct {
	spawner validator.ValidationSpawner
}

func (a *ValidationServerAPI) Name() string {
	return a.spawner.Name()
}

func (a *ValidationServerAPI) Capacity() int {
	return a.spawner.Capacity()
}

// This is for backwards compatibility, should be removed in the future.
func (a *ValidationServerAPI) Room() int {
	return a.spawner.Capacity()
}

func (a *ValidationServerAPI) Validate(ctx context.Context, entry *server_api.InputJSON, moduleRoot common.Hash) (validator.GoGlobalState, error) {
	valInput, err := server_api.ValidationInputFromJson(entry)
	if err != nil {
		return validator.GoGlobalState{}, err
	}
	valRun := a.spawner.Launch(valInput, moduleRoot)
	return valRun.Await(ctx)
}

func (a *ValidationServerAPI) WasmModuleRoots() ([]common.Hash, error) {
	return a.spawner.WasmModuleRoots()
}

func (a *ValidationServerAPI) StylusArchs() ([]rawdb.WasmTarget, error) {
	return a.spawner.StylusArchs(), nil
}

func NewValidationServerAPI(spawner validator.ValidationSpawner) *ValidationServerAPI {
	return &ValidationServerAPI{spawner}
}

type execRunEntry struct {
	run      validator.ExecutionRun
	accessed time.Time
}

type ExecServerAPI struct {
	stopwaiter.StopWaiter
	ValidationServerAPI
	execSpawner validator.ExecutionSpawner

	config server_arb.ArbitratorSpawnerConfigFetcher

	runIdLock sync.Mutex
	nextId    uint64
	runs      map[uint64]*execRunEntry
}

func NewExecutionServerAPI(valSpawner validator.ValidationSpawner, execution validator.ExecutionSpawner, config server_arb.ArbitratorSpawnerConfigFetcher) *ExecServerAPI {
	return &ExecServerAPI{
		ValidationServerAPI: *NewValidationServerAPI(valSpawner),
		execSpawner:         execution,
		nextId:              rand.Uint64(), // good-enough to avoid reusing ids after reboot
		runs:                make(map[uint64]*execRunEntry),
		config:              config,
	}
}

func (a *ExecServerAPI) CreateExecutionRun(ctx context.Context, wasmModuleRoot common.Hash, jsonInput *server_api.InputJSON, useBoldMachineOptional *bool) (uint64, error) {
	if a.Stopped() {
		return 0, errors.New("ExecServerAPI is stopped")
	}
	input, err := server_api.ValidationInputFromJson(jsonInput)
	if err != nil {
		return 0, err
	}
	useBoldMachine := false
	if useBoldMachineOptional != nil {
		useBoldMachine = *useBoldMachineOptional
	}
	execRun, err := a.execSpawner.CreateExecutionRun(wasmModuleRoot, input, useBoldMachine).Await(ctx)
	if err != nil {
		return 0, err
	}
	a.runIdLock.Lock()
	defer a.runIdLock.Unlock()
	newId := a.nextId
	a.nextId++
	a.runs[newId] = &execRunEntry{execRun, time.Now()}
	return newId, nil
}

func (a *ExecServerAPI) removeOldRuns(ctx context.Context) time.Duration {
	oldestKept := time.Now().Add(-1 * a.config().ExecutionRunTimeout)
	a.runIdLock.Lock()
	defer a.runIdLock.Unlock()
	for id, entry := range a.runs {
		if entry.accessed.Before(oldestKept) {
			entry.run.Close()
			delete(a.runs, id)
		}
	}
	return a.config().ExecutionRunTimeout / 5
}

func (a *ExecServerAPI) Start(ctx_in context.Context) {
	a.StopWaiter.Start(ctx_in, a)
	a.CallIteratively(a.removeOldRuns)
}

func (a *ExecServerAPI) StopAndWait() {
	a.StopWaiter.StopAndWait()
	a.runIdLock.Lock()
	defer a.runIdLock.Unlock()
	for _, entry := range a.runs {
		entry.run.Close()
	}
}

var errRunNotFound error = errors.New("run not found")

func (a *ExecServerAPI) getRun(id uint64) (validator.ExecutionRun, error) {
	if a.Stopped() {
		return nil, errRunNotFound
	}
	a.runIdLock.Lock()
	defer a.runIdLock.Unlock()
	entry := a.runs[id]
	if entry == nil {
		return nil, errRunNotFound
	}
	entry.accessed = time.Now()
	return entry.run, nil
}

func (a *ExecServerAPI) GetStepAt(ctx context.Context, execid uint64, position uint64) (*server_api.MachineStepResultJson, error) {
	run, err := a.getRun(execid)
	if err != nil {
		return nil, err
	}
	step := run.GetStepAt(position)
	res, err := step.Await(ctx)
	if err != nil {
		return nil, err
	}
	return server_api.MachineStepResultToJson(res), nil
}

func (a *ExecServerAPI) GetMachineHashesWithStepSize(ctx context.Context, execid, fromStep, stepSize, maxIterations uint64) ([]common.Hash, error) {
	run, err := a.getRun(execid)
	if err != nil {
		return nil, err
	}
	hashesInRange := run.GetMachineHashesWithStepSize(fromStep, stepSize, maxIterations)
	res, err := hashesInRange.Await(ctx)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (a *ExecServerAPI) GetProofAt(ctx context.Context, execid uint64, position uint64) (string, error) {
	run, err := a.getRun(execid)
	if err != nil {
		return "", err
	}
	promise := run.GetProofAt(position)
	res, err := promise.Await(ctx)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(res), nil
}

func (a *ExecServerAPI) PrepareRange(ctx context.Context, execid uint64, start, end uint64) error {
	run, err := a.getRun(execid)
	if err != nil {
		return err
	}
	_, err = run.PrepareRange(start, end).Await(ctx)
	return err
}

func (a *ExecServerAPI) ExecKeepAlive(ctx context.Context, execid uint64) error {
	_, err := a.getRun(execid)
	if err != nil {
		return err
	}
	return nil
}

func (a *ExecServerAPI) CheckAlive(ctx context.Context, execid uint64) error {
	run, err := a.getRun(execid)
	if err != nil {
		return err
	}
	return run.CheckAlive(ctx)
}

func (a *ExecServerAPI) CloseExec(execid uint64) {
	// Protect map access with runIdLock to avoid concurrent map read/write.
	// Call Close() outside the lock to avoid holding the mutex during a potentially long operation.
	a.runIdLock.Lock()
	entry := a.runs[execid]
	if entry != nil {
		delete(a.runs, execid)
	}
	a.runIdLock.Unlock()

	if entry != nil {
		entry.run.Close()
	}
}
