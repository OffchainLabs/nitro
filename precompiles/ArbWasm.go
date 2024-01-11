// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type ArbWasm struct {
	Address addr // 0x71

	ProgramActivated              func(ctx, mech, hash, hash, addr, uint16) error
	ProgramActivatedGasCost       func(hash, hash, addr, uint16) (uint64, error)
	ProgramNotActivatedError      func() error
	ProgramNeedsUpgradeError      func(version, stylusVersion uint16) error
	ProgramExpiredError           func(age uint64) error
	ProgramUpToDateError          func() error
	ProgramKeepaliveTooSoonError  func(age uint64) error
	ProgramInsufficientValueError func(have, want huge) error
}

// Compile a wasm program with the latest instrumentation
func (con ArbWasm) ActivateProgram(c ctx, evm mech, value huge, program addr) (uint16, error) {
	debug := evm.ChainConfig().DebugMode()

	// charge a fixed cost up front to begin activation
	if err := c.Burn(1659168); err != nil {
		return 0, err
	}
	version, codeHash, moduleHash, dataFee, takeAllGas, err := c.State.Programs().ActivateProgram(evm, program, debug)
	if takeAllGas {
		_ = c.BurnOut()
	}
	if err != nil {
		return version, err
	}
	if err := con.payActivationDataFee(c, evm, value, dataFee); err != nil {
		return version, err
	}
	return version, con.ProgramActivated(c, evm, codeHash, moduleHash, program, version)
}

// Pays the data component of activation costs
func (con ArbWasm) payActivationDataFee(c ctx, evm mech, value, dataFee huge) error {
	if arbmath.BigLessThan(value, dataFee) {
		return con.ProgramInsufficientValueError(value, dataFee)
	}
	network, err := c.State.NetworkFeeAccount()
	if err != nil {
		return err
	}
	scenario := util.TracingDuringEVM
	repay := arbmath.BigSub(value, dataFee)

	// transfer the fee to the network account, and the rest back to the user
	err = util.TransferBalance(&con.Address, &network, dataFee, evm, scenario, "activate")
	if err != nil {
		return err
	}
	return util.TransferBalance(&con.Address, &c.caller, repay, evm, scenario, "reimburse")
}

// Gets the latest stylus version
func (con ArbWasm) StylusVersion(c ctx, evm mech) (uint16, error) {
	return c.State.Programs().StylusVersion()
}

// Gets the amount of ink 1 gas buys
func (con ArbWasm) InkPrice(c ctx, _ mech) (uint32, error) {
	ink, err := c.State.Programs().InkPrice()
	return ink.ToUint32(), err
}

// Gets the wasm stack size limit
func (con ArbWasm) MaxStackDepth(c ctx, _ mech) (uint32, error) {
	return c.State.Programs().MaxStackDepth()
}

// Gets the number of free wasm pages a tx gets
func (con ArbWasm) FreePages(c ctx, _ mech) (uint16, error) {
	return c.State.Programs().FreePages()
}

// Gets the base cost of each additional wasm page
func (con ArbWasm) PageGas(c ctx, _ mech) (uint16, error) {
	return c.State.Programs().PageGas()
}

// Gets the ramp that drives exponential memory costs
func (con ArbWasm) PageRamp(c ctx, _ mech) (uint64, error) {
	return c.State.Programs().PageRamp()
}

// Gets the maximum initial number of pages a wasm may allocate
func (con ArbWasm) PageLimit(c ctx, _ mech) (uint16, error) {
	return c.State.Programs().PageLimit()
}

// Gets the stylus version that program with codehash was most recently compiled with
func (con ArbWasm) CodehashVersion(c ctx, evm mech, codehash bytes32) (uint16, error) {
	return c.State.Programs().CodehashVersion(codehash, evm.Context.Time)
}

// Extends a program's expiration date (reverts if too soon)
func (con ArbWasm) CodehashKeepalive(c ctx, evm mech, value huge, codehash bytes32) error {
	dataFee, err := c.State.Programs().ProgramKeepalive(codehash, evm.Context.Time)
	if err != nil {
		return err
	}
	return con.payActivationDataFee(c, evm, value, dataFee)
}

// Gets the stylus version that program at addr was most recently compiled with
func (con ArbWasm) ProgramVersion(c ctx, evm mech, program addr) (uint16, error) {
	codehash, err := c.GetCodeHash(program)
	if err != nil {
		return 0, err
	}
	return con.CodehashVersion(c, evm, codehash)
}

// Gets the cost to invoke the program (not including MinInitGas)
func (con ArbWasm) ProgramInitGas(c ctx, evm mech, program addr) (uint32, error) {
	codehash, err := c.GetCodeHash(program)
	if err != nil {
		return 0, err
	}
	return c.State.Programs().ProgramInitGas(codehash, evm.Context.Time)
}

// Gets the footprint of program at addr
func (con ArbWasm) ProgramMemoryFootprint(c ctx, evm mech, program addr) (uint16, error) {
	codehash, err := c.GetCodeHash(program)
	if err != nil {
		return 0, err
	}
	return c.State.Programs().ProgramMemoryFootprint(codehash, evm.Context.Time)
}

// Gets returns the amount of time remaining until the program expires
func (con ArbWasm) ProgramTimeLeft(c ctx, evm mech, program addr) (uint64, error) {
	codehash, err := c.GetCodeHash(program)
	if err != nil {
		return 0, err
	}
	return c.State.Programs().ProgramTimeLeft(codehash, evm.Context.Time)
}

// Gets the minimum cost to invoke a program
func (con ArbWasm) MinInitGas(c ctx, _ mech) (uint16, error) {
	return c.State.Programs().MinInitGas()
}

// Gets the number of days after which programs deactivate
func (con ArbWasm) ExpiryDays(c ctx, _ mech) (uint16, error) {
	return c.State.Programs().ExpiryDays()
}

// Gets the age a program must be to perform a keepalive
func (con ArbWasm) KeepaliveDays(c ctx, _ mech) (uint16, error) {
	return c.State.Programs().KeepaliveDays()
}
