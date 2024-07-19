package server_arb

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode/redis"

	"github.com/ethereum/go-ethereum/common"
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

func (s *ArbitratorSpawner) StylusArch() string {
	return "wavm"
}

func (s *ArbitratorSpawner) Name() string {
	return "arbitrator"
}

func (v *ArbitratorSpawner) loadEntryToMachine(ctx context.Context, entry *validator.ValidationInput, mach *ArbitratorMachine) error {
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
	if entry.StylusArch != "wavm" {
		return fmt.Errorf("bad stylus arch loaded to machine. Expected wavm. Got: %s", entry.StylusArch)
	}
	for moduleHash, module := range entry.UserWasms {
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

var launchTime = time.Now().Format("2006_01_02__15_04")

//nolint:gosec
func (v *ArbitratorSpawner) writeToFile(ctx context.Context, input *validator.ValidationInput, expOut validator.GoGlobalState, moduleRoot common.Hash) error {
	outDirPath := filepath.Join(v.locator.RootPath(), v.config().OutputPath, launchTime, fmt.Sprintf("block_%d", input.Id))
	err := os.MkdirAll(outDirPath, 0755)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	rootPathAssign := ""
	if executable, err := os.Executable(); err == nil {
		rootPathAssign = "ROOTPATH=\"" + filepath.Dir(executable) + "\"\n"
	}
	cmdFile, err := os.OpenFile(filepath.Join(outDirPath, "run-prover.sh"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer cmdFile.Close()
	_, err = cmdFile.WriteString("#!/bin/bash\n" +
		fmt.Sprintf("# expected output: batch %d, postion %d, hash %s\n", expOut.Batch, expOut.PosInBatch, expOut.BlockHash) +
		"MACHPATH=\"" + v.locator.GetMachinePath(moduleRoot) + "\"\n" +
		rootPathAssign +
		"if (( $# > 1 )); then\n" +
		"	if [[ $1 == \"-m\" ]]; then\n" +
		"		MACHPATH=$2\n" +
		"		shift\n" +
		"		shift\n" +
		"	fi\n" +
		"fi\n" +
		"${ROOTPATH}/bin/prover ${MACHPATH}/replay.wasm")
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	libraries := []string{"soft-float.wasm", "wasi_stub.wasm", "go_stub.wasm", "host_io.wasm", "brotli.wasm"}
	for _, module := range libraries {
		_, err = cmdFile.WriteString(" -l " + "${MACHPATH}/" + module)
		if err != nil {
			return err
		}
	}
	_, err = cmdFile.WriteString(fmt.Sprintf(" --inbox-position %d --position-within-message %d --last-block-hash %s", input.StartState.Batch, input.StartState.PosInBatch, input.StartState.BlockHash))
	if err != nil {
		return err
	}

	for _, msg := range input.BatchInfo {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		sequencerFileName := fmt.Sprintf("sequencer_%d.bin", msg.Number)
		err = os.WriteFile(filepath.Join(outDirPath, sequencerFileName), msg.Data, 0644)
		if err != nil {
			return err
		}
		_, err = cmdFile.WriteString(" --inbox " + sequencerFileName)
		if err != nil {
			return err
		}
	}

	preimageFile, err := os.Create(filepath.Join(outDirPath, "preimages.bin"))
	if err != nil {
		return err
	}
	defer preimageFile.Close()
	for ty, preimages := range input.Preimages {
		_, err = preimageFile.Write([]byte{byte(ty)})
		if err != nil {
			return err
		}
		for _, data := range preimages {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lenbytes := make([]byte, 8)
			binary.LittleEndian.PutUint64(lenbytes, uint64(len(data)))
			_, err := preimageFile.Write(lenbytes)
			if err != nil {
				return err
			}
			_, err = preimageFile.Write(data)
			if err != nil {
				return err
			}
		}
	}

	_, err = cmdFile.WriteString(" --preimages preimages.bin")
	if err != nil {
		return err
	}

	if input.HasDelayedMsg {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_, err = cmdFile.WriteString(fmt.Sprintf(" --delayed-inbox-position %d", input.DelayedMsgNr))
		if err != nil {
			return err
		}
		filename := fmt.Sprintf("delayed_%d.bin", input.DelayedMsgNr)
		err = os.WriteFile(filepath.Join(outDirPath, filename), input.DelayedMsg, 0644)
		if err != nil {
			return err
		}
		_, err = cmdFile.WriteString(fmt.Sprintf(" --delayed-inbox %s", filename))
		if err != nil {
			return err
		}
	}

	_, err = cmdFile.WriteString(" \"$@\"\n")
	if err != nil {
		return err
	}
	return nil
}

func (v *ArbitratorSpawner) WriteToFile(input *validator.ValidationInput, expOut validator.GoGlobalState, moduleRoot common.Hash) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](v, func(ctx context.Context) (struct{}, error) {
		err := v.writeToFile(ctx, input, expOut, moduleRoot)
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
