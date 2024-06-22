// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type ArbWasm struct {
	Address addr // 0x71

	ProgramActivated               func(ctx, mech, hash, hash, addr, huge, uint16) error
	ProgramActivatedGasCost        func(hash, hash, addr, huge, uint16) (uint64, error)
	ProgramLifetimeExtended        func(ctx, mech, hash, huge) error
	ProgramLifetimeExtendedGasCost func(hash, huge) (uint64, error)

	ProgramNotWasmError           func() error
	ProgramNotActivatedError      func() error
	ProgramNeedsUpgradeError      func(version, stylusVersion uint16) error
	ProgramExpiredError           func(age uint64) error
	ProgramUpToDateError          func() error
	ProgramKeepaliveTooSoonError  func(age uint64) error
	ProgramInsufficientValueError func(have, want huge) error
}

// Compile a wasm program with the latest instrumentation
func (con ArbWasm) ActivateProgram(c ctx, evm mech, value huge, program addr) (uint16, huge, error) {
	debug := evm.ChainConfig().DebugMode()
	runMode := c.txProcessor.RunMode()
	programs := c.State.Programs()

	// charge a fixed cost up front to begin activation
	if err := c.Burn(1659168); err != nil {
		return 0, nil, err
	}
	version, codeHash, moduleHash, dataFee, takeAllGas, err := programs.ActivateProgram(evm, program, runMode, debug)
	if takeAllGas {
		_ = c.BurnOut()
	}
	if err != nil {
		return version, dataFee, err
	}
	if err := con.payActivationDataFee(c, evm, value, dataFee); err != nil {
		return version, dataFee, err
	}
	return version, dataFee, con.ProgramActivated(c, evm, codeHash, moduleHash, program, dataFee, version)
}

// Extends a program's expiration date (reverts if too soon)
func (con ArbWasm) CodehashKeepalive(c ctx, evm mech, value huge, codehash bytes32) error {
	params, err := c.State.Programs().Params()
	if err != nil {
		return err
	}
	dataFee, err := c.State.Programs().ProgramKeepalive(codehash, evm.Context.Time, params)
	if err != nil {
		return err
	}
	if err := con.payActivationDataFee(c, evm, value, dataFee); err != nil {
		return err
	}
	return con.ProgramLifetimeExtended(c, evm, codehash, dataFee)
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
	params, err := c.State.Programs().Params()
	return params.Version, err
}

// Gets the amount of ink 1 gas buys
func (con ArbWasm) InkPrice(c ctx, _ mech) (uint32, error) {
	params, err := c.State.Programs().Params()
	return params.InkPrice.ToUint32(), err
}

// Gets the wasm stack size limit
func (con ArbWasm) MaxStackDepth(c ctx, _ mech) (uint32, error) {
	params, err := c.State.Programs().Params()
	return params.MaxStackDepth, err
}

// Gets the number of free wasm pages a tx gets
func (con ArbWasm) FreePages(c ctx, _ mech) (uint16, error) {
	params, err := c.State.Programs().Params()
	return params.FreePages, err
}

// Gets the base cost of each additional wasm page
func (con ArbWasm) PageGas(c ctx, _ mech) (uint16, error) {
	params, err := c.State.Programs().Params()
	return params.PageGas, err
}

// Gets the ramp that drives exponential memory costs
func (con ArbWasm) PageRamp(c ctx, _ mech) (uint64, error) {
	params, err := c.State.Programs().Params()
	return params.PageRamp, err
}

// Gets the maximum initial number of pages a wasm may allocate
func (con ArbWasm) PageLimit(c ctx, _ mech) (uint16, error) {
	params, err := c.State.Programs().Params()
	return params.PageLimit, err
}

// Gets the minimum costs to invoke a program
func (con ArbWasm) MinInitGas(c ctx, _ mech) (uint64, uint64, error) {
	params, err := c.State.Programs().Params()
	init := uint64(params.MinInitGas) * programs.MinInitGasUnits
	cached := uint64(params.MinCachedInitGas) * programs.MinCachedGasUnits
	return init, cached, err
}

// Gets the linear adjustment made to program init costs
func (con ArbWasm) InitCostScalar(c ctx, _ mech) (uint64, error) {
	params, err := c.State.Programs().Params()
	return uint64(params.InitCostScalar) * programs.CostScalarPercent, err
}

// Gets the number of days after which programs deactivate
func (con ArbWasm) ExpiryDays(c ctx, _ mech) (uint16, error) {
	params, err := c.State.Programs().Params()
	return params.ExpiryDays, err
}

// Gets the age a program must be to perform a keepalive
func (con ArbWasm) KeepaliveDays(c ctx, _ mech) (uint16, error) {
	params, err := c.State.Programs().Params()
	return params.KeepaliveDays, err
}

// Gets the number of extra programs ArbOS caches during a given block.
func (con ArbWasm) BlockCacheSize(c ctx, _ mech) (uint16, error) {
	params, err := c.State.Programs().Params()
	return params.BlockCacheSize, err
}

// Gets the stylus version that program with codehash was most recently compiled with
func (con ArbWasm) CodehashVersion(c ctx, evm mech, codehash bytes32) (uint16, error) {
	params, err := c.State.Programs().Params()
	if err != nil {
		return 0, err
	}
	return c.State.Programs().CodehashVersion(codehash, evm.Context.Time, params)
}

// Gets a program's asm size in bytes
func (con ArbWasm) CodehashAsmSize(c ctx, evm mech, codehash bytes32) (uint32, error) {
	params, err := c.State.Programs().Params()
	if err != nil {
		return 0, err
	}
	return c.State.Programs().ProgramAsmSize(codehash, evm.Context.Time, params)
}

// Gets the stylus version that program at addr was most recently compiled with
func (con ArbWasm) ProgramVersion(c ctx, evm mech, program addr) (uint16, error) {
	codehash, err := c.GetCodeHash(program)
	if err != nil {
		return 0, err
	}
	return con.CodehashVersion(c, evm, codehash)
}

// Gets the cost to invoke the program
func (con ArbWasm) ProgramInitGas(c ctx, evm mech, program addr) (uint64, uint64, error) {
	codehash, params, err := con.getCodeHash(c, program)
	if err != nil {
		return 0, 0, err
	}
	return c.State.Programs().ProgramInitGas(codehash, evm.Context.Time, params, c.State.ArbOSVersion())
}

// Gets the footprint of program at addr
func (con ArbWasm) ProgramMemoryFootprint(c ctx, evm mech, program addr) (uint16, error) {
	codehash, params, err := con.getCodeHash(c, program)
	if err != nil {
		return 0, err
	}
	return c.State.Programs().ProgramMemoryFootprint(codehash, evm.Context.Time, params)
}

// Gets returns the amount of time remaining until the program expires
func (con ArbWasm) ProgramTimeLeft(c ctx, evm mech, program addr) (uint64, error) {
	codehash, params, err := con.getCodeHash(c, program)
	if err != nil {
		return 0, err
	}
	return c.State.Programs().ProgramTimeLeft(codehash, evm.Context.Time, params)
}

func (con ArbWasm) getCodeHash(c ctx, program addr) (hash, *programs.StylusParams, error) {
	params, err := c.State.Programs().Params()
	if err != nil {
		return common.Hash{}, params, err
	}
	codehash, err := c.GetCodeHash(program)
	return codehash, params, err
}
