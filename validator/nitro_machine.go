// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validator

/*
#cgo CFLAGS: -g -Wall -I../target/include/
#include "arbitrator.h"
#include <stdlib.h>
*/
import "C"
import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type NitroMachineConfig struct {
	RootPath                string // a folder with various machines in it
	ProverBinPath           string
	ModulePaths             []string
	InitialMachineCachePath string
}

var DefaultNitroMachineConfig = NitroMachineConfig{
	RootPath:                "./target/machines/",
	ProverBinPath:           "replay.wasm",
	ModulePaths:             []string{"soft-float.wasm", "wasi_stub.wasm", "go_stub.wasm", "host_io.wasm", "brotli.wasm"},
	InitialMachineCachePath: "./target/etc/initial-machine-cache",
}

func init() {
	_, thisfile, _, _ := runtime.Caller(0)
	projectDir := filepath.Dir(filepath.Dir(thisfile))
	DefaultNitroMachineConfig.RootPath = filepath.Join(projectDir, "target", "machines")
}

func (c NitroMachineConfig) getMachinePath(moduleRoot common.Hash) string {
	if moduleRoot == (common.Hash{}) {
		return filepath.Join(c.RootPath, "latest")
	} else {
		return filepath.Join(c.RootPath, moduleRoot.String())
	}
}

func (c NitroMachineConfig) ReadLatestWasmModuleRoot() (common.Hash, error) {
	fileToRead := filepath.Join(c.getMachinePath(common.Hash{}), "module_root")
	fileBytes, err := ioutil.ReadFile(fileToRead)
	if err != nil {
		return common.Hash{}, err
	}
	s := strings.TrimSpace(string(fileBytes))
	if len(s) > 64 {
		s = s[0:64]
	}
	return common.HexToHash(s), nil
}

type loaderMachineStatus struct {
	machine    *ArbitratorMachine
	chanSignal chan struct{}
	err        error
}

func (s *loaderMachineStatus) signalReady() {
	close(s.chanSignal)
}

func (s *loaderMachineStatus) createZeroStepMachineInternal(config NitroMachineConfig, moduleRoot common.Hash, realModuleRoot common.Hash) {
	defer s.signalReady()
	machinePath := config.getMachinePath(moduleRoot)
	moduleList := []string{}
	for _, module := range config.ModulePaths {
		moduleList = append(moduleList, filepath.Join(machinePath, module))
	}
	binPath := filepath.Join(machinePath, config.ProverBinPath)
	cModuleList := CreateCStringList(moduleList)
	defer FreeCStringList(cModuleList, len(moduleList))
	cBinPath := C.CString(binPath)
	defer C.free(unsafe.Pointer(cBinPath))
	log.Info("creating nitro machine", "binpath", binPath, "moduleList", moduleList)
	baseMachine := C.arbitrator_load_machine(cBinPath, cModuleList, C.intptr_t(len(moduleList)))
	if baseMachine == nil {
		s.err = errors.New("failed to create base machine")
		return
	}
	nitroMachine := machineFromPointer(baseMachine)
	machineModuleRoot := nitroMachine.GetModuleRoot()
	if machineModuleRoot != realModuleRoot {
		s.err = fmt.Errorf("attempting to load module root %v got machine with module root %v", realModuleRoot, machineModuleRoot)
		return
	}
	s.machine = nitroMachine
	s.machine.Freeze()
}

// We try to store/load state before first host_io to a file.
// We will chicken out of that if something fails, but still try to calculate the machine
func (s *loaderMachineStatus) createHostIoMachineInternal(config NitroMachineConfig, moduleRoot common.Hash, zerostep *ArbitratorMachine) {
	defer s.signalReady()
	ctx := context.Background()
	machine := zerostep.Clone()
	hash := machine.Hash()
	expectedName := hash.String() + ".bin"
	cacheDir := config.InitialMachineCachePath
	foundInCache := false
	saveStateToFile := true
	err := os.MkdirAll(cacheDir, 0o755)
	if err != nil {
		saveStateToFile = false
	}
	var files []fs.FileInfo
	if saveStateToFile {
		files, err = ioutil.ReadDir(cacheDir)
		if err != nil {
			saveStateToFile = false
		}
	}
	if saveStateToFile {
		cleanCacheBefore := time.Now().Add(-time.Hour * 24)

		for _, file := range files {
			if file.Name() == expectedName {
				foundInCache = true
			} else if file.ModTime().Before(cleanCacheBefore) {
				log.Info("removing unknown old machine cache", "name", file.Name())
				err := os.Remove(filepath.Join(cacheDir, file.Name()))
				if err != nil {
					log.Error("failed removing old machine cache")
					saveStateToFile = false
					break
				}
			} else {
				log.Info("keeping unknown old machine cache", "name", file.Name())
			}
		}
	}

	file := filepath.Join(cacheDir, expectedName)
	if foundInCache {
		// Update the file's last modified time so it doesn't get cleaned up
		now := time.Now()
		err := os.Chtimes(file, now, now)
		if err != nil {
			foundInCache = false
			if !errors.Is(err, os.ErrNotExist) {
				saveStateToFile = false
			}
		}
	}

	if foundInCache {
		log.Info("found cached initial machine", "hash", hash)

		err := machine.DeserializeAndReplaceState(file)
		if err != nil {
			// Safe as if DeserializeAndReplaceState returns an error it will not have mutated the machine
			log.Info("failed to load initial machine cache; will reexecute", "err", err)
		} else {
			s.machine = machine
			s.machine.Freeze()
			return
		}
	} else {
		log.Info("didn't find initial machine in cache", "hash", hash)
	}

	s.err = machine.StepUntilHostIo(ctx)
	if s.err != nil {
		return
	}

	if machine.IsErrored() {
		s.err = errors.New("machine entered errored state while caching execution up to host io")
		return
	}

	s.machine = machine
	s.machine.Freeze()
	if !saveStateToFile {
		return
	}
	log.Info("saving initial machine cache", "hash", hash)

	wipFile := file + ".wip"
	err = machine.SerializeState(wipFile)
	if err != nil {
		log.Error("error trying to save machine state cache", "err", err)
		return
	}
	err = os.Rename(wipFile, file)
	if err != nil {
		log.Error("error trying to rename machine state cache", "err", err)
		return
	}
}

