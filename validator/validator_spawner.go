package validator

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/pkg/errors"
)

type ValidationSpawner struct {
	machineLoader *NitroMachineLoader
}

func NewValidationSpawner(config NitroMachineConfig, fatalErrChan chan error) (*ValidationSpawner, error) {
	// TODO
	// // the machine will be lazily created if need be later otherwise
	// if config.ArbitratorValidator {
	// 	if err := validationSpawner.CreateMachine(validator.pendingWasmModuleRoot, true, false); err != nil {
	// 		return nil, err
	// 	}
	// }
	// if config.JitValidator {
	// 	if err := validationSpawner.CreateMachine(validator.pendingWasmModuleRoot, true, true); err != nil {
	// 		return nil, err
	// 	}
	// }
	machineLoader := newNitroMachineLoader(config, fatalErrChan)
	return &ValidationSpawner{
		machineLoader: machineLoader,
	}, nil
}

func (s *ValidationSpawner) LatestWasmModuleRoot() (common.Hash, error) {
	return s.machineLoader.GetConfig().ReadLatestWasmModuleRoot()
}

func (v *ValidationSpawner) loadEntryToMachine(ctx context.Context, entry *ValidationInput, mach *ArbitratorMachine) error {
	resolver := func(hash common.Hash) ([]byte, error) {
		// Check if it's a known preimage
		if preimage, ok := entry.Preimages[hash]; ok {
			return preimage, nil
		}
		return nil, errors.New("preimage not found")
	}
	if err := mach.SetPreimageResolver(resolver); err != nil {
		return err
	}
	err := mach.SetGlobalState(entry.StartState)
	if err != nil {
		log.Error("error while setting global state for proving", "err", err, "gsStart", entry.StartState)
		return fmt.Errorf("error while setting global state for proving: %w", err)
	}
	for _, batch := range entry.BatchInfo {
		err = mach.AddSequencerInboxMessage(batch.Number, batch.Data)
		if err != nil {
			log.Error(
				"error adding sequencer msg for proving",
				"err", err, "seq", entry.StartState.Batch, "blockNr", entry.Id,
			)
			return fmt.Errorf("error adding sequencer msg for proving: %w", err)
		}
	}
	for call, wasm := range entry.UserWasms {
		err = mach.AddUserWasm(call, wasm)
		if err != nil {
			log.Error(
				"error adding user wasm for proving",
				"err", colors.Uncolor(err.Error()), "address", call.Address, "blockNr", entry.Id,
			)
			return fmt.Errorf("error adding user wasm for proving:\n%w", err)
		}
	}
	if entry.HasDelayedMsg {
		err = mach.AddDelayedInboxMessage(entry.DelayedMsgNr, entry.DelayedMsg)
		if err != nil {
			log.Error(
				"error adding delayed msg for proving",
				"err", err, "seq", entry.DelayedMsgNr, "blockNr", entry.Id,
			)
			return fmt.Errorf("error adding delayed msg for proving: %w", err)
		}
	}
	return nil
}

func (v *ValidationSpawner) ExecuteArbitrator(
	ctx context.Context, entry *ValidationInput, moduleRoot common.Hash,
) (GoGlobalState, error) {
	basemachine, err := v.machineLoader.GetMachine(ctx, moduleRoot, true)
	if err != nil {
		return GoGlobalState{}, fmt.Errorf("unable to get WASM machine: %w", err)
	}

	mach := basemachine.Clone()
	err = v.loadEntryToMachine(ctx, entry, mach)
	if err != nil {
		return GoGlobalState{}, err
	}
	var steps uint64
	for mach.IsRunning() {
		var count uint64 = 500000000
		err = mach.Step(ctx, count)
		if steps > 0 {
			log.Debug("validation", "moduleRoot", moduleRoot, "block", entry.Id, "steps", steps)
		}
		if err != nil {
			return GoGlobalState{}, fmt.Errorf("machine execution failed with error: %w", err)
		}
		steps += count
	}
	if mach.IsErrored() {
		log.Error("machine entered errored state during attempted validation", "block", entry.Id)
		return GoGlobalState{}, errors.New("machine entered errored state during attempted validation")
	}
	return mach.GetGlobalState(), nil
}

func (v *ValidationSpawner) ExecuteJit(
	ctx context.Context, entry *ValidationInput, moduleRoot common.Hash,
) (GoGlobalState, error) {
	empty := GoGlobalState{}

	machine, err := v.machineLoader.GetJitMachine(ctx, moduleRoot, true)
	if err != nil {
		return empty, fmt.Errorf("unabled to get WASM machine: %w", err)
	}

	resolver := func(hash common.Hash) ([]byte, error) {
		// Check if it's a known preimage
		if preimage, ok := entry.Preimages[hash]; ok {
			return preimage, nil
		}
		return nil, errors.New("preimage not found")
	}
	state, err := machine.prove(ctx, entry, resolver)
	return state, err
}

