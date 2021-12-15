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
)

// Holds an arbitrator machine pointer, and manages its lifetime
type ArbitratorMachine struct {
	ptr *C.struct_Machine
}

func freeMachine(mach *ArbitratorMachine) {
	C.arbitrator_free_machine(mach.ptr)
}

func machineFromPointer(ptr *C.struct_Machine) *ArbitratorMachine {
	mach := &ArbitratorMachine{ptr: ptr}
	runtime.SetFinalizer(mach, freeMachine)
	return mach
}

func (m *ArbitratorMachine) Clone() *ArbitratorMachine {
	defer runtime.KeepAlive(m)
	return machineFromPointer(C.arbitrator_clone_machine(m.ptr))
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
