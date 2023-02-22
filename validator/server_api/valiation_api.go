package server_api

import (
	"context"
	"encoding/base64"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/validator"
)

const Namespace string = "validation"

type ValidationServerAPI struct {
	spawner validator.ValidationSpawner
}

func (a *ValidationServerAPI) Name() string {
	return a.spawner.Name()
}

func (a *ValidationServerAPI) Room() int {
	return a.spawner.Room()
}

func (a *ValidationServerAPI) Validate(ctx context.Context, entry *ValidationInputJson, moduleRoot common.Hash) (validator.GoGlobalState, error) {
	valInput, err := ValidationInputFromJson(entry)
	if err != nil {
		return validator.GoGlobalState{}, err
	}
	valRun := a.spawner.Launch(valInput, moduleRoot)
	return valRun.Await(ctx)
}

func NewValidationServerAPI(spawner validator.ValidationSpawner) *ValidationServerAPI {
	return &ValidationServerAPI{spawner}
}

type ExecServerAPI struct {
	ValidationServerAPI
	execSpawner validator.ExecutionSpawner

	runIdLock sync.Mutex
	nextId    uint64
	runs      map[uint64]validator.ExecutionRun // TODO: expire when old
}

func NewExecutionServerAPI(valSpawner validator.ValidationSpawner, execution validator.ExecutionSpawner) *ExecServerAPI {
	rand.Seed(time.Now().UnixNano())
	return &ExecServerAPI{
		ValidationServerAPI: *NewValidationServerAPI(valSpawner),
		execSpawner:         execution,
		nextId:              rand.Uint64(), // good-enough to aver reusing ids after reboot
		runs:                make(map[uint64]validator.ExecutionRun),
	}

}

func (a *ExecServerAPI) CreateExecutionRun(wasmModuleRoot common.Hash, jsonInput *ValidationInputJson) (uint64, error) {
	input, err := ValidationInputFromJson(jsonInput)
	if err != nil {
		return 0, err
	}
	execRun, err := a.execSpawner.CreateExecutionRun(wasmModuleRoot, input)
	if err != nil {
		return 0, err
	}
	a.runIdLock.Lock()
	defer a.runIdLock.Unlock()
	newId := a.nextId
	a.nextId++
	a.runs[newId] = execRun
	return newId, nil
}

func (a *ExecServerAPI) LatestWasmModuleRoot() (common.Hash, error) {
	return a.execSpawner.LatestWasmModuleRoot()
}

func (a *ExecServerAPI) WriteToFile(jsonInput *ValidationInputJson, expOut validator.GoGlobalState, moduleRoot common.Hash) error {
	input, err := ValidationInputFromJson(jsonInput)
	if err != nil {
		return err
	}
	return a.execSpawner.WriteToFile(input, expOut, moduleRoot)
}

var errRunNotFound error = errors.New("run not found")

func (a *ExecServerAPI) getRun(id uint64) (validator.ExecutionRun, error) {
	a.runIdLock.Lock()
	defer a.runIdLock.Unlock()
	run, found := a.runs[id]
	if !found {
		return nil, errRunNotFound
	}
	return run, nil
}

func (a *ExecServerAPI) GetStepAt(ctx context.Context, execid uint64, position uint64) (*MachineStepResultJson, error) {
	run, err := a.getRun(execid)
	if err != nil {
		return nil, err
	}
	step := run.GetStepAt(position)
	res, err := step.Await(ctx)
	if err != nil {
		return nil, err
	}
	return MachineStepResultToJson(&res), nil
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
	run.PrepareRange(start, end)
	return nil
}

func (a *ExecServerAPI) CloseExec(execid uint64) {
	a.runIdLock.Lock()
	defer a.runIdLock.Unlock()
	run, found := a.runs[execid]
	if !found {
		return
	}
	run.Close()
	delete(a.runs, execid)
}