type nitroMachineRequest struct {
	moduleRoot  common.Hash
	untilHostIo bool
}

type NitroMachineLoader struct {
	config       NitroMachineConfig
	machinesLock sync.Mutex
	machines     map[nitroMachineRequest]*loaderMachineStatus
}

func NewNitroMachineLoader(config NitroMachineConfig) *NitroMachineLoader {
	return &NitroMachineLoader{
		config:   config,
		machines: make(map[nitroMachineRequest]*loaderMachineStatus),
	}
}

func (s *loaderMachineStatus) waitForMachine(ctx context.Context) (*ArbitratorMachine, error) {
	select {
	case <-s.chanSignal:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if s.err != nil {
		return nil, s.err
	}
	if s.machine == nil {
		return nil, errors.New("machine is nil")
	}
	return s.machine, nil
}

func (l *NitroMachineLoader) createMachineImpl(moduleRoot common.Hash, untilHostIo bool) (*loaderMachineStatus, error) {
	machineRequest := nitroMachineRequest{
		moduleRoot:  moduleRoot,
		untilHostIo: untilHostIo,
	}

	// Fast path: check if we already have the machine
	l.machinesLock.Lock()
	machine, ok := l.machines[machineRequest]
	if ok {
		return machine, nil
	}
	l.machinesLock.Unlock()

	// Attempt to resolve any alias to the module root (due to the latest machine being separate).
	realModuleRoot := moduleRoot
	if moduleRoot == (common.Hash{}) {
		var err error
		realModuleRoot, err = l.config.ReadLatestWasmModuleRoot()
		if err != nil {
			return nil, err
		}
	} else {
		_, err := os.Stat(filepath.Join(l.config.getMachinePath(moduleRoot), l.config.ProverBinPath))
		if errors.Is(err, os.ErrNotExist) {
			// Attempt to load the latest module root instead (maybe it's what we're looking for).
			originalErr := err
			realModuleRoot, err = l.config.ReadLatestWasmModuleRoot()
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					// Be nice and return the original error, as it's clarifies what went wrong.
					return nil, originalErr
				} else {
					return nil, err
				}
			}
			if realModuleRoot == moduleRoot {
				// The latest machine is the requested one! Pretend we're loading the latest machine instead.
				moduleRoot = common.Hash{}
				machineRequest.moduleRoot = common.Hash{}
			} else {
				// The latest machine is different, so return the original error loading this machine.
				return nil, originalErr
			}
		} else if err != nil {
			return nil, err
		}
	}

	l.machinesLock.Lock()
	defer l.machinesLock.Unlock()

	realMachineRequest := nitroMachineRequest{
		moduleRoot:  realModuleRoot,
		untilHostIo: untilHostIo,
	}
	machine, ok = l.machines[machineRequest]
	if !ok && moduleRoot != realModuleRoot {
		machine, ok = l.machines[realMachineRequest]
	}

	if !ok {
		machine = &loaderMachineStatus{
			chanSignal: make(chan struct{}),
		}
		l.machines[machineRequest] = machine
		if moduleRoot != realModuleRoot {
			l.machines[realMachineRequest] = machine
		}

		go func() {
			if untilHostIo {
				zeroStep, err := l.GetMachine(context.Background(), moduleRoot, false)
				if err != nil {
					machine.err = err
					machine.signalReady()
				} else {
					machine.createHostIoMachineInternal(l.config, moduleRoot, zeroStep)
				}
			} else {
				machine.createZeroStepMachineInternal(l.config, moduleRoot, realModuleRoot)
			}
		}()
	}

	return machine, nil
}

// Starts work on creating the machine in a separate goroutine
// Returns immediately. Can be called multiple times.
func (l *NitroMachineLoader) CreateMachine(moduleRoot common.Hash, untilHostIo bool) error {
	_, err := l.createMachineImpl(moduleRoot, untilHostIo)
	return err
}

// Gets machine when one is ready
// Returns with proper error if context aborts
func (l *NitroMachineLoader) GetMachine(ctx context.Context, moduleRoot common.Hash, untilHostIo bool) (*ArbitratorMachine, error) {
	machine, err := l.createMachineImpl(moduleRoot, untilHostIo)
	if err != nil {
		return nil, err
	}
	return machine.waitForMachine(ctx)
}

func (l *NitroMachineLoader) GetConfig() NitroMachineConfig {
	return l.config
}
