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
	version     uint64 // set during OpenArbosState
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

func (c *Context) RequireGas(amount uint64) error {
	if c.gasLeft < amount {
		c.gasLeft = 0
		return vm.ErrOutOfGas
	}
	return nil
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

func (c *Context) OutsideTx() bool {
	return false
}

func (c *Context) TracingInfo() *util.TracingInfo {
	return c.tracingInfo
}

func (c *Context) ChargeWithVersion(temporaryVersion uint64, closure func() error) error {
	current := c.version
	c.version = temporaryVersion
	err := closure()
	c.version = current
	return err
}

func (c *Context) Version() uint64 {
	return c.version
}

func (c *Context) SetVersion(version uint64) {
	c.version = version
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
