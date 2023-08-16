// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"math/big"

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
	gasLeft     uint64
	txProcessor *arbos.TxProcessor
	State       *arbosState.ArbosState
	tracingInfo *util.TracingInfo
	readOnly    bool
}

func (c *Context) Burn(amount uint64) error {
	if c.gasLeft < amount {
		return c.BurnOut()
	}
	c.gasLeft -= amount
	return nil
}

//nolint:unused
func (c *Context) Burned() uint64 {
	return c.gasSupplied - c.gasLeft
}

func (c *Context) BurnOut() error {
	c.gasLeft = 0
	return vm.ErrOutOfGas
}

func (c *Context) GasLeft() uint64 {
	return c.gasLeft
}

func (c *Context) Restrict(err error) {
	log.Crit("A metered burner was used for access-controlled work", "error", err)
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
		log.Crit("unable to open arbos state", "error", err)
	}
	ctx.State = state
	var ok bool
	ctx.txProcessor, ok = evm.ProcessingHook.(*arbos.TxProcessor)
	if !ok {
		log.Crit("must have tx processor")
	}
	return ctx
}
