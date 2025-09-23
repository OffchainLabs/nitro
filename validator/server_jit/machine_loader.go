package server_jit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/validator/server_common"
)

type JitMachineConfig struct {
	ProverBinPath        string
	JitCranelift         bool
	WasmMemoryUsageLimit int
	JitPath              string
}

var DefaultJitMachineConfig = JitMachineConfig{
	JitCranelift:         true,
	ProverBinPath:        "replay.wasm",
	WasmMemoryUsageLimit: 4294967296,
	JitPath:              "", // Empty string means use default path resolution
}

func getJitPath(configPath string) (string, error) {
	// If a custom path is provided, use it directly
	if configPath != "" {
		_, err := os.Stat(configPath)
		return configPath, err
	}

	// Fall back to original logic for auto-detection
	var jitBinary string
	executable, err := os.Executable()
	if err == nil {
		if strings.Contains(filepath.Base(executable), "test") || strings.Contains(filepath.Dir(executable), "system_tests") {
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

func NewJitMachineLoader(config *JitMachineConfig, locator *server_common.MachineLocator, maxExecutionTime time.Duration, fatalErrChan chan error) (*JitMachineLoader, error) {
	jitPath, err := getJitPath(config.JitPath)
	if err != nil {
		return nil, err
	}
	createMachineThreadFunc := func(ctx context.Context, moduleRoot common.Hash) (*JitMachine, error) {
		binPath := filepath.Join(locator.GetMachinePath(moduleRoot), config.ProverBinPath)
		return createJitMachine(jitPath, binPath, config.JitCranelift, config.WasmMemoryUsageLimit, maxExecutionTime, moduleRoot, fatalErrChan)
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
