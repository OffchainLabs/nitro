// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

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
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_common"
)

func createArbMachine(ctx context.Context, locator *server_common.MachineLocator, config *ArbitratorMachineConfig, moduleRoot common.Hash) (*arbMachines, error) {
	binPath := filepath.Join(locator.GetMachinePath(moduleRoot), config.WavmBinaryPath)
	cBinPath := C.CString(binPath)
	defer C.free(unsafe.Pointer(cBinPath))
	log.Info("creating nitro machine", "binpath", binPath)
	baseMachine := C.arbitrator_load_wavm_binary(cBinPath)
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

func CreateTestArbMachine(ctx context.Context, locator *server_common.MachineLocator, entry *validator.ValidationInput) (*ArbitratorMachine, error) {
	binPath := filepath.Join(locator.GetMachinePath(common.Hash{}), DefaultArbitratorMachineConfig.WavmBinaryPath)
	cBinPath := C.CString(binPath)
	defer C.free(unsafe.Pointer(cBinPath))
	log.Info("creating nitro machine", "binpath", binPath)
	baseMachine := C.arbitrator_load_wavm_binary(cBinPath)
	if baseMachine == nil {
		return nil, errors.New("failed to load base machine")
	}
	mach := machineFromPointer(baseMachine)
	resolver := func(ty arbutil.PreimageType, hash common.Hash) ([]byte, error) {
		// Check if it's a known preimage
		if preimage, ok := entry.Preimages[ty][hash]; ok {
			return preimage, nil
		}
		return nil, errors.New("preimage not found")
	}
	if err := mach.SetPreimageResolver(resolver); err != nil {
		return nil, err
	}
	err := mach.SetGlobalState(entry.StartState)
	if err != nil {
		log.Error("error while setting global state for proving", "err", err, "gsStart", entry.StartState)
		return nil, fmt.Errorf("error while setting global state for proving: %w", err)
	}
	for _, batch := range entry.BatchInfo {
		err = mach.AddSequencerInboxMessage(batch.Number, batch.Data)
		if err != nil {
			log.Error(
				"error while trying to add sequencer msg for proving",
				"err", err, "seq", entry.StartState.Batch, "blockNr", entry.Id,
			)
			return nil, fmt.Errorf("error while trying to add sequencer msg for proving: %w", err)
		}
	}
	if entry.HasDelayedMsg {
		err = mach.AddDelayedInboxMessage(entry.DelayedMsgNr, entry.DelayedMsg)
		if err != nil {
			log.Error(
				"error while trying to add delayed msg for proving",
				"err", err, "seq", entry.DelayedMsgNr, "blockNr", entry.Id,
			)
			return nil, fmt.Errorf("error while trying to add delayed msg for proving: %w", err)
		}
	}
	err = mach.AddHotShotCommitment(entry.BlockHeight, entry.HotShotCommitment[:])
	if err != nil {
		log.Error("error while setting hotshot commitment: %w", err)
		return nil, fmt.Errorf("error while setting hotshot commitment: %w", err)
	}

	err = mach.AddHotShotLiveness(entry.BlockHeight, entry.HotShotLiveness)
	if err != nil {
		log.Error("error while setting hotshot liveness: %w", err)
		return nil, fmt.Errorf("error while setting hotshot liveness: %w", err)
	}

	return mach, nil
}
