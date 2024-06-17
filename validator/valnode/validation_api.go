// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package valnode

import (
	"context"
	"encoding/base64"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

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

func (a *ValidationServerAPI) Room() int {
	return a.spawner.Room()
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

	config server_arb.ArbitratorSpawnerConfigFecher

	runIdLock sync.Mutex
	nextId    uint64
	runs      map[uint64]*execRunEntry
}

func NewExecutionServerAPI(valSpawner validator.ValidationSpawner, execution validator.ExecutionSpawner, config server_arb.ArbitratorSpawnerConfigFecher) *ExecServerAPI {
	return &ExecServerAPI{
		ValidationServerAPI: *NewValidationServerAPI(valSpawner),
		execSpawner:         execution,
		nextId:              rand.Uint64(), // good-enough to aver reusing ids after reboot
		runs:                make(map[uint64]*execRunEntry),
		config:              config,
	}
}

func (a *ExecServerAPI) CreateExecutionRun(ctx context.Context, wasmModuleRoot common.Hash, jsonInput *server_api.InputJSON) (uint64, error) {
	input, err := server_api.ValidationInputFromJson(jsonInput)
	if err != nil {
		return 0, err
	}
	execRun, err := a.execSpawner.CreateExecutionRun(wasmModuleRoot, input).Await(ctx)
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

func (a *ExecServerAPI) LatestWasmModuleRoot(ctx context.Context) (common.Hash, error) {
	return a.execSpawner.LatestWasmModuleRoot().Await(ctx)
}

func (a *ExecServerAPI) removeOldRuns(ctx context.Context) time.Duration {
	oldestKept := time.Now().Add(-1 * a.config().ExecutionRunTimeout)
	a.runIdLock.Lock()
	defer a.runIdLock.Unlock()
	for id, entry := range a.runs {
		if entry.accessed.Before(oldestKept) {
			delete(a.runs, id)
		}
	}
	return a.config().ExecutionRunTimeout / 5
}

func (a *ExecServerAPI) Start(ctx_in context.Context) {
	a.StopWaiter.Start(ctx_in, a)
	a.CallIteratively(a.removeOldRuns)
}

func (a *ExecServerAPI) WriteToFile(ctx context.Context, jsonInput *server_api.InputJSON, expOut validator.GoGlobalState, moduleRoot common.Hash) error {
	input, err := server_api.ValidationInputFromJson(jsonInput)
	if err != nil {
		return err
	}
	_, err = a.execSpawner.WriteToFile(input, expOut, moduleRoot).Await(ctx)
	return err
}

var errRunNotFound error = errors.New("run not found")

func (a *ExecServerAPI) getRun(id uint64) (validator.ExecutionRun, error) {
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

func (a *ExecServerAPI) GetMachineHashesWithStepSize(ctx context.Context, execid, fromStep, stepSize, numRequiredHashes uint64) ([]common.Hash, error) {
	run, err := a.getRun(execid)
	if err != nil {
		return nil, err
	}
	leavesInRange := run.GetMachineHashesWithStepSize(fromStep, stepSize, numRequiredHashes)
	res, err := leavesInRange.Await(ctx)
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

func (a *ExecServerAPI) CloseExec(execid uint64) {
	a.runIdLock.Lock()
	defer a.runIdLock.Unlock()
	run, found := a.runs[execid]
	if !found {
		return
	}
	run.run.Close()
	delete(a.runs, execid)
}
