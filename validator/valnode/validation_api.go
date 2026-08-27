// Copyright 2023-2026, Offchain Labs, Inc.
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

// implements subset of methods required by ValidationClient
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

// ExecServer manages execution runs.
type ExecServer struct {
	stopwaiter.StopWaiter
	execSpawner validator.ExecutionSpawner

	config server_arb.ArbitratorSpawnerConfigFetcher

	runIdLock sync.Mutex
	nextId    uint64
	runs      map[uint64]*execRunEntry
}

func NewExecServer(execution validator.ExecutionSpawner, config server_arb.ArbitratorSpawnerConfigFetcher) *ExecServer {
	return &ExecServer{
		execSpawner: execution,
		nextId:      rand.Uint64(), // good-enough to avoid reusing ids after reboot
		runs:        make(map[uint64]*execRunEntry),
		config:      config,
	}
}

func (s *ExecServer) CreateExecutionRun(ctx context.Context, wasmModuleRoot common.Hash, jsonInput *server_api.InputJSON, useBoldMachineOptional *bool) (uint64, error) {
	if s.Stopped() {
		return 0, errors.New("ExecServer is stopped")
	}
	input, err := server_api.ValidationInputFromJson(jsonInput)
	if err != nil {
		return 0, err
	}
	useBoldMachine := false
	if useBoldMachineOptional != nil {
		useBoldMachine = *useBoldMachineOptional
	}
	execRun, err := s.execSpawner.CreateExecutionRun(wasmModuleRoot, input, useBoldMachine).Await(ctx)
	if err != nil {
		return 0, err
	}
	s.runIdLock.Lock()
	defer s.runIdLock.Unlock()
	newId := s.nextId
	s.nextId++
	s.runs[newId] = &execRunEntry{execRun, time.Now()}
	return newId, nil
}

func (s *ExecServer) removeOldRuns(ctx context.Context) time.Duration {
	oldestKept := time.Now().Add(-1 * s.config().ExecutionRunTimeout)
	s.runIdLock.Lock()
	defer s.runIdLock.Unlock()
	for id, entry := range s.runs {
		if entry.accessed.Before(oldestKept) {
			entry.run.Close()
			delete(s.runs, id)
		}
	}
	return s.config().ExecutionRunTimeout / 5
}

func (s *ExecServer) Start(ctx_in context.Context) {
	s.StopWaiter.Start(ctx_in, s)
	s.CallIteratively(s.removeOldRuns)
}

func (s *ExecServer) StopAndWait() {
	s.StopWaiter.StopAndWait()
	s.runIdLock.Lock()
	defer s.runIdLock.Unlock()
	for _, entry := range s.runs {
		entry.run.Close()
	}
}

var errRunNotFound error = errors.New("run not found")

func (s *ExecServer) getRun(id uint64) (validator.ExecutionRun, error) {
	if s.Stopped() {
		return nil, errRunNotFound
	}
	s.runIdLock.Lock()
	defer s.runIdLock.Unlock()
	entry := s.runs[id]
	if entry == nil {
		return nil, errRunNotFound
	}
	entry.accessed = time.Now()
	return entry.run, nil
}

func (s *ExecServer) GetStepAt(ctx context.Context, execid uint64, position uint64) (*server_api.MachineStepResultJson, error) {
	run, err := s.getRun(execid)
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

func (s *ExecServer) GetMachineHashesWithStepSize(ctx context.Context, execid, fromStep, stepSize, maxIterations uint64) ([]common.Hash, error) {
	run, err := s.getRun(execid)
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

func (s *ExecServer) GetProofAt(ctx context.Context, execid uint64, position uint64) (string, error) {
	run, err := s.getRun(execid)
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

func (s *ExecServer) PrepareRange(ctx context.Context, execid uint64, start, end uint64) error {
	run, err := s.getRun(execid)
	if err != nil {
		return err
	}
	_, err = run.PrepareRange(start, end).Await(ctx)
	return err
}

func (s *ExecServer) ExecKeepAlive(ctx context.Context, execid uint64) error {
	_, err := s.getRun(execid)
	if err != nil {
		return err
	}
	return nil
}

func (s *ExecServer) CheckAlive(ctx context.Context, execid uint64) error {
	run, err := s.getRun(execid)
	if err != nil {
		return err
	}
	return run.CheckAlive(ctx)
}

func (s *ExecServer) CloseExec(execid uint64) {
	// Protect map access with runIdLock to avoid concurrent map read/write.
	// Call Close() outside the lock to avoid holding the mutex during a potentially long operation.
	s.runIdLock.Lock()
	entry := s.runs[execid]
	if entry != nil {
		delete(s.runs, execid)
	}
	s.runIdLock.Unlock()

	if entry != nil {
		entry.run.Close()
	}
}

type ExecServerAPI struct {
	valServerAPI *ValidationServerAPI
	execServer   *ExecServer
}

func NewExecServerAPI(valSpawner validator.ValidationSpawner, execServer *ExecServer) *ExecServerAPI {
	return &ExecServerAPI{
		valServerAPI: NewValidationServerAPI(valSpawner),
		execServer:   execServer,
	}
}

func (a *ExecServerAPI) Name() string {
	return a.valServerAPI.Name()
}

func (a *ExecServerAPI) Capacity() int {
	return a.valServerAPI.Capacity()
}

func (a *ExecServerAPI) Room() int {
	return a.valServerAPI.Room()
}

func (a *ExecServerAPI) Validate(ctx context.Context, entry *server_api.InputJSON, moduleRoot common.Hash) (validator.GoGlobalState, error) {
	return a.valServerAPI.Validate(ctx, entry, moduleRoot)
}

func (a *ExecServerAPI) WasmModuleRoots() ([]common.Hash, error) {
	return a.valServerAPI.WasmModuleRoots()
}

func (a *ExecServerAPI) StylusArchs() ([]rawdb.WasmTarget, error) {
	return a.valServerAPI.StylusArchs()
}

func (a *ExecServerAPI) CreateExecutionRun(ctx context.Context, wasmModuleRoot common.Hash, jsonInput *server_api.InputJSON, useBoldMachineOptional *bool) (uint64, error) {
	return a.execServer.CreateExecutionRun(ctx, wasmModuleRoot, jsonInput, useBoldMachineOptional)
}

func (a *ExecServerAPI) GetStepAt(ctx context.Context, execid uint64, position uint64) (*server_api.MachineStepResultJson, error) {
	return a.execServer.GetStepAt(ctx, execid, position)
}

func (a *ExecServerAPI) GetMachineHashesWithStepSize(ctx context.Context, execid, fromStep, stepSize, maxIterations uint64) ([]common.Hash, error) {
	return a.execServer.GetMachineHashesWithStepSize(ctx, execid, fromStep, stepSize, maxIterations)
}

func (a *ExecServerAPI) GetProofAt(ctx context.Context, execid uint64, position uint64) (string, error) {
	return a.execServer.GetProofAt(ctx, execid, position)
}

func (a *ExecServerAPI) PrepareRange(ctx context.Context, execid uint64, start, end uint64) error {
	return a.execServer.PrepareRange(ctx, execid, start, end)
}

func (a *ExecServerAPI) ExecKeepAlive(ctx context.Context, execid uint64) error {
	return a.execServer.ExecKeepAlive(ctx, execid)
}

func (a *ExecServerAPI) CheckAlive(ctx context.Context, execid uint64) error {
	return a.execServer.CheckAlive(ctx, execid)
}

func (a *ExecServerAPI) CloseExec(execid uint64) {
	a.execServer.CloseExec(execid)
}
