// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"math/big"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/util"
)

type addr = common.Address
type mech = *vm.EVM
type huge = *big.Int
type hash = common.Hash
type bytes4 = [4]byte
type bytes32 = [32]byte
type ctx = *Context

type Context struct {
	caller      addr
	gasSupplied uint64
	gasUsed     multigas.MultiGas
	txProcessor *arbos.TxProcessor
	State       *arbosState.ArbosState
	tracingInfo *util.TracingInfo
	readOnly    bool
}

func (c *Context) Burn(kind multigas.ResourceKind, amount uint64) error {
	if c.GasLeft() < amount {
		return c.BurnOut()
	}
	c.gasUsed.SaturatingIncrementInto(kind, amount)
	return nil
}

//nolint:unused
func (c *Context) Burned() uint64 {
	return c.gasUsed.SingleGas()
}

func (c *Context) BurnOut() error {
	c.gasUsed.SaturatingIncrementInto(multigas.ResourceKindComputation, c.GasLeft())
	return vm.ErrOutOfGas
}

func (c *Context) GasLeft() uint64 {
	return c.gasSupplied - c.gasUsed.SingleGas()
}

func (c *Context) Restrict(err error) {
	panic("A metered burner was used for access-controlled work :" + err.Error())
}

func (c *Context) HandleError(err error) error {
	return err
}

func (c *Context) ReadOnly() bool {
	return c.readOnly
}

func (c *Context) TracingInfo() *util.TracingInfo {
	return c.tracingInfo
}

func (c *Context) GetCodeHash(address common.Address) (common.Hash, error) {
	return c.State.BackingStorage().GetCodeHash(address)
}

func testContext(caller addr, evm mech) *Context {
	tracingInfo := util.NewTracingInfo(evm, common.Address{}, types.ArbosAddress, util.TracingDuringEVM)
	ctx := &Context{
		caller:      caller,
		gasSupplied: ^uint64(0),
		gasUsed:     multigas.ZeroGas(),
		tracingInfo: tracingInfo,
		readOnly:    false,
	}
	state, err := arbosState.OpenArbosState(evm.StateDB, burn.NewSystemBurner(tracingInfo, false))
	if err != nil {
		panic("unable to open arbos state :" + err.Error())
	}
	ctx.State = state
	var ok bool
	ctx.txProcessor, ok = evm.ProcessingHook.(*arbos.TxProcessor)
	if !ok {
		panic("must have tx processor")
	}
	return ctx
}

func makeContext(p *Precompile, method *PrecompileMethod, caller common.Address, gas uint64, evm *vm.EVM) (*Context, error) {
	txProcessor, ok := evm.ProcessingHook.(*arbos.TxProcessor)
	if !ok {
		log.Error("processing hook not set")
		return nil, vm.ErrExecutionReverted
	}

	readOnly := method.purity <= view

	callerCtx := &Context{
		caller:      caller,
		gasSupplied: gas,
		gasUsed:     multigas.ZeroGas(),
		readOnly:    readOnly,
		txProcessor: txProcessor,
		tracingInfo: util.NewTracingInfo(evm, caller, p.address, util.TracingDuringEVM),
	}

	if method.purity != pure {
		state, err := arbosState.OpenArbosState(evm.StateDB, callerCtx)
		if err != nil {
			return nil, err
		}
		callerCtx.State = state
	}

	if method.purity >= write && evm.ReadOnly() {
		toBurn, err := callerCtx.State.L2PricingState().PerTxGasLimit()
		if err != nil {
			return nil, err
		}
		err = callerCtx.Burn(multigas.ResourceKindComputation, toBurn)
		if err != nil {
			return nil, err
		}
	}

	return callerCtx, nil
}
