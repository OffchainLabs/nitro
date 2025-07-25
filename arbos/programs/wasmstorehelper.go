// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !wasm
// +build !wasm

package programs

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
)

// SaveActiveProgramToWasmStore is used to save active stylus programs to wasm store during rebuilding
func (p Programs) SaveActiveProgramToWasmStore(statedb *state.StateDB, codeHash common.Hash, code []byte, time uint64, debugMode bool, rebuildingStartBlockTime uint64, targets []rawdb.WasmTarget) error {
	progParams, err := p.Params()
	if err != nil {
		return err
	}

	program, err := p.getActiveProgram(codeHash, time, progParams)
	if err != nil {
		// The program is not active so return early
		log.Info("program is not active, getActiveProgram returned error, hence do not include in rebuilding", "err", err)
		return nil
	}

	// It might happen that node crashed some time after rebuilding commenced and before it completed, hence when rebuilding
	// resumes after node is restarted the latest diskdb derived from statedb might now have codehashes that were activated
	// during the last rebuilding session. In such cases we don't need to fetch moduleshashes but instead return early
	// since they would already be added to the wasm store
	currentHoursSince := hoursSinceArbitrum(rebuildingStartBlockTime)
	if currentHoursSince < program.activatedAt {
		return nil
	}

	moduleHash, err := p.moduleHashes.Get(codeHash)
	if err != nil {
		return err
	}

	_, missingTargets, err := statedb.ActivatedAsmMap(targets, moduleHash)
	if err != nil {
		return err
	}
	// If already in wasm store then return early
	if len(missingTargets) == 0 {
		return nil
	}

	wasm, err := getWasmFromContractCode(code, progParams.MaxWasmSize)
	if err != nil {
		log.Error("Failed to reactivate program while rebuilding wasm store: getWasmFromContractCode", "expected moduleHash", moduleHash, "err", err)
		return fmt.Errorf("failed to reactivate program while rebuilding wasm store: %w", err)
	}

	// don't charge gas
	zeroArbosVersion := uint64(0)
	zeroGas := uint64(0)

	// We know program is activated, so it must be in correct version and not use too much memory
	// Empty program address is supplied because we dont have access to this during rebuilding of wasm store
	moduleActivationMandatory := false
	// recompile only missing targets
	info, asmMap, err := activateProgramInternal(common.Address{}, codeHash, wasm, progParams.PageLimit, program.version, zeroArbosVersion, debugMode, &zeroGas, missingTargets, moduleActivationMandatory)
	if err != nil {
		log.Error("failed to reactivate program while rebuilding wasm store", "expected moduleHash", moduleHash, "err", err)
		return fmt.Errorf("failed to reactivate program while rebuilding wasm store: %w", err)
	}

	if info != nil && info.moduleHash != moduleHash {
		log.Error("failed to reactivate program while rebuilding wasm store", "expected moduleHash", moduleHash, "got", info.moduleHash)
		return fmt.Errorf("failed to reactivate program while rebuilding wasm store, expected ModuleHash: %v", moduleHash)
	}

	batch := statedb.Database().WasmStore().NewBatch()
	rawdb.WriteActivation(batch, moduleHash, asmMap)
	if err := batch.Write(); err != nil {
		log.Error("failed writing re-activation to state while rebuilding wasm store", "err", err)
		return err
	}

	return nil
}
