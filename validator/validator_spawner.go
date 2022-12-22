package validator

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

type ValidationSpawner interface {
	Launch(entry *ValidationInput, moduleRoot common.Hash) ValidationRun
	Stop()
	Name() string
	Room() int
}

type ValidationRun interface {
	WasmModuleRoot() common.Hash
	Done() bool
	ChDone() chan struct{}
	Result() (GoGlobalState, error)
	Close()
}

type ArbitratorSpawner struct {
	count         int32
	ctx           context.Context
	cancel        func()
	locator       *MachineLocator
	machineLoader *ArbMachineLoader
}

type valRun struct {
	err      error
	root     common.Hash
	chanDone chan struct{}
	boolDone int32
	result   GoGlobalState
}

var ErrNotDone error = errors.New("not done")

func (r *valRun) Done() bool {
	return atomic.LoadInt32(&r.boolDone) != 0
}

func (r *valRun) ChDone() chan struct{} {
	return r.chanDone
}

func (r *valRun) Result() (GoGlobalState, error) {
	if !r.Done() {
		return GoGlobalState{}, ErrNotDone
	}
	return r.result, r.err
}

func (r *valRun) WasmModuleRoot() common.Hash {
	return r.root
}

func (r *valRun) Close() {}

func NewvalRun(root common.Hash) *valRun {
	return &valRun{
		root:     root,
		boolDone: 0,
		chanDone: make(chan struct{}),
	}
}

func (r *valRun) consumeResult(res GoGlobalState, err error) {
	r.result = res
	r.err = err
	atomic.StoreInt32(&r.boolDone, 1)
	close(r.chanDone)
}

func NewArbitratorSpawner(locator *MachineLocator) (*ArbitratorSpawner, error) {
	// TODO: preload machines
	ctx, cancel := context.WithCancel(context.Background())
	return &ArbitratorSpawner{
		ctx:           ctx,
		cancel:        cancel,
		locator:       locator,
		machineLoader: NewArbMachineLoader(&DefaultArbitratorMachineConfig, locator),
	}, nil
}

func (s *ArbitratorSpawner) LatestWasmModuleRoot() (common.Hash, error) {
	return s.locator.LatestWasmModuleRoot(), nil
}

func (s *ArbitratorSpawner) Name() string {
	return "arbitrator"
}

