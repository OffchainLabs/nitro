// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

type ArbWasm struct {
	Address addr // 0x71

	ProgramNotCompiledError func() error
	ProgramOutOfDateError   func(version uint32) error
	ProgramUpToDateError    func() error
}

// Compile a wasm program with the latest instrumentation
func (con ArbWasm) CompileProgram(c ctx, evm mech, program addr) (uint32, error) {
	// TODO: pay for gas by some compilation pricing formula
	return c.State.Programs().CompileProgram(evm.StateDB, program, evm.ChainConfig().DebugMode())
}

// Gets the latest stylus version
func (con ArbWasm) StylusVersion(c ctx, _ mech) (uint32, error) {
	return c.State.Programs().StylusVersion()
}

// Gets the price (in evm gas basis points) of ink
func (con ArbWasm) InkPrice(c ctx, _ mech) (uint64, error) {
	bips, err := c.State.Programs().InkPrice()
	return bips.Uint64(), err
}

// Gets the wasm stack size limit
func (con ArbWasm) WasmMaxDepth(c ctx, _ mech) (uint32, error) {
	return c.State.Programs().WasmMaxDepth()
}

// Gets the cost of starting a stylus hostio call
func (con ArbWasm) WasmHostioInk(c ctx, _ mech) (uint64, error) {
	return c.State.Programs().WasmHostioInk()
}

// Gets the current program version
func (con ArbWasm) ProgramVersion(c ctx, evm mech, program addr) (uint32, error) {
	return c.State.Programs().ProgramVersion(evm.StateDB, program)
}
