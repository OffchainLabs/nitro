// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package server_arb

/*
#cgo CFLAGS: -g -I../../target/include/
#include "arbitrator.h"

ResolvedPreimage preimageResolverC(size_t context, uint8_t preimageType, const uint8_t* hash);
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/validator"
)

type u8 = C.uint8_t
type u16 = C.uint16_t
type u32 = C.uint32_t
type u64 = C.uint64_t
type usize = C.size_t

type MachineInterface interface {
	CloneMachineInterface() MachineInterface
	GetStepCount() uint64
	IsRunning() bool
	IsErrored() bool
	ValidForStep(uint64) bool
	Status() uint8
	Step(context.Context, uint64) error
	Hash() common.Hash
	GetGlobalState() validator.GoGlobalState
	ProveNextStep() []byte
	GetNextOpcode() uint16
	Freeze()
	Destroy()
}

// ArbitratorMachine holds an arbitrator machine pointer, and manages its lifetime
type ArbitratorMachine struct {
	mutex     sync.Mutex // needed because go finalizers don't synchronize (meaning they aren't thread safe)
	ptr       *C.struct_Machine
	contextId *int64
	frozen    bool // does not allow anything that changes machine state, not cloned with the machine
}

// Assert that ArbitratorMachine implements MachineInterface
var _ MachineInterface = (*ArbitratorMachine)(nil)

var preimageResolvers containers.SyncMap[int64, goPreimageResolverWithRefCounter]
var lastPreimageResolverId atomic.Int64 // atomic

func dereferenceContextId(contextId *int64) {
	if contextId != nil {
		resolverWithRefCounter, ok := preimageResolvers.Load(*contextId)
		if !ok {
			panic(fmt.Sprintf("dereferenceContextId: resolver with ref counter not found, contextId: %v", *contextId))
		}

		refCount := resolverWithRefCounter.refCounter.Add(-1)
		if refCount < 0 {
			panic(fmt.Sprintf("dereferenceContextId: ref counter is negative, contextId: %v", *contextId))
		} else if refCount == 0 {
			preimageResolvers.Delete(*contextId)
		}
	}
}

// Any future calls to this machine will result in a panic
func (m *ArbitratorMachine) Destroy() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.ptr != nil {
		C.arbitrator_free_machine(m.ptr)
		m.ptr = nil
		// We no longer need a finalizer
		runtime.SetFinalizer(m, nil)
	}

	dereferenceContextId(m.contextId)
	m.contextId = nil
}

func machineFromPointer(ptr *C.struct_Machine) *ArbitratorMachine {
	if ptr == nil {
		return nil
	}
	mach := &ArbitratorMachine{ptr: ptr}
	C.arbitrator_set_preimage_resolver(ptr, (*[0]byte)(C.preimageResolverC))
	runtime.SetFinalizer(mach, (*ArbitratorMachine).Destroy)
	return mach
}

func LoadSimpleMachine(wasm string, libraries []string, debugChain bool) (*ArbitratorMachine, error) {
	cWasm := C.CString(wasm)
	cLibraries := CreateCStringList(libraries)
	debug := usize(arbmath.BoolToUint32(debugChain))
	mach := C.arbitrator_load_machine(cWasm, cLibraries, C.long(len(libraries)), debug)
	C.free(unsafe.Pointer(cWasm))
	FreeCStringList(cLibraries, len(libraries))
	if mach == nil {
		return nil, fmt.Errorf("failed to load simple machine at path %v", wasm)
	}
	return machineFromPointer(mach), nil
}

func NewFinishedMachine(gs validator.GoGlobalState) *ArbitratorMachine {
	mach := C.arbitrator_new_finished(GlobalStateToC(gs))
	if mach == nil {
		return nil
	}
	return machineFromPointer(mach)
}

func (m *ArbitratorMachine) Freeze() {
	m.frozen = true
}

// Even if origin is frozen - clone is not
func (m *ArbitratorMachine) Clone() *ArbitratorMachine {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	newMach := machineFromPointer(C.arbitrator_clone_machine(m.ptr))
	newMach.contextId = m.contextId

	if m.contextId != nil {
		resolverWithRefCounter, ok := preimageResolvers.Load(*m.contextId)
		if ok {
			resolverWithRefCounter.refCounter.Add(1)
		} else {
			panic(fmt.Sprintf("Clone: resolver with ref counter not found, contextId: %v", *m.contextId))
		}
	}

	return newMach
}

func (m *ArbitratorMachine) CloneMachineInterface() MachineInterface {
	return m.Clone()
}

func (m *ArbitratorMachine) SetGlobalState(globalState validator.GoGlobalState) error {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.frozen {
		return errors.New("machine frozen")
	}
	cGlobalState := GlobalStateToC(globalState)
	C.arbitrator_set_global_state(m.ptr, cGlobalState)
	return nil
}

func (m *ArbitratorMachine) GetGlobalState() validator.GoGlobalState {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	cGlobalState := C.arbitrator_global_state(m.ptr)
	return GlobalStateFromC(cGlobalState)
}

func (m *ArbitratorMachine) GetStepCount() uint64 {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return uint64(C.arbitrator_get_num_steps(m.ptr))
}

func (m *ArbitratorMachine) IsRunning() bool {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return C.arbitrator_get_status(m.ptr) == C.ARBITRATOR_MACHINE_STATUS_RUNNING
}

func (m *ArbitratorMachine) IsErrored() bool {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return C.arbitrator_get_status(m.ptr) == C.ARBITRATOR_MACHINE_STATUS_ERRORED
}

func (m *ArbitratorMachine) Status() uint8 {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return uint8(C.arbitrator_get_status(m.ptr))
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

func manageConditionByte(ctx context.Context) (*u8, func()) {
	var zero u8
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
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.frozen {
		return errors.New("machine frozen")
	}
	conditionByte, cancel := manageConditionByte(ctx)
	defer cancel()

	err := C.arbitrator_step(m.ptr, u64(count), conditionByte)
	defer C.free(unsafe.Pointer(err))
	if err != nil {
		return errors.New(C.GoString(err))
	}

	return ctx.Err()
}

func (m *ArbitratorMachine) StepUntilHostIo(ctx context.Context) error {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.frozen {
		return errors.New("machine frozen")
	}

	conditionByte, cancel := manageConditionByte(ctx)
	defer cancel()

	err := C.arbitrator_step_until_host_io(m.ptr, conditionByte)
	defer C.free(unsafe.Pointer(err))
	if err != nil {
		return errors.New(C.GoString(err))
	}

	return ctx.Err()
}

func (m *ArbitratorMachine) Hash() (hash common.Hash) {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	bytes := C.arbitrator_hash(m.ptr)
	for i, b := range bytes.bytes {
		hash[i] = byte(b)
	}
	return
}

func (m *ArbitratorMachine) GetModuleRoot() (hash common.Hash) {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	bytes := C.arbitrator_module_root(m.ptr)
	for i, b := range bytes.bytes {
		hash[i] = byte(b)
	}
	return
}

func (m *ArbitratorMachine) ProveNextStep() []byte {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	output := &C.RustBytes{}
	C.arbitrator_gen_proof(m.ptr, output)
	defer C.free_rust_bytes(*output)
	if output.len == 0 {
		return nil
	}
	proofBytes := C.GoBytes(unsafe.Pointer(output.ptr), C.int(output.len))

	return proofBytes
}

func (m *ArbitratorMachine) GetNextOpcode() uint16 {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return uint16(C.arbitrator_get_opcode(m.ptr))
}

func (m *ArbitratorMachine) SerializeState(path string) error {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()

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
	m.mutex.Lock()
	defer m.mutex.Unlock()

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
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.frozen {
		return errors.New("machine frozen")
	}
	cbyte := CreateCByteArray(data)
	status := C.arbitrator_add_inbox_message(m.ptr, u64(0), u64(index), cbyte)
	DestroyCByteArray(cbyte)
	if status != 0 {
		return errors.New("failed to add sequencer inbox message")
	} else {
		return nil
	}
}

func (m *ArbitratorMachine) AddDelayedInboxMessage(index uint64, data []byte) error {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.frozen {
		return errors.New("machine frozen")
	}

	cbyte := CreateCByteArray(data)
	status := C.arbitrator_add_inbox_message(m.ptr, u64(1), u64(index), cbyte)
	DestroyCByteArray(cbyte)
	if status != 0 {
		return errors.New("failed to add sequencer inbox message")
	} else {
		return nil
	}
}

type GoPreimageResolver = func(arbutil.PreimageType, common.Hash) ([]byte, error)
type goPreimageResolverWithRefCounter struct {
	resolver   GoPreimageResolver
	refCounter *atomic.Int64
}

//export preimageResolver
func preimageResolver(context C.size_t, ty C.uint8_t, ptr unsafe.Pointer) C.ResolvedPreimage {
	var hash common.Hash
	input := (*[1 << 30]byte)(ptr)[:32]
	copy(hash[:], input)
	resolverWithRefCounter, ok := preimageResolvers.Load(int64(context))
	if !ok {
		log.Error("preimageResolver: resolver with ref counter not found", "context", int64(context))
		return C.ResolvedPreimage{
			len: -1,
		}
	}
	preimage, err := resolverWithRefCounter.resolver(arbutil.PreimageType(ty), hash)
	if err != nil {
		log.Error("preimage resolution failed", "err", err)
		return C.ResolvedPreimage{
			len: -1,
		}
	}
	return C.ResolvedPreimage{
		ptr: (*u8)(C.CBytes(preimage)),
		len: (C.ptrdiff_t)(len(preimage)),
	}
}

func (m *ArbitratorMachine) SetPreimageResolver(resolver GoPreimageResolver) error {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.frozen {
		return errors.New("machine frozen")
	}
	dereferenceContextId(m.contextId)

	id := lastPreimageResolverId.Add(1)
	refCounter := atomic.Int64{}
	refCounter.Store(1)
	resolverWithRefCounter := goPreimageResolverWithRefCounter{
		resolver:   resolver,
		refCounter: &refCounter,
	}
	preimageResolvers.Store(id, resolverWithRefCounter)

	m.contextId = &id
	C.arbitrator_set_context(m.ptr, u64(id))
	return nil
}

func (m *ArbitratorMachine) AddUserWasm(moduleHash common.Hash, module []byte) error {
	defer runtime.KeepAlive(m)
	if m.frozen {
		return errors.New("machine frozen")
	}
	hashBytes := [32]u8{}
	for index, byte := range moduleHash.Bytes() {
		hashBytes[index] = u8(byte)
	}
	C.arbitrator_add_user_wasm(
		m.ptr,
		(*u8)(arbutil.SliceToPointer(module)),
		usize(len(module)),
		&C.struct_Bytes32{hashBytes},
	)
	return nil
}
