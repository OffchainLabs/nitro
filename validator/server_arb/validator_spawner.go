package server_arb

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode/redis"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

var arbitratorValidationSteps = metrics.NewRegisteredHistogram("arbitrator/validation/steps", nil, metrics.NewBoundedHistogramSample())

type ArbitratorSpawnerConfig struct {
	Workers                     int                          `koanf:"workers" reload:"hot"`
	OutputPath                  string                       `koanf:"output-path" reload:"hot"`
	Execution                   MachineCacheConfig           `koanf:"execution" reload:"hot"` // hot reloading for new executions only
	ExecutionRunTimeout         time.Duration                `koanf:"execution-run-timeout" reload:"hot"`
	RedisValidationServerConfig redis.ValidationServerConfig `koanf:"redis-validation-server-config"`
}

type ArbitratorSpawnerConfigFecher func() *ArbitratorSpawnerConfig

var DefaultArbitratorSpawnerConfig = ArbitratorSpawnerConfig{
	Workers:                     0,
	OutputPath:                  "./target/output",
	Execution:                   DefaultMachineCacheConfig,
	ExecutionRunTimeout:         time.Minute * 15,
	RedisValidationServerConfig: redis.DefaultValidationServerConfig,
}

func ArbitratorSpawnerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Int(prefix+".workers", DefaultArbitratorSpawnerConfig.Workers, "number of concurrent validation threads")
	f.Duration(prefix+".execution-run-timeout", DefaultArbitratorSpawnerConfig.ExecutionRunTimeout, "timeout before discarding execution run")
	f.String(prefix+".output-path", DefaultArbitratorSpawnerConfig.OutputPath, "path to write machines to")
	MachineCacheConfigConfigAddOptions(prefix+".execution", f)
	redis.ValidationServerConfigAddOptions(prefix+".redis-validation-server-config", f)
}

func DefaultArbitratorSpawnerConfigFetcher() *ArbitratorSpawnerConfig {
	return &DefaultArbitratorSpawnerConfig
}

type ArbitratorSpawner struct {
	stopwaiter.StopWaiter
	count         atomic.Int32
	locator       *server_common.MachineLocator
	machineLoader *ArbMachineLoader
	config        ArbitratorSpawnerConfigFecher
}

func NewArbitratorSpawner(locator *server_common.MachineLocator, config ArbitratorSpawnerConfigFecher) (*ArbitratorSpawner, error) {
	// TODO: preload machines
	spawner := &ArbitratorSpawner{
		locator:       locator,
		machineLoader: NewArbMachineLoader(&DefaultArbitratorMachineConfig, locator),
		config:        config,
	}
	return spawner, nil
}

func (s *ArbitratorSpawner) Start(ctx_in context.Context) error {
	s.StopWaiter.Start(ctx_in, s)
	return nil
}

func (s *ArbitratorSpawner) LatestWasmModuleRoot() containers.PromiseInterface[common.Hash] {
	return containers.NewReadyPromise(s.locator.LatestWasmModuleRoot(), nil)
}

func (s *ArbitratorSpawner) WasmModuleRoots() ([]common.Hash, error) {
	return s.locator.ModuleRoots(), nil
}

func (s *ArbitratorSpawner) StylusArchs() []rawdb.Target {
	return []rawdb.Target{rawdb.TargetWavm}
}

func (s *ArbitratorSpawner) Name() string {
	return "arbitrator"
}

func (v *ArbitratorSpawner) loadEntryToMachine(_ context.Context, entry *validator.ValidationInput, mach *ArbitratorMachine) error {
	resolver := func(ty arbutil.PreimageType, hash common.Hash) ([]byte, error) {
		// Check if it's a known preimage
		if preimage, ok := entry.Preimages[ty][hash]; ok {
			return preimage, nil
		}
		return nil, errors.New("preimage not found")
	}
	if err := mach.SetPreimageResolver(resolver); err != nil {
		return err
	}
	err := mach.SetGlobalState(entry.StartState)
	if err != nil {
		log.Error("error while setting global state for proving", "err", err, "gsStart", entry.StartState)
		return fmt.Errorf("error while setting global state for proving: %w", err)
	}
	for _, batch := range entry.BatchInfo {
		err = mach.AddSequencerInboxMessage(batch.Number, batch.Data)
		if err != nil {
			log.Error(
				"error while trying to add sequencer msg for proving",
				"err", err, "seq", entry.StartState.Batch, "blockNr", entry.Id,
			)
			return fmt.Errorf("error while trying to add sequencer msg for proving: %w", err)
		}
	}
	if len(entry.UserWasms[rawdb.TargetWavm]) == 0 {
		for stylusArch, wasms := range entry.UserWasms {
			if len(wasms) > 0 {
				return fmt.Errorf("bad stylus arch loaded to machine. Expected wavm. Got: %s", stylusArch)
			}
		}
	}
	for moduleHash, module := range entry.UserWasms[rawdb.TargetWavm] {
		err = mach.AddUserWasm(moduleHash, module)
		if err != nil {
			log.Error(
				"error adding user wasm for proving",
				"err", err, "moduleHash", moduleHash, "blockNr", entry.Id,
			)
			return fmt.Errorf("error adding user wasm for proving: %w", err)
		}
	}
	if entry.HasDelayedMsg {
		err = mach.AddDelayedInboxMessage(entry.DelayedMsgNr, entry.DelayedMsg)
		if err != nil {
			log.Error(
				"error while trying to add delayed msg for proving",
				"err", err, "seq", entry.DelayedMsgNr, "blockNr", entry.Id,
			)
			return fmt.Errorf("error while trying to add delayed msg for proving: %w", err)
		}
	}
	return nil
}

