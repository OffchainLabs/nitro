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

type jitMachineStatus struct {
	machine   *JitMachine
	readyChan chan struct{}
	err       error
}

func newJitMachineStatus() *jitMachineStatus {
	return &jitMachineStatus{
		readyChan: make(chan struct{}),
	}
}

type JitMachineLoader struct {
	mapMutex     sync.Mutex
	machines     map[common.Hash]*jitMachineStatus
	config       *JitMachineConfig
	locator      *MachineLocator
	fatalErrChan chan error
	stopped      bool
	jitPath      string
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

func NewJitMachineLoader(config *JitMachineConfig, locator *MachineLocator, fatalErrChan chan error) (*JitMachineLoader, error) {
	jitPath, err := getJitPath()
	if err != nil {
		return nil, err
	}
	return &JitMachineLoader{
		machines:     make(map[common.Hash]*jitMachineStatus),
		config:       config,
		locator:      locator,
		fatalErrChan: fatalErrChan,
		jitPath:      jitPath,
	}, nil
}

func (j *JitMachineLoader) createMachineThread(ctx context.Context, moduleRoot common.Hash, status *jitMachineStatus) {
	defer close(status.readyChan)
	binPath := filepath.Join(j.locator.getMachinePath(moduleRoot), j.config.ProverBinPath)
	status.machine, status.err = createJitMachine(j.jitPath, binPath, j.config, moduleRoot, j.fatalErrChan)
}

func (j *JitMachineLoader) getMachineStatus(ctx context.Context, moduleRoot common.Hash) (*jitMachineStatus, error) {
	if moduleRoot == (common.Hash{}) {
		moduleRoot = j.locator.LatestWasmModuleRoot()
		if (moduleRoot == common.Hash{}) {
			return nil, ErrMachineNotFound
		}
	}
	j.mapMutex.Lock()
	status := j.machines[moduleRoot]
	if status == nil {
		status = newJitMachineStatus()
		j.machines[moduleRoot] = status
		go j.createMachineThread(context.Background(), moduleRoot, status)
	}
	j.mapMutex.Unlock()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-status.readyChan:
	}
	return status, status.err
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
