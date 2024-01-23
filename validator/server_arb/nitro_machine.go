// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package server_arb

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
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
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/validator/server_common"
)

func createArbMachine(ctx context.Context, locator *server_common.MachineLocator, config *ArbitratorMachineConfig, moduleRoot common.Hash, opts ...server_common.MachineLoaderOpt) (*arbMachines, error) {
	loaderCfg := &server_common.MachineLoaderCfg{}
	for _, o := range opts {
		o(loaderCfg)
	}
	binPath := filepath.Join(locator.GetMachinePath(moduleRoot), config.WavmBinaryPath)
	cBinPath := C.CString(binPath)
	defer C.free(unsafe.Pointer(cBinPath))

	log.Info("creating nitro machine", "binpath", binPath, "alwaysMerkleize", loaderCfg.ShouldAlwaysMerkleize())
	shouldMerkleize := C.uint8_t(1)
	baseMachine := C.arbitrator_load_wavm_binary(cBinPath, shouldMerkleize)
	if baseMachine == nil {
		return nil, errors.New("failed to load base machine")
	}
	machine := machineFromPointer(baseMachine)
	machineModuleRoot := machine.GetModuleRoot()
	if machineModuleRoot != moduleRoot {
		return nil, fmt.Errorf("attempting to load module root %v got machine with module root %v", moduleRoot, machineModuleRoot)
	}
	result := &arbMachines{
		zeroStep: machine,
	}
	result.zeroStep.Freeze()
	machine = result.zeroStep.Clone()

	// We try to store/load state before first host_io to a file.
	// We will chicken out of that if something fails, but still try to calculate the machine
	statePath := filepath.Join(locator.GetMachinePath(moduleRoot), config.UntilHostIoStatePath)
	_, err := os.Stat(statePath)
	if err == nil {
		log.Info("found cached machine until host io state", "moduleRoot", moduleRoot)

		err := machine.DeserializeAndReplaceState(statePath)
		if err != nil {
			// Safe as if DeserializeAndReplaceState returns an error it will not have mutated the machine
			log.Warn("failed to load machine until host io state; will reexecute", "err", err)
		} else {
			result.hostIo = machine
			result.hostIo.Freeze()
			return result, nil
		}
	} else if errors.Is(err, os.ErrNotExist) {
		log.Info("didn't find cached machine until host io state", "path", statePath)
	} else {
		log.Warn("error checking if machine until host io state is cached", "path", statePath, "err", err)
	}

	if err := machine.StepUntilHostIo(ctx); err != nil {
		return nil, err
	}

	if machine.IsErrored() {
		return nil, errors.New("machine entered errored state while caching execution up to host io")
	}

	result.hostIo = machine
	result.hostIo.Freeze()
	return result, nil
}
