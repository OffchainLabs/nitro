//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

// All calls to this precompile are authorized by the DebugPrecompile wrapper,
// which ensures these methods are not accessible in production.
type ArbDebug struct {
	Address      addr
	Basic        func(mech, bool, [32]byte)                     // index'd: 2nd
	Mixed        func(mech, bool, bool, [32]byte, addr, addr)   // index'd: 1st 3rd 5th
	Store        func(mech, bool, addr, huge, [32]byte, []byte) // index'd: 1st 2nd
	BasicGasCost func(bool, [32]byte) uint64
	MixedGasCost func(bool, bool, [32]byte, addr, addr) uint64
	StoreGasCost func(bool, addr, huge, [32]byte, []byte) uint64
}

func (con ArbDebug) Events(c ctx, evm mech, paid huge, flag bool, value [32]byte) (addr, huge, error) {
	// Emits 2 events that cover each case
	//   Basic tests an index'd value & a normal value
	//   Mixed interleaves index'd and normal values that may need to be padded

	cost := con.BasicGasCost(true, value) + con.MixedGasCost(true, true, value, c.caller, c.caller)
	if err := c.burn(cost); err != nil {
		return c.caller, paid, err
	}

	con.Basic(evm, !flag, value)
	con.Mixed(evm, flag, !flag, value, con.Address, c.caller)

	return c.caller, paid, nil
}

func (con ArbDebug) BecomeChainOwner(c ctx, evm mech) error {
	return c.state.ChainOwners().Add(c.caller)
}

func (con ArbDebug) GetL2GasPrice(c ctx, evm mech) (huge, error) {
	return c.state.GasPriceWei()
}
