package server_jit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/validator/server_common"
)

type JitMachineConfig struct {
	ProverBinPath string
	JitCranelift  bool
}

var DefaultJitMachineConfig = JitMachineConfig{
	JitCranelift:  true,
	ProverBinPath: "replay.wasm",
}

func getJitPath() (string, error) {
	var jitBinary string
	executable, err := os.Executable()
	if err == nil {
		if strings.Contains(filepath.Base(executable), "test") {
			_, thisfile, _, _ := runtime.Caller(0)
			projectDir := filepath.Dir(filepath.Dir(filepath.Dir(thisfile)))
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

type JitMachineLoader struct {
	server_common.MachineLoader[JitMachine]
	stopped bool
}

func NewJitMachineLoader(config *JitMachineConfig, locator *server_common.MachineLocator, fatalErrChan chan error) (*JitMachineLoader, error) {
	jitPath, err := getJitPath()
	if err != nil {
		return nil, err
	}
	createMachineThreadFunc := func(ctx context.Context, moduleRoot common.Hash) (*JitMachine, error) {
		binPath := filepath.Join(locator.GetMachinePath(moduleRoot), config.ProverBinPath)
		return createJitMachine(jitPath, binPath, config.JitCranelift, moduleRoot, fatalErrChan)
	}
	return &JitMachineLoader{
		MachineLoader: *server_common.NewMachineLoader[JitMachine](locator, createMachineThreadFunc),
	}, nil
}

func (j *JitMachineLoader) Stop() {
	if j.stopped {
		return
	}
	j.ForEachReadyMachine(func(machine *JitMachine) { machine.close() })
	j.stopped = true
}
