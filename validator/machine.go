// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validator

/*
#cgo CFLAGS: -g -Wall -I../target/include/
#include "arbitrator.h"

ResolvedPreimage preimageResolverC(size_t context, const uint8_t* hash);
*/
import "C"
import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

type MachineInterface interface {
	CloneMachineInterface() MachineInterface
	GetStepCount() uint64
	IsRunning() bool
	ValidForStep(uint64) bool
	Step(context.Context, uint64) error
	Hash() common.Hash
	GetGlobalState() GoGlobalState
	ProveNextStep() []byte
}

// Holds an arbitrator machine pointer, and manages its lifetime
type ArbitratorMachine struct {
	ptr              *C.struct_Machine
	preimageResolver int64
	frozen           bool // does not allow anything that changes machine state, not cloned with the machine
}

// Assert that ArbitratorMachine implements MachineInterface
var _ MachineInterface = (*ArbitratorMachine)(nil)

var preimageResolvers sync.Map
var lastPreimageResolverId int64 // atomic

func freeMachine(mach *ArbitratorMachine) {
	C.arbitrator_free_machine(mach.ptr)
	if mach.preimageResolver != 0 {
		preimageResolvers.Delete(mach.preimageResolver)
	}
}

func machineFromPointer(ptr *C.struct_Machine) *ArbitratorMachine {
	if ptr == nil {
		return nil
	}
	mach := &ArbitratorMachine{ptr: ptr}
	runtime.SetFinalizer(mach, freeMachine)
	return mach
}

func LoadSimpleMachine(wasm string, libraries []string) (*ArbitratorMachine, error) {
	cWasm := C.CString(wasm)
	cLibraries := CreateCStringList(libraries)
	mach := C.arbitrator_load_machine(cWasm, cLibraries, C.long(len(libraries)))
	C.free(unsafe.Pointer(cWasm))
	FreeCStringList(cLibraries, len(libraries))
	if mach == nil {
		return nil, errors.Errorf("failed to load simple machine at path %v", wasm)
	}
	return machineFromPointer(mach), nil
}

func (m *ArbitratorMachine) Freeze() {
	m.frozen = true
}

// Even if origin is frozen - clone is not
func (m *ArbitratorMachine) Clone() *ArbitratorMachine {
	defer runtime.KeepAlive(m)
	return machineFromPointer(C.arbitrator_clone_machine(m.ptr))
}

func (m *ArbitratorMachine) CloneMachineInterface() MachineInterface {
	return m.Clone()
}

func (m *ArbitratorMachine) SetGlobalState(globalState GoGlobalState) error {
	defer runtime.KeepAlive(m)
	if m.frozen {
		return errors.New("machine frozen")
	}
	cGlobalState := GlobalStateToC(globalState)
	C.arbitrator_set_global_state(m.ptr, cGlobalState)
	return nil
}

func (m *ArbitratorMachine) GetGlobalState() GoGlobalState {
	defer runtime.KeepAlive(m)
	cGlobalState := C.arbitrator_global_state(m.ptr)
	return GlobalStateFromC(cGlobalState)
}

func (m *ArbitratorMachine) GetStepCount() uint64 {
	defer runtime.KeepAlive(m)
	return uint64(C.arbitrator_get_num_steps(m.ptr))
}

func (m *ArbitratorMachine) IsRunning() bool {
	defer runtime.KeepAlive(m)
	return C.arbitrator_get_status(m.ptr) == C.ARBITRATOR_MACHINE_STATUS_RUNNING
}

