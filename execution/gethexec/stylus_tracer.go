// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package gethexec

import (
	"encoding/json"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/offchainlabs/nitro/util/stack"
)

func init() {
	tracers.DefaultDirectory.Register("stylusTracer", newStylusTracer, false)
}

// stylusTracer captures Stylus HostIOs and returns them in a structured format to be used in Cargo
// Stylus Replay.
type stylusTracer struct {
	open      *stack.Stack[HostioTraceInfo]
	stack     *stack.Stack[*stack.Stack[HostioTraceInfo]]
	interrupt atomic.Bool
	reason    error
}

// HostioTraceInfo contains the captured HostIO log returned by stylusTracer.
type HostioTraceInfo struct {
	Name     string                        `json:"name"`
	Args     hexutil.Bytes                 `json:"args"`
	Outs     hexutil.Bytes                 `json:"outs"`
	StartInk uint64                        `json:"startInk"`
	EndInk   uint64                        `json:"endInk"`
	Address  *common.Address               `json:"address,omitempty"`
	Steps    *stack.Stack[HostioTraceInfo] `json:"steps,omitempty"`
}

// nestsHostios contains the hostios with nested calls.
var nestsHostios = map[string]bool{
	"call_contract":          true,
	"delegate_call_contract": true,
	"static_call_contract":   true,
}

func newStylusTracer(ctx *tracers.Context, _ json.RawMessage) (tracers.Tracer, error) {
	return &stylusTracer{
		open:  stack.NewStack[HostioTraceInfo](),
		stack: stack.NewStack[*stack.Stack[HostioTraceInfo]](),
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
		info.Address = last.Address
		info.Steps = last.Steps
	}
	t.open.Push(info)
}

func (t *stylusTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	if t.interrupt.Load() {
		return
	}
	inner := stack.NewStack[HostioTraceInfo]()
	info := HostioTraceInfo{
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