var launchTime = time.Now().Format("2006_01_02__15_04")

//nolint:gosec
func (v *ValidationSpawner) WriteToFile(outPath string, input *ValidationInput, expOut GoGlobalState, moduleRoot common.Hash, sequencerMsg []byte) error {
	machConf := v.machineLoader.GetConfig()
	outDirPath := filepath.Join(machConf.RootPath, outPath, launchTime, fmt.Sprintf("block_%d", input.Id))
	err := os.MkdirAll(outDirPath, 0755)
	if err != nil {
		return err
	}

	rootPathAssign := ""
	if executable, err := os.Executable(); err == nil {
		rootPathAssign = "ROOTPATH=\"" + filepath.Dir(executable) + "\"\n"
	}
	cmdFile, err := os.OpenFile(filepath.Join(outDirPath, "run-prover.sh"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer cmdFile.Close()
	_, err = cmdFile.WriteString("#!/bin/bash\n" +
		fmt.Sprintf("# expected output: batch %d, postion %d, hash %s\n", expOut.Batch, expOut.PosInBatch, expOut.BlockHash) +
		"MACHPATH=\"" + machConf.getMachinePath(moduleRoot) + "\"\n" +
		rootPathAssign +
		"if (( $# > 1 )); then\n" +
		"	if [[ $1 == \"-m\" ]]; then\n" +
		"		MACHPATH=$2\n" +
		"		shift\n" +
		"		shift\n" +
		"	fi\n" +
		"fi\n" +
		"${ROOTPATH}/bin/prover ${MACHPATH}/" + machConf.ProverBinPath)
	if err != nil {
		return err
	}

	for _, module := range machConf.LibraryPaths {
		_, err = cmdFile.WriteString(" -l " + "${MACHPATH}/" + module)
		if err != nil {
			return err
		}
	}
	_, err = cmdFile.WriteString(fmt.Sprintf(" --inbox-position %d --position-within-message %d --last-block-hash %s", input.StartState.Batch, input.StartState.PosInBatch, input.StartState.BlockHash))
	if err != nil {
		return err
	}

	sequencerFileName := fmt.Sprintf("sequencer_%d.bin", input.StartState.Batch)
	err = os.WriteFile(filepath.Join(outDirPath, sequencerFileName), sequencerMsg, 0644)
	if err != nil {
		return err
	}
	_, err = cmdFile.WriteString(" --inbox " + sequencerFileName)
	if err != nil {
		return err
	}

	preimageFile, err := os.Create(filepath.Join(outDirPath, "preimages.bin"))
	if err != nil {
		return err
	}
	defer preimageFile.Close()
	for _, data := range input.Preimages {
		lenbytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(lenbytes, uint64(len(data)))
		_, err := preimageFile.Write(lenbytes)
		if err != nil {
			return err
		}
		_, err = preimageFile.Write(data)
		if err != nil {
			return err
		}
	}

	_, err = cmdFile.WriteString(" --preimages preimages.bin")
	if err != nil {
		return err
	}

	if input.HasDelayedMsg {
		_, err = cmdFile.WriteString(fmt.Sprintf(" --delayed-inbox-position %d", input.DelayedMsgNr))
		if err != nil {
			return err
		}
		filename := fmt.Sprintf("delayed_%d.bin", input.DelayedMsgNr)
		err = os.WriteFile(filepath.Join(outDirPath, filename), input.DelayedMsg, 0644)
		if err != nil {
			return err
		}
		_, err = cmdFile.WriteString(fmt.Sprintf(" --delayed-inbox %s", filename))
		if err != nil {
			return err
		}
	}

	_, err = cmdFile.WriteString(" \"$@\"\n")
	if err != nil {
		return err
	}
	return nil
}

func (v *ValidationSpawner) CreateExecutionBackend(ctx context.Context, wasmModuleRoot common.Hash, input *ValidationInput, targetMachineNum int) (*ExecutionChallengeBackend, error) {
	initialFrozenMachine, err := v.machineLoader.GetMachine(ctx, wasmModuleRoot, false)
	if err != nil {
		return nil, err
	}
	machine := initialFrozenMachine.Clone()
	err = v.loadEntryToMachine(ctx, input, machine)
	if err != nil {
		return nil, err
	}
	machine.Freeze()
	return NewExecutionChallengeBackend(machine, targetMachineNum, nil)
}

func (v *ValidationSpawner) Stop() {
	v.machineLoader.Stop()
}