func (m *ArbitratorMachine) IsErrored() bool {
	defer runtime.KeepAlive(m)
	return C.arbitrator_get_status(m.ptr) == C.ARBITRATOR_MACHINE_STATUS_ERRORED
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

func manageConditionByte(ctx context.Context) (*C.uint8_t, func()) {
	var zero C.uint8_t
	conditionByte := &zero

	doneEarlyChan := make(chan struct{})

	go (func() {
		defer runtime.KeepAlive(conditionByte)
		select {
		case <-ctx.Done():
			C.atomic_u8_store(conditionByte, 1)
		case <-doneEarlyChan:
		}
	})()

	cancel := func() {
		runtime.KeepAlive(conditionByte)
		close(doneEarlyChan)
	}

	return conditionByte, cancel
}

func (m *ArbitratorMachine) Step(ctx context.Context, count uint64) error {
	defer runtime.KeepAlive(m)

	if m.frozen {
		return errors.New("machine frozen")
	}
	conditionByte, cancel := manageConditionByte(ctx)
	defer cancel()

	err := C.arbitrator_step(m.ptr, C.uint64_t(count), conditionByte)
	if err != nil {
		errString := C.GoString(err)
		C.free(unsafe.Pointer(err))
		return errors.New(errString)
	}

	return ctx.Err()
}

func (m *ArbitratorMachine) StepUntilHostIo(ctx context.Context) error {
	defer runtime.KeepAlive(m)
	if m.frozen {
		return errors.New("machine frozen")
	}

	conditionByte, cancel := manageConditionByte(ctx)
	defer cancel()

	C.arbitrator_step_until_host_io(m.ptr, conditionByte)

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

func (m *ArbitratorMachine) GetModuleRoot() (hash common.Hash) {
	defer runtime.KeepAlive(m)
	bytes := C.arbitrator_module_root(m.ptr)
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

func (m *ArbitratorMachine) SerializeState(path string) error {
	defer runtime.KeepAlive(m)

	cPath := C.CString(path)
	status := C.arbitrator_serialize_state(m.ptr, cPath)
	C.free(unsafe.Pointer(cPath))

	if status != 0 {
		return errors.New("failed to serialize machine state")
	} else {
		return nil
	}
}

func (m *ArbitratorMachine) DeserializeAndReplaceState(path string) error {
	defer runtime.KeepAlive(m)

	if m.frozen {
		return errors.New("machine frozen")
	}

	cPath := C.CString(path)
	status := C.arbitrator_deserialize_and_replace_state(m.ptr, cPath)
	C.free(unsafe.Pointer(cPath))

	if status != 0 {
		return errors.New("failed to deserialize machine state")
	} else {
		return nil
	}
}

func (m *ArbitratorMachine) AddSequencerInboxMessage(index uint64, data []byte) error {
	defer runtime.KeepAlive(m)

	if m.frozen {
		return errors.New("machine frozen")
	}
	cbyte := CreateCByteArray(data)
	status := C.arbitrator_add_inbox_message(m.ptr, C.uint64_t(0), C.uint64_t(index), cbyte)
	DestroyCByteArray(cbyte)
	if status != 0 {
		return errors.New("failed to add sequencer inbox message")
	} else {
		return nil
	}
}

func (m *ArbitratorMachine) AddDelayedInboxMessage(index uint64, data []byte) error {
	defer runtime.KeepAlive(m)

	if m.frozen {
		return errors.New("machine frozen")
	}

	cbyte := CreateCByteArray(data)
	status := C.arbitrator_add_inbox_message(m.ptr, C.uint64_t(1), C.uint64_t(index), cbyte)
	DestroyCByteArray(cbyte)
	if status != 0 {
		return errors.New("failed to add sequencer inbox message")
	} else {
		return nil
	}
}

type GoPreimageResolver = func(common.Hash) ([]byte, error)

//export preimageResolver
func preimageResolver(context C.size_t, ptr unsafe.Pointer) C.ResolvedPreimage {
	var hash common.Hash
	input := (*[1 << 30]byte)(ptr)[:32]
	copy(hash[:], input)
	resolver, ok := preimageResolvers.Load(int64(context))
	if !ok {
		return C.ResolvedPreimage{
			len: -1,
		}
	}
	resolverFunc, ok := resolver.(GoPreimageResolver)
	if !ok {
		log.Warn("preimage resolver has wrong type")
		return C.ResolvedPreimage{
			len: -1,
		}
	}
	preimage, err := resolverFunc(hash)
	if err != nil {
		log.Error("preimage resolution failed", "err", err)
		return C.ResolvedPreimage{
			len: -1,
		}
	}
	return C.ResolvedPreimage{
		ptr: (*C.uint8_t)(C.CBytes(preimage)),
		len: (C.ptrdiff_t)(len(preimage)),
	}
}

func (m *ArbitratorMachine) SetPreimageResolver(resolver GoPreimageResolver) error {
	if m.frozen {
		return errors.New("machine frozen")
	}
	if m.preimageResolver != 0 {
		return errors.New("attempted to set preimage resolver twice on machine")
	}
	id := atomic.AddInt64(&lastPreimageResolverId, 1)
	preimageResolvers.Store(id, resolver)
	m.preimageResolver = id
	C.arbitrator_set_preimage_resolver(m.ptr, C.size_t(id), (*[0]byte)(C.preimageResolverC))
	return nil
}
