// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

type ArbWasm struct {
	Address addr // 0x71

	ProgramActivated             func(ctx, mech, hash, hash, addr, uint16) error
	ProgramActivatedGasCost      func(hash, hash, addr, uint16) (uint64, error)
	ProgramNotActivatedError     func() error
	ProgramNeedsUpgradeError     func(version, stylusVersion uint16) error
	ProgramExpiredError          func(age uint64) error
	ProgramUpToDateError         func() error
	ProgramKeepaliveTooSoonError func(age uint64) error
}

// Compile a wasm program with the latest instrumentation
func (con ArbWasm) ActivateProgram(c ctx, evm mech, program addr) (uint16, error) {
	debug := evm.ChainConfig().DebugMode()

	// charge 3 million up front to begin activation
	if err := c.Burn(3000000); err != nil {
		return 0, err
	}
	version, codeHash, moduleHash, takeAllGas, err := c.State.Programs().ActivateProgram(evm, program, debug)
	if takeAllGas {
		_ = c.BurnOut()
	}
	if err != nil {
		return version, err
	}
	return version, con.ProgramActivated(c, evm, codeHash, moduleHash, program, version)
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

// @notice extends a program's expiration date (reverts if too soon)
func (con ArbWasm) CodehashKeepalive(c ctx, evm mech, codehash bytes32) error {
	return c.State.Programs().ProgramKeepalive(codehash, evm.Context.Time)
}

// Gets the stylus version that program at addr was most recently compiled with
func (con ArbWasm) ProgramVersion(c ctx, evm mech, program addr) (uint16, error) {
	codehash, err := c.GetCodeHash(program)
	if err != nil {
		return 0, err
	}
	return con.CodehashVersion(c, evm, codehash)
}

// Gets the uncompressed size of program at addr
func (con ArbWasm) ProgramSize(c ctx, evm mech, program addr) (uint32, error) {
	codehash, err := c.GetCodeHash(program)
	if err != nil {
		return 0, err
	}
	return c.State.Programs().ProgramSize(codehash, evm.Context.Time)
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

// Gets the added wasm call cost paid per half kb uncompressed wasm
func (con ArbWasm) CallScalar(c ctx, _ mech) (uint16, error) {
	return c.State.Programs().CallScalar()
}

// Gets the number of days after which programs deactivate
func (con ArbWasm) ExpiryDays(c ctx, _ mech) (uint16, error) {
	return c.State.Programs().ExpiryDays()
}

// Gets the age a program must be to perform a keepalive
func (con ArbWasm) KeepaliveDays(c ctx, _ mech) (uint16, error) {
	return c.State.Programs().KeepaliveDays()
}
