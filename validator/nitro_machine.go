//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

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
	"io/fs"
	"io/ioutil"
	"os"
	"path"
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
	RootPath                string // prepends all other paths
	ProverBinPath           string
	ModulePaths             []string
	InitialMachineCachePath string
}

var DefaultNitroMachineConfig = NitroMachineConfig{
	RootPath:                "./target/machines/latest/",
	ProverBinPath:           "replay.wasm",
	ModulePaths:             []string{"soft-float.wasm", "wasi_stub.wasm", "go_stub.wasm", "host_io.wasm", "brotli.wasm"},
	InitialMachineCachePath: "./target/etc/initial-machine-cache",
}

func init() {
	_, thisfile, _, _ := runtime.Caller(0)
	projectDir := filepath.Dir(filepath.Dir(thisfile))
	DefaultNitroMachineConfig.RootPath = filepath.Join(projectDir, "target", "machines", "latest")
}

func (c NitroMachineConfig) ReadWasmModuleRoot() (common.Hash, error) {
	fileToRead := path.Join(c.RootPath, "module_root")
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
	once       sync.Once
}

func (s *loaderMachineStatus) signalReady() {
	close(s.chanSignal)
}

type NitroMachineLoader struct {
	config      NitroMachineConfig
	zeroStep    loaderMachineStatus
	untilHostIo loaderMachineStatus
}

func NewNitroMachineLoader(config NitroMachineConfig) *NitroMachineLoader {
	return &NitroMachineLoader{
		config: config,
		zeroStep: loaderMachineStatus{
			chanSignal: make(chan struct{}),
		},
		untilHostIo: loaderMachineStatus{
			chanSignal: make(chan struct{}),
		},
	}
}

func (l *NitroMachineLoader) createZeroStepMachineInternal() {
	defer l.zeroStep.signalReady()
	moduleList := []string{}
	for _, module := range l.config.ModulePaths {
		moduleList = append(moduleList, filepath.Join(l.config.RootPath, module))
	}
	binPath := filepath.Join(l.config.RootPath, l.config.ProverBinPath)
	cModuleList := CreateCStringList(moduleList)
	cBinPath := C.CString(binPath)
	log.Info("creating nitro machine", "binpath", binPath, "moduleList", moduleList)
	baseMachine := C.arbitrator_load_machine(cBinPath, cModuleList, C.intptr_t(len(moduleList)))
	if baseMachine == nil {
		l.zeroStep.err = errors.New("failed to create base machine")
		return
	}
	FreeCStringList(cModuleList, len(moduleList))
	C.free(unsafe.Pointer(cBinPath))
	l.zeroStep.machine = machineFromPointer(baseMachine)
	l.zeroStep.machine.Freeze()
}

// We try to store/load state before first host_io to a file.
// We will chicken out of that if something fails, but still try to calculate the machine
func (l *NitroMachineLoader) createHostIoMachineInternal() {
	defer l.untilHostIo.signalReady()
	ctx := context.Background()
	zerostep, err := l.GetZeroStepMachine(ctx)
	if err != nil {
		l.untilHostIo.err = err
		return
	}
	machine := zerostep.Clone()
	hash := machine.Hash()
	expectedName := hash.String() + ".bin"
	cacheDir := l.config.InitialMachineCachePath
	foundInCache := false
	saveStateToFile := true
	err = os.MkdirAll(cacheDir, 0o755)
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
				err := os.Remove(path.Join(cacheDir, file.Name()))
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

	file := path.Join(cacheDir, expectedName)
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
			l.untilHostIo.machine = machine
			l.untilHostIo.machine.Freeze()
			return
		}
	} else {
		log.Info("didn't find initial machine in cache", "hash", hash)
	}

	l.untilHostIo.err = machine.StepUntilHostIo(ctx)
	if l.untilHostIo.err != nil {
		return
	}

	if machine.IsErrored() {
		panic("Machine entered errored state while caching execution up to host io")
	}

	l.untilHostIo.machine = machine
	l.untilHostIo.machine.Freeze()
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

// Starts work on creating the machine in a separate goroutine
// Returns immediately. Can be called multiple times.
func (l *NitroMachineLoader) CreateZeroStepMachine() {
	l.zeroStep.once.Do(func() { go l.createZeroStepMachineInternal() })
}

// Starts work on creating the machine in a separate goroutine
// Returns immediately. Can be called multiple times.
func (l *NitroMachineLoader) CreateHostIoMachine() {
	l.untilHostIo.once.Do(func() { go l.createHostIoMachineInternal() })
}

// Gets Zero-Steps machine (used by challenges) when one is ready
// Returns with proper error if context aborts
func (l *NitroMachineLoader) GetZeroStepMachine(ctx context.Context) (*ArbitratorMachine, error) {
	l.CreateZeroStepMachine()
	return l.zeroStep.waitForMachine(ctx)
}

// Gets Zero-Steps machine (used by challenges) when one is ready
// Returns with proper error if context aborts
func (l *NitroMachineLoader) GetHostIoMachine(ctx context.Context) (*ArbitratorMachine, error) {
	l.CreateHostIoMachine()
	return l.untilHostIo.waitForMachine(ctx)
}

func (l *NitroMachineLoader) RecomputeInitialModuleRoot(ctx context.Context) (common.Hash, error) {
	machine, err := l.GetZeroStepMachine(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return machine.GetModuleRoot(), nil
}

func (l *NitroMachineLoader) GetConfig() NitroMachineConfig {
	return l.config
}
