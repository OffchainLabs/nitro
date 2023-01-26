package validator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type JitMachineConfig struct {
	ProverBinPath string
	JitCranelift  bool
}

var DefaultJitMachineConfig = JitMachineConfig{
	JitCranelift:  true,
	ProverBinPath: "replay.wasm",
}

type machineStatus[M any] struct {
	machine   *M
	readyChan chan struct{}
	err       error
}

func newMachineStatus[M any]() *machineStatus[M] {
	return &machineStatus[M]{
		readyChan: make(chan struct{}),
	}
}

type machineLoader[M any] struct {
	mapMutex            sync.Mutex
	machines            map[common.Hash]*machineStatus[M]
	locator             *MachineLocator
	createMachineThread func(ctx context.Context, moduleRoot common.Hash, status *machineStatus[M])
	stopped             bool
}

func getJitPath() (string, error) {
	var jitBinary string
	executable, err := os.Executable()
	if err == nil {
		if strings.Contains(filepath.Base(executable), "test") {
			_, thisfile, _, _ := runtime.Caller(0)
			projectDir := filepath.Dir(filepath.Dir(thisfile))
			jitBinary = filepath.Join(projectDir, "target", "bin", "jit")
		} else {
			jitBinary = filepath.Join(filepath.Dir(executable), "jit")
		}
		_, err = os.Stat(jitBinary)
	}
	if err != nil {
		var lookPathErr error
		jitBinary, lookPathErr = exec.LookPath("jit")
		if lookPathErr == nil {
			return jitBinary, nil
		}
	}
	return jitBinary, err
}

func newMachineLoader[M any](
	locator *MachineLocator,
	createMachineThread func(ctx context.Context, moduleRoot common.Hash, status *machineStatus[M]),
) *machineLoader[M] {

	return &machineLoader[M]{
		machines:            make(map[common.Hash]*machineStatus[M]),
		locator:             locator,
		createMachineThread: createMachineThread,
	}
}

func (l *machineLoader[M]) getMachineStatus(ctx context.Context, moduleRoot common.Hash) (*machineStatus[M], error) {
	if moduleRoot == (common.Hash{}) {
		moduleRoot = l.locator.LatestWasmModuleRoot()
		if (moduleRoot == common.Hash{}) {
			return nil, ErrMachineNotFound
		}
	}
	l.mapMutex.Lock()
	status := l.machines[moduleRoot]
	if status == nil {
		status = newMachineStatus[M]()
		l.machines[moduleRoot] = status
		go func() {
			l.createMachineThread(context.Background(), moduleRoot, status)
			close(status.readyChan)
		}()
	}
	l.mapMutex.Unlock()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-status.readyChan:
	}
	return status, status.err
}

type JitMachineLoader struct {
	machineLoader[JitMachine]
}

func NewJitMachineLoader(config *JitMachineConfig, locator *MachineLocator, fatalErrChan chan error) (*JitMachineLoader, error) {
	jitPath, err := getJitPath()
	if err != nil {
		return nil, err
	}
	createMachineThreadFunc := func(ctx context.Context, moduleRoot common.Hash, status *machineStatus[JitMachine]) {
		binPath := filepath.Join(locator.getMachinePath(moduleRoot), config.ProverBinPath)
		status.machine, status.err = createJitMachine(jitPath, binPath, config, moduleRoot, fatalErrChan)
	}
	return &JitMachineLoader{
		machineLoader: *newMachineLoader[JitMachine](locator, createMachineThreadFunc),
	}, nil
}

func (a *JitMachineLoader) GetMachine(ctx context.Context, moduleRoot common.Hash) (*JitMachine, error) {
	status, err := a.getMachineStatus(ctx, moduleRoot)
	if err != nil {
		return nil, err
	}
	return status.machine, status.err
}

func (j *JitMachineLoader) Stop() {
	if j.stopped {
		return
	}
	for _, stat := range j.machines {
		if stat.machine != nil {
			stat.machine.close()
		}
	}
	j.stopped = true
}

type ArbitratorMachineConfig struct {
	WavmBinaryPath       string
	UntilHostIoStatePath string
}

var DefaultArbitratorMachineConfig = ArbitratorMachineConfig{
	WavmBinaryPath:       "machine.wavm.br",
	UntilHostIoStatePath: "until-host-io-state.bin",
}

type arbMachines struct {
	zeroStep *ArbitratorMachine
	hostIo   *ArbitratorMachine
}

type ArbMachineLoader struct {
	machineLoader[arbMachines]
}

func NewArbMachineLoader(config *ArbitratorMachineConfig, locator *MachineLocator) *ArbMachineLoader {
	createMachineThreadFunc := func(ctx context.Context, moduleRoot common.Hash, status *machineStatus[arbMachines]) {
		createArbMachine(ctx, locator, config, moduleRoot, status)
	}
	return &ArbMachineLoader{
		machineLoader: *newMachineLoader[arbMachines](locator, createMachineThreadFunc),
	}
}

func (a *ArbMachineLoader) GetHostIoMachine(ctx context.Context, moduleRoot common.Hash) (*ArbitratorMachine, error) {
	status, err := a.getMachineStatus(ctx, moduleRoot)
	if err != nil {
		return nil, err
	}
	return status.machine.hostIo, status.err
}

func (a *ArbMachineLoader) GetZeroStepMachine(ctx context.Context, moduleRoot common.Hash) (*ArbitratorMachine, error) {
	status, err := a.getMachineStatus(ctx, moduleRoot)
	if err != nil {
		return nil, err
	}
	return status.machine.zeroStep, status.err
}
