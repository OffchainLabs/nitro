package server_api

import (
	"context"
	"encoding/base64"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_api/validation"
	"github.com/offchainlabs/nitro/validator/server_arb"
	"github.com/spf13/pflag"
)

const Namespace string = "validation"

type ValidationServerAPI struct {
	stopwaiter.StopWaiter
	spawner           validator.ValidationSpawner
	consumerThreadCnt int
	consumer          *pubsub.Consumer[validation.Request, any]
}

func (a *ValidationServerAPI) Start(ctx_in context.Context) {
	a.StopWaiter.Start(ctx_in, a)
	if a.consumer != nil {
		a.consumer.Start(ctx_in)
		for i := 0; i < a.consumerThreadCnt; i++ {
			a.StopWaiter.CallIteratively(func(ctx context.Context) time.Duration {
				req, err := a.consumer.Consume(ctx)
				if err != nil {
					log.Error("Consuming request", "error", err)
					return 0
				}
				res, err := a.Validate(a.GetContext(), req.Value.Input, req.Value.ModuleRoot)
				if err != nil {
					log.Error("Error validating", "input", req.Value.Input, "moduleRoot", req.Value.ModuleRoot, "error", err)
					return 0
				}
				if err := a.consumer.SetResult(a.GetContext(), req.ID, res); err != nil {
					log.Error("Error setting result for request", "id", req.ID, "result", res, "error", err)
					return 0
				}
				return time.Second
			})
		}
	}
}

func (a *ValidationServerAPI) StopAndWait() {
	a.StopWaiter.StopAndWait()
}

func (a *ValidationServerAPI) Name() string {
	return a.spawner.Name()
}

func (a *ValidationServerAPI) Room() int {
	return a.spawner.Room()
}

func (a *ValidationServerAPI) Validate(ctx context.Context, entry *validation.InputJSON, moduleRoot common.Hash) (validator.GoGlobalState, error) {
	valInput, err := ValidationInputFromJson(entry)
	if err != nil {
		return validator.GoGlobalState{}, err
	}
	valRun := a.spawner.Launch(valInput, moduleRoot)
	return valRun.Await(ctx)
}

func NewValidationServerAPI(spawner validator.ValidationSpawner) *ValidationServerAPI {
	return &ValidationServerAPI{spawner: spawner}
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

type ExecutionServerAPIConfig struct {
	Arbitrator  server_arb.ArbitratorSpawnerConfig `koanf:"arbitrator"`
	ConsumerCfg pubsub.ConsumerConfig              `koanf:"consumer"`
	// Number of consumer threads concurrently consuming requests through single
	// consumer.
	ConsumerThreadCount uint64 `koanf:"consumer-thread-count"`
}

func ExecutionServerAPIConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Uint64(prefix+".consumer-thread-count", DefaultExecutionServerAPIConfig.ConsumerThreadCount, "number of consumer threads")
	server_arb.ArbitratorSpawnerConfigAddOptions(prefix+".arbitrator", f)
	pubsub.ConsumerConfigAddOptions(prefix+".consumer", f)
}

var DefaultExecutionServerAPIConfig = ExecutionServerAPIConfig{
	Arbitrator:          server_arb.DefaultArbitratorSpawnerConfig,
	ConsumerCfg:         *pubsub.DefaultConsumerConfig,
	ConsumerThreadCount: 1,
}

var TestExecutionServerAPIConfig = ExecutionServerAPIConfig{
	Arbitrator:          server_arb.DefaultArbitratorSpawnerConfig,
	ConsumerCfg:         *pubsub.TestConsumerConfig,
	ConsumerThreadCount: 1,
}

type ExecutionServerAPIConfigFetcher func() *ExecutionServerAPIConfig

func NewExecutionServerAPI(valSpawner validator.ValidationSpawner, execution validator.ExecutionSpawner, config ExecutionServerAPIConfigFetcher) *ExecServerAPI {
	srvAPICfg := func() *server_arb.ArbitratorSpawnerConfig {
		return &config().Arbitrator
	}

	return &ExecServerAPI{
		ValidationServerAPI: *NewValidationServerAPI(valSpawner),
		execSpawner:         execution,
		nextId:              rand.Uint64(), // good-enough to aver reusing ids after reboot
		runs:                make(map[uint64]*execRunEntry),
		config:              srvAPICfg,
	}
}

func (a *ExecServerAPI) CreateExecutionRun(ctx context.Context, wasmModuleRoot common.Hash, jsonInput *validation.InputJSON) (uint64, error) {
	input, err := ValidationInputFromJson(jsonInput)
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
	a.ValidationServerAPI.Start(ctx_in)
	a.StopWaiter.Start(ctx_in, a)
	a.CallIteratively(a.removeOldRuns)
}

func (a *ExecServerAPI) WriteToFile(ctx context.Context, jsonInput *validation.InputJSON, expOut validator.GoGlobalState, moduleRoot common.Hash) error {
	input, err := ValidationInputFromJson(jsonInput)
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
	return MachineStepResultToJson(res), nil
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