func (v *ArbitratorSpawner) execute(
	ctx context.Context, entry *validator.ValidationInput, moduleRoot common.Hash,
) (validator.GoGlobalState, error) {
	basemachine, err := v.machineLoader.GetHostIoMachine(ctx, moduleRoot)
	if err != nil {
		return validator.GoGlobalState{}, fmt.Errorf("unabled to get WASM machine: %w", err)
	}

	mach := basemachine.Clone()
	defer mach.Destroy()
	err = v.loadEntryToMachine(ctx, entry, mach)
	if err != nil {
		return validator.GoGlobalState{}, err
	}
	var steps uint64
	for mach.IsRunning() {
		var count uint64 = 500000000
		err = mach.Step(ctx, count)
		if steps > 0 {
			log.Debug("validation", "moduleRoot", moduleRoot, "block", entry.Id, "steps", steps)
		}
		if err != nil {
			return validator.GoGlobalState{}, fmt.Errorf("machine execution failed with error: %w", err)
		}
		steps += count
	}
	arbitratorValidationSteps.Update(int64(mach.GetStepCount()))
	if mach.IsErrored() {
		log.Error("machine entered errored state during attempted validation", "block", entry.Id)
		return validator.GoGlobalState{}, errors.New("machine entered errored state during attempted validation")
	}
	return mach.GetGlobalState(), nil
}

func (v *ArbitratorSpawner) Launch(entry *validator.ValidationInput, moduleRoot common.Hash) validator.ValidationRun {
	println("LAUCHING ARBITRATOR VALIDATION")
	v.count.Add(1)
	promise := stopwaiter.LaunchPromiseThread[validator.GoGlobalState](v, func(ctx context.Context) (validator.GoGlobalState, error) {
		defer v.count.Add(-1)
		return v.execute(ctx, entry, moduleRoot)
	})
	return server_common.NewValRun(promise, moduleRoot)
}

func (v *ArbitratorSpawner) Room() int {
	avail := v.config().Workers
	if avail == 0 {
		avail = runtime.NumCPU()
	}
	return avail
}

func (v *ArbitratorSpawner) writeToFile(_ context.Context, input *validator.ValidationInput, _ common.Hash) error {
	jsonInput := server_api.ValidationInputToJson(input)
	if err := jsonInput.WriteToFile(); err != nil {
		return err
	}
	return nil
}

func (v *ArbitratorSpawner) WriteToFile(input *validator.ValidationInput, moduleRoot common.Hash) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](v, func(ctx context.Context) (struct{}, error) {
		err := v.writeToFile(ctx, input, moduleRoot)
		return struct{}{}, err
	})
}

func (v *ArbitratorSpawner) CreateExecutionRun(wasmModuleRoot common.Hash, input *validator.ValidationInput) containers.PromiseInterface[validator.ExecutionRun] {
	getMachine := func(ctx context.Context) (MachineInterface, error) {
		initialFrozenMachine, err := v.machineLoader.GetZeroStepMachine(ctx, wasmModuleRoot)
		if err != nil {
			return nil, err
		}
		machine := initialFrozenMachine.Clone()
		err = v.loadEntryToMachine(ctx, input, machine)
		if err != nil {
			machine.Destroy()
			return nil, err
		}
		return machine, nil
	}
	currentExecConfig := v.config().Execution
	return stopwaiter.LaunchPromiseThread[validator.ExecutionRun](v, func(ctx context.Context) (validator.ExecutionRun, error) {
		return NewExecutionRun(v.GetContext(), getMachine, &currentExecConfig)
	})
}

func (v *ArbitratorSpawner) Stop() {
	v.StopOnly()
}