func (v *ArbitratorSpawner) loadEntryToMachine(ctx context.Context, entry *ValidationInput, mach *ArbitratorMachine) error {
	resolver := func(hash common.Hash) ([]byte, error) {
		// Check if it's a known preimage
		if preimage, ok := entry.Preimages[hash]; ok {
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
	ctx context.Context, entry *ValidationInput, moduleRoot common.Hash,
) (GoGlobalState, error) {
	basemachine, err := v.machineLoader.GetHostIoMachine(ctx, moduleRoot)
	if err != nil {
		return GoGlobalState{}, fmt.Errorf("unabled to get WASM machine: %w", err)
	}

	mach := basemachine.Clone()
	err = v.loadEntryToMachine(ctx, entry, mach)
	if err != nil {
		return GoGlobalState{}, err
	}
	var steps uint64
	for mach.IsRunning() {
		var count uint64 = 500000000
		err = mach.Step(ctx, count)
		if steps > 0 {
			log.Debug("validation", "moduleRoot", moduleRoot, "block", entry.Id, "steps", steps)
		}
		if err != nil {
			return GoGlobalState{}, fmt.Errorf("machine execution failed with error: %w", err)
		}
		steps += count
	}
	if mach.IsErrored() {
		log.Error("machine entered errored state during attempted validation", "block", entry.Id)
		return GoGlobalState{}, errors.New("machine entered errored state during attempted validation")
	}
	return mach.GetGlobalState(), nil
}

func (v *ArbitratorSpawner) Launch(entry *ValidationInput, moduleRoot common.Hash) ValidationRun {
	atomic.AddInt32(&v.count, 1)
	run := NewvalRun(moduleRoot)
	go func() {
		run.consumeResult(v.execute(v.ctx, entry, moduleRoot))
		atomic.AddInt32(&v.count, -1)
	}()
	return run
}

func (v *ArbitratorSpawner) Room() int {
	return runtime.NumCPU() - int(atomic.LoadInt32(&v.count))
}

var launchTime = time.Now().Format("2006_01_02__15_04")

//nolint:gosec
func (v *ArbitratorSpawner) WriteToFile(outPath string, input *ValidationInput, expOut GoGlobalState, moduleRoot common.Hash) error {
	outDirPath := filepath.Join(v.locator.rootPath, outPath, launchTime, fmt.Sprintf("block_%d", input.Id))
	err := os.MkdirAll(outDirPath, 0755)
	if err != nil {
		return err
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
		"MACHPATH=\"" + v.locator.getMachinePath(moduleRoot) + "\"\n" +
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
	for _, data := range input.Preimages {
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

	_, err = cmdFile.WriteString(" --preimages preimages.bin")
	if err != nil {
		return err
	}

	if input.HasDelayedMsg {
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

func (v *ArbitratorSpawner) CreateExecutionBackend(ctx context.Context, wasmModuleRoot common.Hash, input *ValidationInput, targetMachineNum int) (*ExecutionChallengeBackend, error) {
	initialFrozenMachine, err := v.machineLoader.GetZeroStepMachine(ctx, wasmModuleRoot)
	if err != nil {
		return nil, err
	}
	machine := initialFrozenMachine.Clone()
	err = v.loadEntryToMachine(ctx, input, machine)
	if err != nil {
		return nil, err
	}
	machine.Freeze()
	return NewExecutionChallengeBackend(machine, targetMachineNum, nil)
}

func (v *ArbitratorSpawner) Stop() {
	v.cancel()
}

type JitSpawner struct {
	ctx           context.Context
	cancel        func()
	count         int32
	locator       *MachineLocator
	machineLoader *JitMachineLoader
	config        *JitMachineConfig
}

func NewJitSpawner(locator *MachineLocator, fatalErrChan chan error) (*JitSpawner, error) {
	// TODO - preload machines
	config := &DefaultJitMachineConfig
	loader, err := NewJitMachineLoader(config, locator, fatalErrChan)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &JitSpawner{
		ctx:           ctx,
		cancel:        cancel,
		locator:       locator,
		machineLoader: loader,
		config:        config,
	}, nil
}

func (v *JitSpawner) execute(
	ctx context.Context, entry *ValidationInput, moduleRoot common.Hash,
) (GoGlobalState, error) {
	empty := GoGlobalState{}

	machine, err := v.machineLoader.GetMachine(ctx, moduleRoot)
	if err != nil {
		return empty, fmt.Errorf("unabled to get WASM machine: %w", err)
	}

	resolver := func(hash common.Hash) ([]byte, error) {
		// Check if it's a known preimage
		if preimage, ok := entry.Preimages[hash]; ok {
			return preimage, nil
		}
		return nil, errors.New("preimage not found")
	}
	state, err := machine.prove(ctx, entry, resolver)
	return state, err
}

func (s *JitSpawner) Name() string {
	if s.config.JitCranelift {
		return "jit-cranelift"
	}
	return "jit"
}

func (v *JitSpawner) Launch(entry *ValidationInput, moduleRoot common.Hash) ValidationRun {
	atomic.AddInt32(&v.count, 1)
	run := NewvalRun(moduleRoot)
	go func() {
		run.consumeResult(v.execute(v.ctx, entry, moduleRoot))
		atomic.AddInt32(&v.count, -1)
	}()
	return run
}

func (v *JitSpawner) Room() int {
	return runtime.NumCPU() - int(atomic.LoadInt32(&v.count))
}

func (v *JitSpawner) Stop() {
	v.cancel()
	v.machineLoader.Stop()
}
