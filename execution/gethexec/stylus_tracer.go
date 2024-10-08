// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package gethexec

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/containers"
)

func init() {
	tracers.DefaultDirectory.Register("stylusTracer", newStylusTracer, false)
}

// stylusTracer captures Stylus HostIOs and returns them in a structured format to be used in Cargo
// Stylus Replay.
type stylusTracer struct {
	open      *containers.Stack[HostioTraceInfo]
	stack     *containers.Stack[*containers.Stack[HostioTraceInfo]]
	interrupt atomic.Bool
	reason    error
}

// HostioTraceInfo contains the captured HostIO log returned by stylusTracer.
type HostioTraceInfo struct {
	// Name of the HostIO.
	Name string `json:"name"`

	// Arguments of the HostIO encoded as binary.
	// For details about the encoding check the HostIO implemenation on
	// arbitrator/wasm-libraries/user-host-trait.
	Args hexutil.Bytes `json:"args"`

	// Outputs of the HostIO encoded as binary.
	// For details about the encoding check the HostIO implemenation on
	// arbitrator/wasm-libraries/user-host-trait.
	Outs hexutil.Bytes `json:"outs"`

	// Amount of Ink before executing the HostIO.
	StartInk uint64 `json:"startInk"`

	// Amount of Ink after executing the HostIO.
	EndInk uint64 `json:"endInk"`

	// For *call HostIOs, the address of the called contract.
	Address *common.Address `json:"address,omitempty"`

	// For *call HostIOs, the steps performed by the called contract.
	Steps *containers.Stack[HostioTraceInfo] `json:"steps,omitempty"`
}

// nestsHostios contains the hostios with nested calls.
var nestsHostios = map[string]bool{
	"call_contract":          true,
	"delegate_call_contract": true,
	"static_call_contract":   true,
}

func newStylusTracer(ctx *tracers.Context, _ json.RawMessage) (tracers.Tracer, error) {
	return &stylusTracer{
		open:  containers.NewStack[HostioTraceInfo](),
		stack: containers.NewStack[*containers.Stack[HostioTraceInfo]](),
	}, nil
}

func (t *stylusTracer) CaptureStylusHostio(name string, args, outs []byte, startInk, endInk uint64) {
	if t.interrupt.Load() {
		return
	}
	info := HostioTraceInfo{
		Name:     name,
		Args:     args,
		Outs:     outs,
		StartInk: startInk,
		EndInk:   endInk,
	}
	if nestsHostios[name] {
		last, err := t.open.Pop()
		if err != nil {
			t.Stop(err)
			return
		}
		if !strings.HasPrefix(last.Name, "evm_") || last.Name[4:] != info.Name {
			t.Stop(fmt.Errorf("trace inconsistency for %v: last opcode is %v", info.Name, last.Name))
			return
		}
		if last.Steps == nil {
			t.Stop(fmt.Errorf("trace inconsistency for %v: nil steps", info.Name))
			return
		}
		info.Address = last.Address
		info.Steps = last.Steps
	}
	t.open.Push(info)
}

func (t *stylusTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	if t.interrupt.Load() {
		return
	}

	// This function adds the prefix evm_ because it assumes the opcode came from the EVM.
	// If the opcode comes from WASM, the CaptureStylusHostio function will remove the evm prefix.
	var name string
	switch typ {
	case vm.CALL:
		name = "evm_call_contract"
	case vm.DELEGATECALL:
		name = "evm_delegate_call_contract"
	case vm.STATICCALL:
		name = "evm_static_call_contract"
	case vm.CREATE:
		name = "evm_create1"
	case vm.CREATE2:
		name = "evm_create2"
	case vm.SELFDESTRUCT:
		name = "evm_self_destruct"
	}

	inner := containers.NewStack[HostioTraceInfo]()
	info := HostioTraceInfo{
		Name:    name,
		Address: &to,
		Steps:   inner,
	}
	t.open.Push(info)
	t.stack.Push(t.open)
	t.open = inner
}

func (t *stylusTracer) CaptureExit(output []byte, gasUsed uint64, _ error) {
	if t.interrupt.Load() {
		return
	}
	var err error
	t.open, err = t.stack.Pop()
	if err != nil {
		t.Stop(err)
	}
}

func (t *stylusTracer) GetResult() (json.RawMessage, error) {
	if t.reason != nil {
		return nil, t.reason
	}

	var internalErr error
	if t.open == nil {
		internalErr = errors.Join(internalErr, fmt.Errorf("tracer.open is nil"))
	}
	if t.stack == nil {
		internalErr = errors.Join(internalErr, fmt.Errorf("tracer.stack is nil"))
	}
	if !t.stack.Empty() {
		internalErr = errors.Join(internalErr, fmt.Errorf("tracer.stack should be empty, but has %d values", t.stack.Len()))
	}
	if internalErr != nil {
		log.Error("stylusTracer: internal error when generating a trace", "error", internalErr)
		return nil, fmt.Errorf("internal error: %w", internalErr)
	}

	msg, err := json.Marshal(t.open)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (t *stylusTracer) Stop(err error) {
	t.reason = err
	t.interrupt.Store(true)
}

// Unimplemented EVMLogger interface methods

func (t *stylusTracer) CaptureArbitrumTransfer(env *vm.EVM, from, to *common.Address, value *big.Int, before bool, purpose string) {
}
func (t *stylusTracer) CaptureArbitrumStorageGet(key common.Hash, depth int, before bool)        {}
func (t *stylusTracer) CaptureArbitrumStorageSet(key, value common.Hash, depth int, before bool) {}
func (t *stylusTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
}
func (t *stylusTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {}
func (t *stylusTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
}
func (t *stylusTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}
func (t *stylusTracer) CaptureTxStart(gasLimit uint64) {}
func (t *stylusTracer) CaptureTxEnd(restGas uint64)    {}
