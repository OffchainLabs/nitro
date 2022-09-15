// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"log"
	"math/big"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/util"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
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
	gasLeft     uint64
	txProcessor *arbos.TxProcessor
	State       *arbosState.ArbosState
	tracingInfo *util.TracingInfo
	readOnly    bool
}

func (c *Context) Burn(amount uint64) error {
	if c.gasLeft < amount {
		c.gasLeft = 0
		return vm.ErrOutOfGas
	}
	c.gasLeft -= amount
	return nil
}

//nolint:unused
func (c *Context) Burned() uint64 {
	return c.gasSupplied - c.gasLeft
}

func (c *Context) Restrict(err error) {
	log.Fatal("A metered burner was used for access-controlled work", err)
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

func testContext(caller addr, evm mech) *Context {
	tracingInfo := util.NewTracingInfo(evm, common.Address{}, types.ArbosAddress, util.TracingDuringEVM)
	ctx := &Context{
		caller:      caller,
		gasSupplied: ^uint64(0),
		gasLeft:     ^uint64(0),
		tracingInfo: tracingInfo,
		readOnly:    false,
	}
	state, err := arbosState.OpenArbosState(evm.StateDB, burn.NewSystemBurner(tracingInfo, false))
	if err != nil {
		panic(err)
	}
	ctx.State = state
	var ok bool
	ctx.txProcessor, ok = evm.ProcessingHook.(*arbos.TxProcessor)
	if !ok {
		panic("must have tx processor")
	}
	return ctx
}
