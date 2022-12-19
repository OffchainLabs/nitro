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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type MachineLocator struct {
	rootPath string
	latest   common.Hash
}

var ErrMachineNotFound = errors.New("machine not found")

func NewMachineLocator(rootPath string) (*MachineLocator, error) {
	var places []string

	if rootPath != "" {
		places = append(places, rootPath)
	} else {
		// Check the project dir: <project>/arbnode/node.go => ../../target/machines
		_, thisFile, _, ok := runtime.Caller(0)
		if !ok {
			panic("failed to find root path")
		}
		projectDir := filepath.Dir(filepath.Dir(thisFile))
		projectPath := filepath.Join(filepath.Join(projectDir, "target"), "machines")
		places = append(places, projectPath)

		// Check the working directory: ./machines and ./target/machines
		workDir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		workPath1 := filepath.Join(workDir, "machines")
		workPath2 := filepath.Join(filepath.Join(workDir, "target"), "machines")
		places = append(places, workPath1)
		places = append(places, workPath2)

		// Check above the executable: <binary> => ../../machines
		execfile, err := os.Executable()
		if err != nil {
			return nil, err
		}
		execPath := filepath.Join(filepath.Dir(filepath.Dir(execfile)), "machines")
		places = append(places, execPath)
	}

	for _, place := range places {
		if _, err := os.Stat(place); err == nil {
			var latestModuleRoot common.Hash
			latestModuleRootPath := filepath.Join(place, "latest", "module-root.txt")
			fileBytes, err := os.ReadFile(latestModuleRootPath)
			if err == nil {
				s := strings.TrimSpace(string(fileBytes))
				latestModuleRoot = common.HexToHash(s)
			}
			return &MachineLocator{place, latestModuleRoot}, nil
		}
	}
	return nil, ErrMachineNotFound
}

func (l MachineLocator) getMachinePath(moduleRoot common.Hash) string {
	if moduleRoot == (common.Hash{}) || moduleRoot == l.latest {
		return filepath.Join(l.rootPath, "latest")
	} else {
		return filepath.Join(l.rootPath, moduleRoot.String())
	}
}

func (l MachineLocator) LatestWasmModuleRoot() common.Hash {
	return l.latest
}

func createArbMachineThread(ctx context.Context, locator *MachineLocator, config *ArbitratorMachineConfig, moduleRoot common.Hash, status *machineStatus[arbMachines]) {
	binPath := filepath.Join(locator.getMachinePath(moduleRoot), config.WavmBinaryPath)
	cBinPath := C.CString(binPath)
	defer C.free(unsafe.Pointer(cBinPath))
	log.Info("creating nitro machine", "binpath", binPath)
	baseMachine := C.arbitrator_load_wavm_binary(cBinPath)
	if baseMachine == nil {
		status.err = errors.New("failed to load base machine")
		return
	}
	machine := machineFromPointer(baseMachine)
	machineModuleRoot := machine.GetModuleRoot()
	if machineModuleRoot != moduleRoot {
		status.err = fmt.Errorf("attempting to load module root %v got machine with module root %v", moduleRoot, machineModuleRoot)
		return
	}
	status.machine = &arbMachines{
		zeroStep: machine,
	}
	status.machine.zeroStep.Freeze()
	machine = status.machine.zeroStep.Clone()

	// We try to store/load state before first host_io to a file.
	// We will chicken out of that if something fails, but still try to calculate the machine
	statePath := filepath.Join(locator.getMachinePath(moduleRoot), config.UntilHostIoStatePath)
	_, err := os.Stat(statePath)
	if err == nil {
		log.Info("found cached machine until host io state", "moduleRoot", moduleRoot)

		err := machine.DeserializeAndReplaceState(statePath)
		if err != nil {
			// Safe as if DeserializeAndReplaceState returns an error it will not have mutated the machine
			log.Warn("failed to load machine until host io state; will reexecute", "err", err)
		} else {
			status.machine.hostIo = machine
			status.machine.hostIo.Freeze()
			return
		}
	} else if errors.Is(err, os.ErrNotExist) {
		log.Info("didn't find cached machine until host io state", "path", statePath)
	} else {
		log.Warn("error checking if machine until host io state is cached", "path", statePath, "err", err)
	}

	status.err = machine.StepUntilHostIo(ctx)
	if status.err != nil {
		return
	}

	if machine.IsErrored() {
		status.err = errors.New("machine entered errored state while caching execution up to host io")
		return
	}

	status.machine.hostIo = machine
	status.machine.hostIo.Freeze()
}
