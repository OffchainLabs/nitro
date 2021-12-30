//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package validator

/*
#cgo CFLAGS: -g -Wall -I../arbitrator/target/env/include/
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
	"sync"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type staticMachineData struct {
	machine    *ArbitratorMachine
	chanSignal chan struct{}
	ready      bool
	err        error
	once       sync.Once
}

type NitroMachineConfig struct {
	RootPath                string // prepends all other paths
	ProverBinPath           string
	ModulePaths             []string
	InitialMachineCachePath string
}

var StaticNitroMachineConfig = NitroMachineConfig{
	RootPath:                "./arbitrator/target/env/",
	ProverBinPath:           "lib/replay.wasm",
	ModulePaths:             []string{"lib/wasi_stub.wasm", "lib/soft-float.wasm", "lib/go_stub.wasm", "lib/host_io.wasm"},
	InitialMachineCachePath: "initial-machine-cache",
}

var zeroStepMachine staticMachineData
var hostIoMachine staticMachineData

func init() {
	_, thisfile, _, _ := runtime.Caller(0)
	projectDir := filepath.Dir(filepath.Dir(thisfile))
	StaticNitroMachineConfig.RootPath = filepath.Join(projectDir, "arbitrator", "target", "env")

	zeroStepMachine.chanSignal = make(chan struct{})
	hostIoMachine.chanSignal = make(chan struct{})
}

func createZeroStepMachineInternal() {
	moduleList := []string{}
	for _, module := range StaticNitroMachineConfig.ModulePaths {
		moduleList = append(moduleList, filepath.Join(StaticNitroMachineConfig.RootPath, module))
	}
	cModuleList := CreateCStringList(moduleList)
	cBinPath := C.CString(filepath.Join(StaticNitroMachineConfig.RootPath, StaticNitroMachineConfig.ProverBinPath))

	cZeroPreimages := C.CMultipleByteArrays{}
	cZeroPreimages.len = 0
	baseMachine := C.arbitrator_load_machine(cBinPath, cModuleList, C.intptr_t(len(moduleList)), C.GlobalState{}, cZeroPreimages)
	FreeCStringList(cModuleList, len(moduleList))
	C.free(unsafe.Pointer(cBinPath))
	zeroStepMachine.machine = machineFromPointer(baseMachine)
	zeroStepMachine.machine.Freeze()
	signalReady(&zeroStepMachine)
}

// We try to store/load state before firt host_io to a file.
// We will chicken out of that if something fails, but still try to calculate the machine
func createHostIoMachineInternal() {
	defer signalReady(&hostIoMachine)
	ctx := context.Background()
	zerostep, err := GetZeroStepMachine(ctx)
	if err != nil {
		hostIoMachine.err = err
		return
	}
	machine := zerostep.Clone()
	hash := machine.Hash()
	expectedName := hash.String() + ".bin"
	cacheDir := path.Join(StaticNitroMachineConfig.RootPath, StaticNitroMachineConfig.InitialMachineCachePath)
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
			hostIoMachine.machine = machine
			hostIoMachine.machine.Freeze()
			return
		}
	} else {
		log.Info("didn't find initial machine in cache", "hash", hash)
	}

	hostIoMachine.err = machine.StepUntilHostIo(ctx)
	if hostIoMachine.err != nil {
		return
	}

	hostIoMachine.machine = machine
	hostIoMachine.machine.Freeze()
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

func signalReady(machine *staticMachineData) {
	machine.ready = true
	close(machine.chanSignal)
}

func waitForMachine(ctx context.Context, machine *staticMachineData) (*ArbitratorMachine, error) {
	select {
	case <-machine.chanSignal:
	case <-ctx.Done():
	}
	if machine.err != nil {
		return nil, machine.err
	}
	if machine.ready {
		return machine.machine, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return nil, errors.New("failed to get machine")
}

// Starts work on creating the machine in a separate goroutine
// Returns immediately. Can be called multiple times.
func CreateZeroStepMachine() {
	zeroStepMachine.once.Do(func() { go createZeroStepMachineInternal() })
}

// Starts work on creating the machine in a separate goroutine
// Returns immediately. Can be called multiple times.
func CreateHostIoMachine() {
	hostIoMachine.once.Do(func() { go createHostIoMachineInternal() })
}

// Gets Zero-Steps machine (used by challenges) when one is ready
// Returns with proper error if context aborts
func GetZeroStepMachine(ctx context.Context) (*ArbitratorMachine, error) {
	CreateZeroStepMachine()
	return waitForMachine(ctx, &zeroStepMachine)
}

// Gets Zero-Steps machine (used by challenges) when one is ready
// Returns with proper error if context aborts
func GetHostIoMachine(ctx context.Context) (*ArbitratorMachine, error) {
	CreateHostIoMachine()
	return waitForMachine(ctx, &hostIoMachine)
}

func GetInitialModuleRoot(ctx context.Context) (common.Hash, error) {
	machine, err := GetZeroStepMachine(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return machine.GetModuleRoot(), nil
}
