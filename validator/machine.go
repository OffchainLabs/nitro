//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package validator

/*
#cgo CFLAGS: -g -Wall -I../arbitrator/target/env/include/
#include "arbitrator.h"
*/
import "C"
import (
	"context"
	"runtime"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

type MachineInterface interface {
	CloneMachineInterface() MachineInterface
	GetStepCount() uint64
	IsRunning() bool
	ValidForStep(uint64) bool
	Step(context.Context, uint64) error
	Hash() common.Hash
	ProveNextStep() []byte
}

// Holds an arbitrator machine pointer, and manages its lifetime
type ArbitratorMachine struct {
	ptr *C.struct_Machine
}

// Assert that ArbitratorMachine implements MachineInterface
var _ MachineInterface = &ArbitratorMachine{}

func freeMachine(mach *ArbitratorMachine) {
	C.arbitrator_free_machine(mach.ptr)
}

func machineFromPointer(ptr *C.struct_Machine) *ArbitratorMachine {
	mach := &ArbitratorMachine{ptr: ptr}
	runtime.SetFinalizer(mach, freeMachine)
	return mach
}

func LoadSimpleMachine(wasm string, libraries []string) (*ArbitratorMachine, error) {
	cWasm := C.CString(wasm)
	cLibraries := CreateCStringList(libraries)
	mach := C.arbitrator_load_machine(cWasm, cLibraries, C.long(len(libraries)), C.struct_GlobalState{}, C.struct_CMultipleByteArrays{}, nil)
	C.free(unsafe.Pointer(cWasm))
	FreeCStringList(cLibraries, len(libraries))
	if mach == nil {
		return nil, errors.Errorf("failed to load simple machine at path %v", wasm)
	}
	return machineFromPointer(mach), nil
}

func (m *ArbitratorMachine) Clone() *ArbitratorMachine {
	defer runtime.KeepAlive(m)
	return machineFromPointer(C.arbitrator_clone_machine(m.ptr))
}

func (m *ArbitratorMachine) CloneMachineInterface() MachineInterface {
	return m.Clone()
}

func (m *ArbitratorMachine) SetGlobalState(globalState C.struct_GlobalState) {
	defer runtime.KeepAlive(m)
	C.arbitrator_set_global_state(m.ptr, globalState)
}

func (m *ArbitratorMachine) GetGlobalState() C.struct_GlobalState {
	defer runtime.KeepAlive(m)
	return C.arbitrator_global_state(m.ptr)
}

func (m *ArbitratorMachine) GetStepCount() uint64 {
	return uint64(C.arbitrator_get_num_steps(m.ptr))
}

func (m *ArbitratorMachine) IsRunning() bool {
	defer runtime.KeepAlive(m)
	return C.arbitrator_get_status(m.ptr) == C.Running
}

func (m *ArbitratorMachine) ValidForStep(requestedStep uint64) bool {
	haveStep := m.GetStepCount()
	if haveStep > requestedStep {
		return false
	} else if haveStep == requestedStep {
		return true
	} else { // haveStep < requestedStep
		// if the machine is halted, its state persists for future steps
		return !m.IsRunning()
	}
}

func (m *ArbitratorMachine) Step(ctx context.Context, count uint64) error {
	defer runtime.KeepAlive(m)

	var zero C.uint8_t
	conditionByte := &zero
	defer runtime.KeepAlive(conditionByte)

	doneEarlyChan := make(chan struct{})

	go (func() {
		defer runtime.KeepAlive(conditionByte)
		select {
		case <-ctx.Done():
			C.atomic_u8_store(conditionByte, 1)
		case <-doneEarlyChan:
		}
	})()

	C.arbitrator_step(m.ptr, C.uint64_t(count), conditionByte)

	close(doneEarlyChan)

	return ctx.Err()
}

func (m *ArbitratorMachine) Hash() (hash common.Hash) {
	defer runtime.KeepAlive(m)
	bytes := C.arbitrator_hash(m.ptr)
	for i, b := range bytes.bytes {
		hash[i] = byte(b)
	}
	return
}

func (m *ArbitratorMachine) ProveNextStep() []byte {
	defer runtime.KeepAlive(m)

	rustProof := C.arbitrator_gen_proof(m.ptr)
	proofBytes := C.GoBytes(unsafe.Pointer(rustProof.ptr), C.int(rustProof.len))
	C.arbitrator_free_proof(rustProof)

	return proofBytes
}
